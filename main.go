package main

import (
	"fmt"
	"math/rand"
	"net/url"

	"github.com/ChimeraCoder/anaconda"
	// Imports the Google Cloud Natural Language API client package.
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	language "cloud.google.com/go/language/apiv1"
	"golang.org/x/net/context"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	// Imports the Google Cloud Storage client package.
	"os"

	"trumpmeltdown/phrases"

	"flag"

	"cloud.google.com/go/storage"

	"database/sql"

	"unicode"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/sqltocsv"
)

type TweetSentiment struct {
	Time          int64
	Tweets        []Tweet
	Average       float32
	NumTweets     int
	NextUpdate    int64
	LastSentTweet string
}

type Tweet struct {
	Text      string
	Sentiment float32
	Id        string
	EmbedHTML string
}

var numTweets = 10

func main() {
	var testing = flag.Bool("testing", false, "enable testing mode. No actual tweeting or API calls.")
	var machineLearning = flag.Bool("machinelearning", false, "run machine learning logic.")
	flag.Parse()

	now := time.Now()
	rand.Seed(now.UTC().UnixNano())

	currentTimestamp := now.Unix()
	nextUpdate := now.Add(5 * time.Second).Unix()

	consumerKey := os.Getenv("TRUMPMELTDOWN_CONSUMER_KEY")
	consumerSecret := os.Getenv("TRUMPMELTDOWN_CONSUMER_SECRET")
	accessToken := os.Getenv("TRUMPMELTDOWN_ACCESS_TOKEN")
	accessSecret := os.Getenv("TRUMPMELTDOWN_ACCESS_SECRET")

	DB_USER := os.Getenv("TRUMPMELTDOWN_DBUSER")
	DB_PASS := os.Getenv("TRUMPMELTDOWN_DBPASS")
	DB_HOST := os.Getenv("DBHOST")

	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(accessToken, accessSecret)

	dsn := DB_USER + ":" + DB_PASS + "@tcp(" + DB_HOST + ":3306)/trumpmeltdown?parseTime=true"

	ctx := context.Background()
	client, err := language.NewClient(ctx)
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
	}
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
	}
	bucketName := os.Getenv("TRUMPMELTDOWN_SENTIMENT_BUCKET")
	bucket := storageClient.Bucket(bucketName)

	latestContents, err := ioutil.ReadFile("latest")
	if err != nil {
		fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketName, "latest", err)
		latestReader, err := bucket.Object("latest").NewReader(ctx)
		if err != nil {
			fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketName, "latest", err)
			return
		}
		defer latestReader.Close()
		latestContents, err = ioutil.ReadAll(latestReader)
		if err != nil {
			fmt.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketName, "latest", err)
			return
		}
	}

	var last TweetSentiment
	json.Unmarshal(latestContents, &last)
	latestTweet := last.Tweets[0]

	if *testing {
		latestTweet = last.Tweets[len(last.Tweets)-1]
	}

	values := url.Values{}
	values.Set("screen_name", "realdonaldtrump")
	values.Set("count", fmt.Sprintf("%d", numTweets))
	values.Set("since_id", fmt.Sprintf("%s", latestTweet.Id))

	tweetsResponse, err := api.GetUserTimeline(values)
	if err != nil {
		fmt.Println(err)
	}

	var Tweets []Tweet
	var totalSentiment float32

	numNewTweets := len(tweetsResponse)
	fmt.Printf("New Tweets: %d\n", len(tweetsResponse))

	var tweetsToSend []Tweet

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer db.Close()
	// make sure our connection is available
	err = db.Ping()
	if err != nil {
		fmt.Print(err.Error())
	}

	var stmt *sql.Stmt

	for i, tweet := range tweetsResponse {
		// Get Embed HTML for Tweet
		values := url.Values{}
		values.Set("omit_script", "true")
		values.Set("align", "center")
		values.Set("related", "istrumpmeltdown")

		oembed, err := api.GetOEmbedId(tweet.Id, values)
		if err != nil {
			fmt.Printf("err: %s", err)
		}
		if !*testing {
			sentiment, err := client.AnalyzeSentiment(ctx, &languagepb.AnalyzeSentimentRequest{
				Document: &languagepb.Document{
					Source: &languagepb.Document_Content{
						Content: tweet.FullText,
					},
					Type: languagepb.Document_PLAIN_TEXT,
				},
				EncodingType: languagepb.EncodingType_UTF8,
			})
			if err != nil {
				log.Fatalf("Failed to analyze text: %v", err)
			}

			fmt.Printf("%d. Text: %v\n", i, tweet.FullText)
			if sentiment.DocumentSentiment.Score >= 0 {
				fmt.Println("Sentiment: positive")
			} else {
				fmt.Println("Sentiment: negative")
			}

			Tweets = append(Tweets, Tweet{tweet.FullText, sentiment.DocumentSentiment.Score, fmt.Sprintf("%d", tweet.Id), oembed.Html})
			tweetsToSend = append(tweetsToSend, Tweet{tweet.FullText, sentiment.DocumentSentiment.Score, fmt.Sprintf("%d", tweet.Id), oembed.Html})
			stmt, err = db.Prepare("insert into tweets (sentiment, time_of_day, time_since_last_tweet, caps_percentage, length, grammar_mistake_count, isRetweet, tweet_id, tweet_text, html) values(?,?,?,?,?,?,?,?,?,?);")
			if err != nil {
				fmt.Print(err.Error())
			}
			_, err = stmt.Exec(sentimentToMeltdown(sentiment.DocumentSentiment.Score), nil, nil, calculateCapsPercentage(tweet.FullText), len(tweet.FullText), 0, 0, tweet.Id, tweet.FullText, oembed.Html)

			if err != nil {
				fmt.Print(err.Error())
			}

			defer stmt.Close()
		} else {
			randSentiment := ((rand.Float32() * 2) - 1)
			Tweets = append(Tweets, Tweet{tweet.FullText, randSentiment, fmt.Sprintf("%d", tweet.Id), oembed.Html})
			tweetsToSend = append(tweetsToSend, Tweet{tweet.FullText, randSentiment, fmt.Sprintf("%d", tweet.Id), oembed.Html})
		}

	}

	fmt.Printf("Number of tweets in latest file: %d\n", len(last.Tweets))
	fmt.Printf("Tweets contains %d, adding %d more...\n", len(Tweets), (numTweets - len(Tweets)))

	for i := 0; (len(Tweets) < numTweets) && (i <= last.NumTweets); i++ {
		Tweets = append(Tweets, Tweet{last.Tweets[i].Text, last.Tweets[i].Sentiment, last.Tweets[i].Id, last.Tweets[i].EmbedHTML})
	}

	for _, tweet := range Tweets {
		totalSentiment += tweet.Sentiment
	}

	average := totalSentiment / float32(len(Tweets))
	numTweets = len(Tweets)
	filename := fmt.Sprintf("%d", currentTimestamp)
	responseJson := TweetSentiment{currentTimestamp, Tweets, average, numTweets, nextUpdate, ""}

	trendingTowardMeltdown := last.Average > average
	for _, tweet := range tweetsToSend {
		statusText := phrases.GetIntroPhrase(sentimentToMeltdown(tweet.Sentiment))
		// TODO: Randomize this statement. Maybe use emojis and stuff.
		statusText += " "
		statusText += " "

		if average >= 0 {
			// Trump Average is not melting down
			statusText += "#Trump is not currently melting down "
			//Todo: insert random adjectives between is and trending
			if trendingTowardMeltdown {
				statusText += "but is trending toward a meltdown!"
			}
		} else {
			statusText += "#TRUMP IS CURRENTLY MELTING DOWN"
		}

		statusTextFinal := fmt.Sprintf("@realDonaldTrump %s\nCheck it out here: https://isTrumpMeltingDown.com", statusText)

		values := url.Values{}
		values.Set("in_reply_to_status_id", fmt.Sprintf("%s", tweet.Id))
		values.Set("auto_populate_reply_metadata", "true")

		if !*testing {
			fmt.Printf("Posting tweet response to tweet ID %d\n", tweet.Id)
			response, err := api.PostTweet(statusTextFinal, values)
			if err != nil {
				fmt.Println(err)
			}
			responseJson.LastSentTweet = fmt.Sprintf("%s", response.Id)
			fmt.Printf("Sent Tweet ID: %d\n", responseJson.LastSentTweet)
		} else {
			fmt.Printf("Tweet: %s\n\n", statusTextFinal)
		}

	}

	jsonString, err := json.MarshalIndent(responseJson, "", "    ")

	if !*testing {
		ioutil.WriteFile(filename, jsonString, 0644)
		ioutil.WriteFile("latest", jsonString, 0644)
	} else {
		ioutil.WriteFile("testjson", jsonString, 0644)
	}

	if numNewTweets > 0 && !*testing {
		fmt.Printf("New tweets exist. Publishing new file to bucket.\n")
		upload := bucket.Object(filename).NewWriter(ctx)
		upload.ContentType = "application/json"
		upload.CacheControl = "public, max-age=60"
		if _, err := upload.Write(jsonString); err != nil {
			fmt.Printf("createFile: unable to write data to bucket %q, file %q: %v", bucketName, filename, err)
			return
		}
		if err := upload.Close(); err != nil {
			fmt.Printf("createFile: unable to close bucket %q, file %q: %v", bucketName, filename, err)
			return
		}

		src := storageClient.Bucket(bucketName).Object(filename)
		dst := storageClient.Bucket(bucketName).Object("latest")

		fmt.Println("Copying file...")
		_, err = dst.CopierFrom(src).Run(ctx)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Setting permissions...")
		acl := storageClient.Bucket(bucketName).Object("latest").ACL()
		if err := acl.Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			log.Fatal(err)
		}

	}

	os.Remove(filename)

	if *machineLearning {
		fmt.Printf("\n===Running Machine learning logic\n===\n")

		//Get all rows in the database
		rows, _ := db.Query("SELECT id, sentiment, caps_percentage, length, not_meltdown_votes, meltdown_votes FROM tweets")
		_, err := os.Create("tweet_data.csv")
		if err != nil {
			panic(err)
		}

		err = sqltocsv.WriteFile("tweet_data.csv", rows)
		if err != nil {
			panic(err)
		}
	}
}

func sentimentToMeltdown(sentiment float32) int {
	return int(100 - int64((sentiment+1)*50))
}

func calculateCapsPercentage(tweetText string) int {
	totalChars := 0
	totalCapitals := 0

	for _, char := range tweetText {
		if unicode.IsLetter(char) {
			if unicode.IsUpper(char) {
				totalCapitals++
			}
			totalChars++
		}
	}

	return int((float32(totalCapitals) / float32(totalChars)) * 100)
}
