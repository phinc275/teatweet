package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/hiendaovinh/toolkit/pkg/arr"
)

const (
	apiCallSearchTimeline string = "search-timeline"
	apiCallRetweeters     string = "retweeters"
	apiCallFavoriters     string = "favoriters"
	apiCallFollowing      string = "following"
)

var apis = map[string]struct {
	URL       string
	Variables string
	Features  string
	CallLimit int64
}{
	apiCallSearchTimeline: {
		URL:       "https://twitter.com/i/api/graphql/tOUz374Df84NaVVr3M1p6g/SearchTimeline?variables=%7B%22rawQuery%22%3A%22quoted_tweet_id%3A1701892872574996627%22%2C%22count%22%3A20%2C%22querySource%22%3A%22tdqt%22%2C%22product%22%3A%22Top%22%7D&features=%7B%22responsive_web_graphql_exclude_directive_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22responsive_web_home_pinned_timelines_enabled%22%3Atrue%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22tweetypie_unmention_optimization_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Afalse%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_media_download_video_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D",
		CallLimit: 50,
	},
	apiCallRetweeters: {
		URL:       "https://twitter.com/i/api/graphql/FnXqVNJSKmqudpmIIEeUCQ/Retweeters?variables=%7B%22tweetId%22%3A%221701892872574996627%22%2C%22count%22%3A20%2C%22includePromotedContent%22%3Atrue%7D&features=%7B%22responsive_web_graphql_exclude_directive_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22responsive_web_home_pinned_timelines_enabled%22%3Atrue%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22tweetypie_unmention_optimization_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Afalse%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_media_download_video_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D",
		CallLimit: 500,
	},
	apiCallFavoriters: {
		URL:       "https://twitter.com/i/api/graphql/zXD9lMy1-V_N1OcON9JtEQ/Favoriters?variables=%7B%22tweetId%22%3A%221701892872574996627%22%2C%22count%22%3A20%2C%22includePromotedContent%22%3Atrue%7D&features=%7B%22responsive_web_graphql_exclude_directive_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22responsive_web_home_pinned_timelines_enabled%22%3Atrue%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22tweetypie_unmention_optimization_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Afalse%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_media_download_video_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D",
		CallLimit: 500,
	},
	apiCallFollowing: {
		URL:       "https://twitter.com/i/api/graphql/OueaMJOJ0r0lmGTxl2V4Mw/Following?variables=%7B%22userId%22%3A%221415522287126671363%22%2C%22count%22%3A20%2C%22includePromotedContent%22%3Afalse%7D&features=%7B%22responsive_web_graphql_exclude_directive_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22responsive_web_home_pinned_timelines_enabled%22%3Atrue%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22tweetypie_unmention_optimization_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Afalse%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_media_download_video_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D",
		CallLimit: 500,
	},
}

type Crawler struct {
	clients map[string]map[string]*Client
}

var _ ICrawlAPI = (*Crawler)(nil)

func NewCrawler(credentials []Credential) (*Crawler, error) {
	clients := make(map[string]map[string]*Client)
	crawler := &Crawler{clients: clients}

	crawler.init(credentials)
	return crawler, nil
}

func NewCrawlerFromEnvs(vs map[string]string) (*Crawler, error) {
	var twitterCredentials []Credential
	credentialsStr := vs["TWITTER_CREDENTIALS"]

	if credentialsStr == "" {
		credentialsStr = `[]`
	}

	err := json.Unmarshal([]byte(credentialsStr), &twitterCredentials)
	if err != nil {
		return nil, err
	}

	return NewCrawler(twitterCredentials)
}

func (crawler *Crawler) init(credentials []Credential) {
	ctx := context.Background()
	wg := &sync.WaitGroup{}

	for idx, credential := range credentials {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, idx int, credential Credential) {
			defer wg.Done()
			baseClient, err := NewBaseClientFromPassword(ctx, credential.Username, credential.Password)
			if err != nil {
				log.Printf("[WARN] skipping twitter (%s) due to error: %s", credential.Username, err)
				return
			}

			for apiName, api := range apis {
				client := &Client{
					baseClient: baseClient,
					baseURL:    api.URL,
					callLimit:  api.CallLimit,
					pending:    0,
					remaining:  0,
					reset:      0,
					mtx:        &sync.Mutex{},
				}
				err = client.fetchLimit()
				if err != nil {
					log.Printf("[WARN] skipping twitter (%s) for api %s due to error: %s", credential.Username, apiName, err)
					continue
				}
				if crawler.clients[apiName] == nil {
					crawler.clients[apiName] = make(map[string]*Client)
				}
				crawler.clients[apiName][credential.Username] = client
			}
		}(ctx, wg, idx, credential)
	}

	wg.Wait()
}

