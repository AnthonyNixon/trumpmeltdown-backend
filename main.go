package main

import (
	"github.com/ChimeraCoder/anaconda"
	"fmt"
	"net/url"
	// Imports the Google Cloud Natural Language API client package.
	language "cloud.google.com/go/language/apiv1"
	"golang.org/x/net/context"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
	"log"
	"encoding/json"
	"time"
	"io/ioutil"
	// Imports the Google Cloud Storage client package.
	"cloud.google.com/go/storage"
	"os"
)

type TweetSentiment struct {
	Time int64
	Tweets []Tweet
	Average float32
	NumTweets int
	NextUpdate int64
}

type Tweet struct {
	Text string
	Sentiment float32
	Id int64

}

var numTweets = 10



func main() {

	now := time.Now()

	currentTimestamp := now.Unix()
	nextUpdate := now.Add(1 * time.Hour).Unix()

	consumerKey := os.Getenv("TRUMPMELTDOWN_CONSUMER_KEY")
	consumerSecret := os.Getenv("TRUMPMELTDOWN_CONSUMER_SECRET")
	accessToken := os.Getenv("TRUMPMELTDOWN_ACCESS_TOKEN")
	accessSecret := os.Getenv("TRUMPMELTDOWN_ACCESS_SECRET")

	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(accessToken, accessSecret)

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

	latestReader, err := bucket.Object("latest").NewReader(ctx)
	if err != nil {
		fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketName, "latest", err)
		return
	}
	defer latestReader.Close()
	latestContents, err := ioutil.ReadAll(latestReader)
	if err != nil {
		fmt.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketName, "latest", err)
		return
	}

	var last TweetSentiment
	json.Unmarshal(latestContents, &last)
	latestTweet := last.Tweets[0]

	values := url.Values{}
	values.Set("screen_name", "realdonaldtrump")
	values.Set("count", fmt.Sprintf("%d", numTweets))
	values.Set("since_id", fmt.Sprintf("%d", latestTweet.Id))

	tweetsResponse, err := api.GetUserTimeline(values)
	if err != nil {
		fmt.Println(err)
	}


	var Tweets []Tweet
	var totalSentiment float32

	fmt.Printf("New Tweets: %d\n", len(tweetsResponse))

	for i, tweet := range tweetsResponse {
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
		Tweets = append(Tweets, Tweet{tweet.FullText, sentiment.DocumentSentiment.Score, tweet.Id})

		fmt.Printf("Posting tweet response to tweet ID %d\n", tweet.Id)

		statusText := fmt.Sprintf("This Tweet scored %d%% meltdown. #Trump is currently ", (100 - int64((sentiment.DocumentSentiment.Score + 1) * 50)))
		if sentiment.DocumentSentiment.Score >= 0 {
			statusText += "not "
		}

		statusText += "melting down!"

		statusTextFinal := fmt.Sprintf("@realDonaldTrump %s\nCheck it out here: http://www.isTrumpMeltingDown.com", statusText)


		values := url.Values{}
		values.Set("in_reply_to_status_id", fmt.Sprintf("%d", tweet.Id))
		values.Set("auto_populate_reply_metadata", "true")

		_, err = api.PostTweet(statusTextFinal, values)
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Printf("Number of tweets in latest file: %d\n", len(last.Tweets))
	fmt.Printf("Tweets contains %d, adding %d more...\n", len(Tweets), (numTweets - len(Tweets)))

	for i := 0; (len(Tweets) < numTweets) && (i <= last.NumTweets); i++ {
		Tweets = append(Tweets, Tweet{last.Tweets[i].Text, last.Tweets[i].Sentiment, last.Tweets[i].Id})
	}

	for _, tweet := range Tweets {
		totalSentiment += tweet.Sentiment
	}

	average := totalSentiment / float32(len(Tweets))
	numTweets = len(Tweets)
	filename := fmt.Sprintf("%d", currentTimestamp)
	responseJson := TweetSentiment{currentTimestamp, Tweets, average, numTweets, nextUpdate}
	jsonString, err := json.MarshalIndent(responseJson, "", "    ")
	ioutil.WriteFile(filename, jsonString, 0644)

	if len(tweetsResponse) > 0 {
		fmt.Printf("New tweets exist. Publishing new file to bucket.\n")
		upload := bucket.Object(filename).NewWriter(ctx)
		upload.ContentType = "application/json"
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

	//if len(tweetsResponse) > 0 {
	//	fmt.Printf("Posting public tweet.\n")
	//	values := url.Values{}
	//
	//	_, err = api.PostTweet(fmt.Sprintf("%s http://www.isTrumpMeltingDown.com", statusText), values)
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//
	//}
}