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

	values := url.Values{}
	values.Set("screen_name", "realdonaldtrump")
	values.Set("count", fmt.Sprintf("%d", numTweets))

	tweetsResponse, err := api.GetUserTimeline(values)
	if err != nil {
		fmt.Println(err)
	}

	ctx := context.Background()
	client, err := language.NewClient(ctx)
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
	}


	var Tweets []Tweet
	var totalSentiment float32

	tweetsResponse = tweetsResponse[:numTweets]

	fmt.Println("Tweets: ")
	//fmt.Printf("%v\n", tweetsResponse)
	fmt.Println(len(tweetsResponse))

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
		totalSentiment += sentiment.DocumentSentiment.Score
		Tweets = append(Tweets, Tweet{tweet.FullText, sentiment.DocumentSentiment.Score, tweet.Id})
	}

	numTweets = len(Tweets)
	filename := fmt.Sprintf("%d", currentTimestamp)
	responseJson := TweetSentiment{currentTimestamp, Tweets, totalSentiment/float32(numTweets-1), numTweets-1, nextUpdate}
	jsonString, err := json.MarshalIndent(responseJson, "", "    ")
	ioutil.WriteFile(filename, jsonString, 0644)

	// uploading the file to a bucket
	storageClient, err := storage.NewClient(ctx)
	bucketName := os.Getenv("TRUMPMELTDOWN_SENTIMENT_BUCKET")
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
	}

	bucket := storageClient.Bucket(bucketName)

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

	os.Remove(filename)
}