var (
	apiTweetFeaturesBz, _ = json.Marshal(map[string]interface{}{
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                false,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_media_download_video_enabled":                             false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})

	apiUserFeaturesBz, _ = json.Marshal(map[string]interface{}{
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                false,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_media_download_video_enabled":                             false,
		"responsive_web_enhance_cards_enabled":                                    false,
		"responsive_web_home_pinned_timelines_enabled":                            true,
	})

	cursorTypes      = map[string]bool{"Bottom": true, "ShowMoreThreads": true, "ShowMoreThreadsPrompt": true}
	regexUserEntryID = regexp.MustCompile(`user-(\d+)`)
)

func (crawler *Crawler) Replies(ctx context.Context, tweetID string, cursor string) ([]Reply, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallSearchTimeline].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"rawQuery":    fmt.Sprintf("filter:replies conversation_id:%s", tweetID),
		"count":       20,
		"querySource": "tdqt",
		"product":     "Latest",
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiTweetFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallSearchTimeline, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj SearchTimelineResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	replies := make([]Reply, 0)
	nextCursor := ""
	mainInstructionEntryLength := 0

	for _, instruction := range respObj.Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions {
		if instruction.Type == "TimelineReplaceEntry" && instruction.Entry != nil {
			if instruction.Entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[instruction.Entry.Content.CursorType] {
				nextCursor = instruction.Entry.Content.Value
			}
			continue
		}

		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			// defensive
			if entry.Content.ItemContent == nil {
				continue
			}

			mainInstructionEntryLength++
			if entry.Content.ItemContent.TweetResults.Result.Legacy.InReplyToStatusIDStr != tweetID {
				continue
			}

			tweetResult := entry.Content.ItemContent.TweetResults.Result
			parts := make([]string, 0, len(tweetResult.Legacy.Entities.URLs)*2+1)
			lastPartIndex := 0

			for _, u := range tweetResult.Legacy.Entities.URLs {
				parts = append(parts, tweetResult.Legacy.FullText[lastPartIndex:u.Indices[0]])
				parts = append(parts, u.DisplayURL)
				lastPartIndex = u.Indices[1]
			}
			parts = append(parts, tweetResult.Legacy.FullText[lastPartIndex:])
			normalizedText := strings.Join(parts, "")

			replies = append(replies, Reply{
				TweetID:         tweetID,
				UserID:          tweetResult.Core.UserResults.Result.RestID,
				Text:            tweetResult.Legacy.FullText,
				NormalizedText:  normalizedText,
				CreatedAt:       tweetResult.Legacy.CreatedAt,
				Hashtags:        arr.ArrMap(tweetResult.Legacy.Entities.Hashtags, func(v Hashtag) string { return v.Text }),
				LoweredHashtags: arr.ArrMap(tweetResult.Legacy.Entities.Hashtags, func(v Hashtag) string { return strings.ToLower(v.Text) }),
				Symbols:         arr.ArrMap(tweetResult.Legacy.Entities.Symbols, func(v Symbol) string { return v.Text }),
				LoweredSymbols:  arr.ArrMap(tweetResult.Legacy.Entities.Symbols, func(v Symbol) string { return strings.ToLower(v.Text) }),
				Sort:            entry.SortIndex,
			})
		}
	}

	if mainInstructionEntryLength == 0 {
		nextCursor = ""
	}

	return replies, nextCursor, nil
}

