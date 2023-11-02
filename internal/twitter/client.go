package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	baseClient *BaseClient
	baseURL    string
	callLimit  int64

	forbidden bool
	pending   int64
	remaining int64
	reset     int64

	mtx *sync.Mutex
}

func (client *Client) isAvailable() (bool, int64) {
	if !client.baseClient.Connected() || client.forbidden {
		return false, -1
	}

	client.mtx.Lock()
	defer client.mtx.Unlock()

	if client.reset < time.Now().Unix() {
		if client.pending >= client.callLimit {
			return false, time.Now().Add(15 * time.Minute).Unix()
		}
	} else if client.pending >= client.remaining {
		return false, client.reset
	}

	client.pending++
	return true, -1
}

func (client *Client) handleResponse(statusCode int, header http.Header, body []byte) {
	client.mtx.Lock()
	defer client.mtx.Unlock()

	if !client.baseClient.Connected() {
		return
	}

	// ignore if SetAuthData has been called recently
	responseTime, _ := time.Parse(time.RFC1123, header.Get("Date"))
	if responseTime.Before(client.baseClient.LastSyncedAt()) {
		return
	}

	client.pending--

	if statusCode == http.StatusUnauthorized {
		client.baseClient.Disconnect()
		go client.reconnect()
		return
	}

	if statusCode == http.StatusForbidden {
		client.forbidden = true
		return
	}

	var respBody struct {
		Errors []Error `json:"errors"`
	}

	err := json.Unmarshal(body, &respBody)
	if err == nil && len(respBody.Errors) > 0 {
		for _, e := range respBody.Errors {
			if e.Name == "AuthorizationError" {
				client.baseClient.Disconnect()
				go client.reconnect()
				return
			}
		}
	}

	newRemaining, _ := strconv.ParseInt(header.Get("X-Rate-Limit-Remaining"), 10, 64)
	newReset, _ := strconv.ParseInt(header.Get("X-Rate-Limit-Reset"), 10, 64)

	if newReset == client.reset {
		// two requests is in the same timeframe
		// if later request somehow acquired lock first, newRemaining > client.remaining, and we should ignore this case
		if newRemaining < client.remaining {
			client.remaining = newRemaining
		}
	} else if newReset > client.reset {
		client.reset = newReset
		client.remaining = newRemaining
	}
}

func (client *Client) reconnect() {
	if !client.baseClient.Connected() {
		// already in connecting progress
		return
	}

	if !client.baseClient.CanReconnect() {
		log.Printf("[WARN] client (%s) encountered an error but cannot reconnect", client.baseClient.Username)
	}

	ctx := context.Background()
	attempt := 0
	for {
		log.Printf("[INFO] client (%s) trying to reconnect...", client.baseClient.Username)
		err := func() error {
			err := client.baseClient.Login(ctx)
			if err != nil {
				return err
			}
			err = client.fetchLimit()
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			sleep := (1 << attempt) * time.Second
			attempt++
			log.Printf("[INFO] client (%s) failed to reconnect: %s, sleeping %s", client.baseClient.Username, err, sleep)
			time.Sleep(sleep)
			continue
		}

		log.Printf("[INFO] client (%s) reconnected", client.baseClient.Username)
		return
	}
}

func (client *Client) fetchLimit() error {
	req, _ := http.NewRequest(http.MethodGet, client.baseURL, nil)
	resp, err := client.baseClient.DoRequestWithAuth(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	newRemaining, err := strconv.ParseInt(resp.Header.Get("X-Rate-Limit-Remaining"), 10, 64)
	if err != nil {
		return err
	}
	client.remaining = newRemaining

	newReset, _ := strconv.ParseInt(resp.Header.Get("X-Rate-Limit-Reset"), 10, 64)
	if err != nil {
		return err
	}
	client.reset = newReset

	switch resp.StatusCode {
	case http.StatusOK:
		bodyBz, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var respBody struct {
			Errors []Error `json:"errors"`
		}

		err = json.Unmarshal(bodyBz, &respBody)
		if err == nil && len(respBody.Errors) > 0 {
			for _, e := range respBody.Errors {
				if e.Name == "AuthorizationError" {
					client.forbidden = true
				}
			}
		}
	case http.StatusForbidden:
		client.forbidden = true

	default:
		return fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	return nil
}
