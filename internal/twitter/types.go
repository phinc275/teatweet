package twitter

import (
	"context"
	"time"
)

type Reply struct {
	TweetID         string
	UserID          string
	Text            string
	NormalizedText  string
	CreatedAt       time.Time
	Hashtags        []string
	LoweredHashtags []string
	Symbols         []string
	LoweredSymbols  []string
	Sort            int64
}

type Quote struct {
	TweetID         string
	UserID          string
	Text            string
	NormalizedText  string
	CreatedAt       time.Time
	Hashtags        []string
	LoweredHashtags []string
	Symbols         []string
	LoweredSymbols  []string
	Sort            int64
}

type Retweet struct {
	TweetID string
	UserID  string
	Sort    int64
}

type Like struct {
	TweetID string
	UserID  string
	Sort    int64
}

type Following struct {
	TargetID   string
	UserID     string
	Name       string
	ScreenName string
}

type StatusStat struct {
	UserID         string
	UserScreenName string
	UserName       string
	ID             string
	CreatedAt      time.Time
	IsQuoteStatus  bool
	ViewCount      int64
	QuoteCount     int64
	ReplyCount     int64
	RetweetCount   int64
	FavoriteCount  int64
}

type ICrawlAPI interface {
	Replies(ctx context.Context, tweetID string, cursor string) ([]Reply, string, error)
	Quotes(ctx context.Context, tweetID string, cursor string) ([]Quote, string, error)
	Retweets(ctx context.Context, tweetID string, cursor string) ([]Retweet, string, error)
	Likes(ctx context.Context, tweetID string, cursor string) ([]Like, string, error)
	Following(ctx context.Context, targetID string, cursor string) ([]Following, string, error)
	StatusesByScreenName(ctx context.Context, userID string, cursor string) ([]StatusStat, string, error)
}