func (crawler *Crawler) Quotes(ctx context.Context, tweetID string, cursor string) ([]Quote, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallSearchTimeline].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"rawQuery":    fmt.Sprintf("quoted_tweet_id:%s", tweetID),
		"count":       20,
		"querySource": "tdqt",
		"product":     "Latest",
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiTweetFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallSearchTimeline, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj SearchTimelineResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	quotes := make([]Quote, 0)
	nextCursor := ""
	mainInstructionEntryLength := 0

	for _, instruction := range respObj.Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions {
		if instruction.Type == "TimelineReplaceEntry" && instruction.Entry != nil {
			if instruction.Entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[instruction.Entry.Content.CursorType] {
				nextCursor = instruction.Entry.Content.Value
			}
			continue
		}

		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			if entry.Content.ItemContent == nil {
				continue
			}

			mainInstructionEntryLength++
			if entry.Content.ItemContent.TweetResults.Result.Legacy.QuotedStatusIDStr != tweetID {
				continue
			}

			tweetResult := entry.Content.ItemContent.TweetResults.Result
			parts := make([]string, 0, len(tweetResult.Legacy.Entities.URLs)*2+1)
			lastPartIndex := 0

			for _, u := range tweetResult.Legacy.Entities.URLs {
				parts = append(parts, tweetResult.Legacy.FullText[lastPartIndex:u.Indices[0]])
				parts = append(parts, u.DisplayURL)
				lastPartIndex = u.Indices[1]
			}
			parts = append(parts, tweetResult.Legacy.FullText[lastPartIndex:])
			normalizedText := strings.Join(parts, "")

			quotes = append(quotes, Quote{
				TweetID:         tweetID,
				UserID:          tweetResult.Core.UserResults.Result.RestID,
				Text:            tweetResult.Legacy.FullText,
				NormalizedText:  normalizedText,
				CreatedAt:       tweetResult.Legacy.CreatedAt,
				Hashtags:        arr.ArrMap(tweetResult.Legacy.Entities.Hashtags, func(v Hashtag) string { return v.Text }),
				LoweredHashtags: arr.ArrMap(tweetResult.Legacy.Entities.Hashtags, func(v Hashtag) string { return strings.ToLower(v.Text) }),
				Symbols:         arr.ArrMap(tweetResult.Legacy.Entities.Symbols, func(v Symbol) string { return v.Text }),
				LoweredSymbols:  arr.ArrMap(tweetResult.Legacy.Entities.Symbols, func(v Symbol) string { return strings.ToLower(v.Text) }),
				Sort:            entry.SortIndex,
			})
		}
	}

	if mainInstructionEntryLength == 0 {
		nextCursor = ""
	}

	return quotes, nextCursor, nil
}

func (crawler *Crawler) Retweets(ctx context.Context, tweetID string, cursor string) ([]Retweet, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallRetweeters].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"tweetId":                tweetID,
		"count":                  100,
		"includePromotedContent": true,
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiTweetFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallRetweeters, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj RetweetersResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	retweeters := make([]Retweet, 0)
	nextCursor := ""

	for _, instruction := range respObj.Data.RetweetersTimeline.Timeline.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			if entry.Content.ItemContent == nil {
				continue
			}

			userID := entry.Content.ItemContent.UserResults.Result.RestID
			if userID == "" {
				if matches := regexUserEntryID.FindStringSubmatch(entry.EntryID); len(matches) == 2 {
					userID = matches[1]
				}
			}

			retweeters = append(retweeters, Retweet{
				TweetID: tweetID,
				UserID:  userID,
				Sort:    entry.SortIndex,
			})
		}
	}

	if len(retweeters) == 0 {
		nextCursor = ""
	}

	return retweeters, nextCursor, nil
}

func (crawler *Crawler) Likes(ctx context.Context, tweetID string, cursor string) ([]Like, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallFavoriters].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"tweetId":                tweetID,
		"count":                  100,
		"includePromotedContent": true,
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiTweetFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallFavoriters, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj FavoritersResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	favoriters := make([]Like, 0)
	nextCursor := ""

	for _, instruction := range respObj.Data.FavoritersTimeline.Timeline.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			if entry.Content.ItemContent == nil {
				continue
			}

			userID := entry.Content.ItemContent.UserResults.Result.RestID
			if userID == "" {
				if matches := regexUserEntryID.FindStringSubmatch(entry.EntryID); len(matches) == 2 {
					userID = matches[1]
				}
			}

			favoriters = append(favoriters, Like{
				TweetID: tweetID,
				UserID:  userID,
				Sort:    entry.SortIndex,
			})
		}
	}

	if len(favoriters) == 0 {
		nextCursor = ""
	}

	return favoriters, nextCursor, nil
}

func (crawler *Crawler) Following(ctx context.Context, targetID string, cursor string) ([]Following, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallFollowing].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"userId":                 targetID,
		"count":                  2,
		"includePromotedContent": false,
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiUserFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallFollowing, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj FollowingResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	followings := make([]Following, 0)
	nextCursor := ""

	for _, instruction := range respObj.Data.User.Result.Timeline.Timeline.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			if entry.Content.ItemContent == nil {
				continue
			}

			userID := entry.Content.ItemContent.UserResults.Result.RestID
			if userID == "" {
				if matches := regexUserEntryID.FindStringSubmatch(entry.EntryID); len(matches) == 2 {
					userID = matches[1]
				}
			}

			followings = append(followings, Following{
				TargetID:   targetID,
				UserID:     userID,
				Name:       entry.Content.ItemContent.UserResults.Result.Legacy.Name,
				ScreenName: entry.Content.ItemContent.UserResults.Result.Legacy.ScreenName,
			})
		}
	}

	if len(followings) == 0 {
		nextCursor = ""
	}

	return followings, nextCursor, nil
}

