package twitter

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrawler(t *testing.T) {
	credentials := []Credential{
		{Username: os.Getenv("TWITTER_USERNAME"), Password: os.Getenv("TWITTER_PASSWORD")},
	}

	crawler, err := NewCrawler(credentials)
	assert.NoError(t, err, "failed to initiate crawler")

	tweetID := "1704696993757667786"
	wg := &sync.WaitGroup{}

	wg.Add(1)
	crawlRepliesResultChan := make(chan crawlResult[Reply], 1)
	go crawlReplies(wg, crawler, tweetID, crawlRepliesResultChan)

	wg.Add(1)
	crawlQuotesResultChan := make(chan crawlResult[Quote], 1)
	go crawlQuotes(wg, crawler, tweetID, crawlQuotesResultChan)

	wg.Add(1)
	crawlRetweetsResultChan := make(chan crawlResult[Retweet], 1)
	go crawlRetweets(wg, crawler, tweetID, crawlRetweetsResultChan)

	wg.Add(1)
	crawlLikesResultChan := make(chan crawlResult[Like], 1)
	go crawlLikes(wg, crawler, tweetID, crawlLikesResultChan)
	userID := "911011433147654144"

	wg.Add(1)
	crawlFollowingResultChan := make(chan crawlResult[Following], 1)
	go crawlFollowing(wg, crawler, userID, crawlFollowingResultChan)
	wg.Wait()

	replies := <-crawlRepliesResultChan
	log.Println("============================================================")
	log.Println("REPLIES:")
	log.Println("ERROR:", replies.Error)
	log.Printf("ITEMS: length=%d\n", len(replies.Items))
	for idx, item := range replies.Items {
		log.Printf("[ITEM %d]: [%s] %s\n", idx, item.CreatedAt, item.Text)
	}
	log.Println()

	quotes := <-crawlQuotesResultChan
	log.Println("============================================================")
	log.Println("QUOTES:")
	log.Println("ERROR:", quotes.Error)
	log.Printf("ITEMS: length=%d\n", len(quotes.Items))
	for idx, item := range quotes.Items {
		log.Printf("[ITEM %d]: [%s] %s\n", idx, item.CreatedAt, item.Text)
	}
	log.Println()

	// retweets := <-crawlRetweetsResultChan
	// log.Println("============================================================")
	// log.Println("RETWEETS:")
	// log.Println("ERROR:", retweets.Error)
	// log.Printf("ITEMS: length=%d\n", len(retweets.Items))
	// for idx, item := range retweets.Items {
	//	log.Printf("[ITEM %d]: %s\n", idx, item.UserID)
	// }
	// log.Println()
	//
	// likes := <-crawlLikesResultChan
	// log.Println("============================================================")
	// log.Println("LIKES:")
	// log.Println("ERROR:", likes.Error)
	// log.Printf("ITEMS: length=%d\n", len(likes.Items))
	// for idx, item := range likes.Items {
	//	log.Printf("[ITEM %d]: %s\n", idx, item.UserID)
	// }
	// log.Println()

	followings := <-crawlFollowingResultChan
	log.Println("============================================================")
	log.Println("FOLLOWING:")
	log.Println("ERROR:", followings.Error)
	log.Printf("ITEMS: length=%d\n", len(followings.Items))
	for idx, item := range followings.Items {
		log.Printf("[ITEM %d]: %s - %s - %s\n", idx, item.UserID, item.ScreenName, item.Name)
	}
	log.Println()
}

type crawlResult[T any] struct {
	Items []T
	Error error
}

func crawlReplies(wg *sync.WaitGroup, crawler *Crawler, tweetID string, resultChan chan crawlResult[Reply]) {
	defer wg.Done()

	cursor := ""
	items := make([]Reply, 0)
	var err error

	for {
		var crawled []Reply
		var nextCursor string
		crawled, nextCursor, err = crawler.Replies(context.Background(), tweetID, cursor)
		if err != nil {
			break
		}

		items = append(items, crawled...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	resultChan <- crawlResult[Reply]{Items: items, Error: err}
	log.Println("[INFO] crawlReplies DONE")
}

func crawlQuotes(wg *sync.WaitGroup, crawler *Crawler, tweetID string, resultChan chan crawlResult[Quote]) {
	defer wg.Done()

	cursor := ""
	items := make([]Quote, 0)
	var err error

	for {
		var crawled []Quote
		var nextCursor string
		crawled, nextCursor, err = crawler.Quotes(context.Background(), tweetID, cursor)
		if err != nil {
			break
		}

		items = append(items, crawled...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	resultChan <- crawlResult[Quote]{Items: items, Error: err}
	log.Println("[INFO] crawlQuotes DONE")
}

func crawlRetweets(wg *sync.WaitGroup, crawler *Crawler, tweetID string, resultChan chan crawlResult[Retweet]) {
	defer wg.Done()

	cursor := ""
	// items := make([]Retweet, 0)
	var err error

	for {
		var crawled []Retweet
		var nextCursor string
		log.Printf("[INFO] crawling retweeters: cursor=%s\n", cursor)
		crawled, nextCursor, err = crawler.Retweets(context.Background(), tweetID, cursor)
		if err != nil {
			break
		}

		log.Printf("[INFO] len(crawled) = %d\n", len(crawled))
		for idx, item := range crawled {
			log.Printf("[ITEM %d]: %d - %s\n", idx, item.Sort, item.UserID)
		}
		log.Println()

		// items = append(items, crawled...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// resultChan <- crawlResult[Retweet]{Items: items, Error: err}
	log.Println("[INFO] crawlRetweets DONE")
}

func crawlLikes(wg *sync.WaitGroup, crawler *Crawler, tweetID string, resultChan chan crawlResult[Like]) {
	defer wg.Done()

	cursor := ""
	// items := make([]Like, 0)
	var err error

	for {
		var crawled []Like
		var nextCursor string
		log.Printf("[INFO] crawling likes: cursor=%s\n", cursor)
		crawled, nextCursor, err = crawler.Likes(context.Background(), tweetID, cursor)
		if err != nil {
			break
		}

		log.Printf("[INFO] len(crawled) = %d\n", len(crawled))
		for idx, item := range crawled {
			log.Printf("[ITEM %d]: %d - %s\n", idx, item.Sort, item.UserID)
		}
		log.Println()

		// items = append(items, crawled...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// resultChan <- crawlResult[Like]{Items: items, Error: err}
	log.Println("[INFO] crawlLikes DONE")
}

func crawlFollowing(wg *sync.WaitGroup, crawler *Crawler, targetID string, resultChan chan crawlResult[Following]) {
	defer wg.Done()

	cursor := ""
	items := make([]Following, 0)
	var err error

	for {
		var crawled []Following
		var nextCursor string
		log.Printf("[INFO] crawling following: cursor=%s\n", cursor)
		crawled, nextCursor, err = crawler.Following(context.Background(), targetID, cursor)
		if err != nil {
			break
		}

		// log.Printf("[INFO] len(crawled) = %d\n", len(crawled))
		// for idx, item := range crawled {
		// 	log.Printf("[ITEM %d]: %s - %s\n", idx, item.UserID, item.Name)
		// }
		// log.Println()

		items = append(items, crawled...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	resultChan <- crawlResult[Following]{Items: items, Error: err}
	log.Println("[INFO] crawlFollowing DONE")
}
