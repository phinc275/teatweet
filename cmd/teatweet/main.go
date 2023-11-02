package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/phinc275/teatweet/internal/twitter"
	"github.com/urfave/cli/v2"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	app := &cli.App{
		Name:  "teatweet",
		Usage: "Icetea Labs Twitter service?",
		Commands: []*cli.Command{
			newServeCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newServeCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "start the web server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Value: "0.0.0.0:8001",
				Usage: "serve address",
			},
		},
		Action: func(c *cli.Context) error {
			credentialsStr := os.Getenv("TWITTER_CREDENTIALS")
			var credentials []twitter.Credential
			err := json.Unmarshal([]byte(credentialsStr), &credentials)
			if err != nil {
				return fmt.Errorf("failed to parse credentials: %v", err)
			}

			crawler, err := twitter.NewCrawler(credentials)
			if err != nil {
				return fmt.Errorf("failed to initiate crawler: %v", err)
			}

			http.HandleFunc("/following", followingHandlerFn(crawler))
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("hello, world!"))
			})

			addr := c.String("addr")
			log.Printf("starting server on %s\n", addr)
			if err := http.ListenAndServe(addr, nil); err != http.ErrServerClosed {
				return fmt.Errorf("failed to start server: %v", err)
			}

			return nil
		},
	}
}

func followingHandlerFn(crawler *twitter.Crawler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		twitterUserID := r.URL.Query().Get("id")
		followingIDs, err := crawlFollowing(r.Context(), crawler, twitterUserID)
		respJSON(w, followingIDs, err)
	}
}

type Following struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

func crawlFollowing(ctx context.Context, crawler *twitter.Crawler, userID string) ([]Following, error) {
	if userID == "" {
		return nil, fmt.Errorf("invalid user id")
	}

	cursor := ""
	results := make([]Following, 0)
	for {
		var items []twitter.Following
		var err error

		items, cursor, err = crawler.Following(ctx, userID, cursor)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			results = append(results, Following{
				ID:       item.UserID,
				Username: item.ScreenName,
				Name:     item.Name,
			})
		}

		if cursor == "" {
			break
		}
	}

	return results, nil
}

func respJSON(w http.ResponseWriter, data interface{}, err error) {
	type resp struct {
		Code    int         `json:"code"`
		Data    interface{} `json:"data"`
		Message string      `json:"message"`
	}

	var r resp
	if err == nil {
		r = resp{
			Code: 0,
			Data: data,
		}
	} else {
		r = resp{
			Code:    -1,
			Message: err.Error(),
		}
	}

	rBz, _ := json.Marshal(r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(rBz)
}