func (crawler *Crawler) StatusesByScreenName(ctx context.Context, screenName string, cursor string) ([]StatusStat, string, error) {
	req, _ := http.NewRequest("GET", apis[apiCallSearchTimeline].URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")

	v := map[string]interface{}{
		"rawQuery":    fmt.Sprintf("(from:%s) -filter:replies", screenName),
		"count":       20,
		"querySource": "typed_query",
		"product":     "Latest",
	}
	if cursor != "" {
		v["cursor"] = cursor
	}
	variablesBz, _ := json.Marshal(v)

	values := req.URL.Query()
	values.Set("variables", string(variablesBz))
	values.Set("features", string(apiTweetFeaturesBz))
	req.URL.RawQuery = values.Encode()

	req = req.WithContext(ctx)
	res, err := crawler.doRequest(apiCallSearchTimeline, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected response code %d", res.StatusCode)
	}

	respBz, _ := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	var respObj SearchTimelineResponse
	err = json.Unmarshal(respBz, &respObj)
	if err != nil {
		return nil, "", err
	}

	if len(respObj.Errors) > 0 {
		return nil, "", fmt.Errorf("server returns error: %s", strings.Join(
			arr.ArrMap(respObj.Errors, func(err Error) string { return err.Message }),
			";",
		))
	}

	statuses := make([]StatusStat, 0)
	nextCursor := ""
	mainInstructionEntryLength := 0

	for _, instruction := range respObj.Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions {
		if instruction.Type == "TimelineReplaceEntry" && instruction.Entry != nil {
			if instruction.Entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[instruction.Entry.Content.CursorType] {
				nextCursor = instruction.Entry.Content.Value
			}
			continue
		}

		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			// itemContent is not null if entry is either main or cursor
			if entry.Content.EntryType == "TimelineTimelineCursor" && cursorTypes[entry.Content.CursorType] {
				nextCursor = entry.Content.Value
				continue
			}

			if entry.Content.ItemContent == nil {
				continue
			}

			mainInstructionEntryLength++
			if entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName != screenName {
				continue
			}

			tweetResult := entry.Content.ItemContent.TweetResults.Result
			statuses = append(statuses, StatusStat{
				UserID:         tweetResult.Core.UserResults.Result.RestID,
				UserScreenName: tweetResult.Core.UserResults.Result.Legacy.ScreenName,
				UserName:       tweetResult.Core.UserResults.Result.Legacy.Name,
				ID:             tweetResult.RestID,
				CreatedAt:      tweetResult.Legacy.CreatedAt,
				IsQuoteStatus:  tweetResult.Legacy.IsQuoteStatus,
				ViewCount:      tweetResult.Views.Count,
				FavoriteCount:  tweetResult.Legacy.FavoriteCount,
				QuoteCount:     tweetResult.Legacy.QuoteCount,
				ReplyCount:     tweetResult.Legacy.ReplyCount,
				RetweetCount:   tweetResult.Legacy.RetweetCount,
			})
		}
	}

	if mainInstructionEntryLength == 0 {
		nextCursor = ""
	}

	return statuses, nextCursor, nil
}

func (crawler *Crawler) doRequest(call string, req *http.Request) (*http.Response, error) {
	clients := crawler.clients[call]
	perm := rand.Perm(len(clients))

	keys := make([]string, 0, len(clients))
	for username := range clients {
		keys = append(keys, username)
	}

	retryAfterInt64 := int64(math.MaxInt64)
	for _, j := range perm {
		client := clients[keys[j]]
		// check if limited
		ok, retryAfter := client.isAvailable()
		if !ok {
			if retryAfter >= 0 && retryAfter < retryAfterInt64 {
				retryAfterInt64 = retryAfter
			}
			continue
		}

		resp, err := client.baseClient.DoRequestWithAuth(req)
		if err != nil {
			return nil, err
		}

		statusCode := resp.StatusCode
		header := http.Header{}
		for headerName, headerValues := range resp.Header {
			for _, headerValue := range headerValues {
				header.Add(headerName, headerValue)
			}
		}

		var save io.ReadCloser
		if resp.Body != nil && resp.Body != http.NoBody {
			save, resp.Body, err = drainBody(resp.Body)
			if err != nil {
				return nil, err
			}
		}

		saveBz, _ := io.ReadAll(save)
		go client.handleResponse(statusCode, header, saveBz)

		return resp, err
	}

	return nil, &rateLimitError{Reset: retryAfterInt64}
}

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
