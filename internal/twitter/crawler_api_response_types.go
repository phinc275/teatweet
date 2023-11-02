package twitter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type UserMention struct {
	IDStr string `json:"id_str"`
}

type Hashtag struct {
	Text string `json:"text"`
}

type Symbol struct {
	Text string `json:"text"`
}

type URL struct {
	DisplayURL  string `json:"display_url"`
	ExpandedURL string `json:"expanded_url"`
	URL         string `json:"url"`
	Indices     [2]int `json:"indices"`
}

type UserResults struct {
	Result struct {
		RestID string          `json:"rest_id"`
		Legacy TweetUserLegacy `json:"legacy"`
	} `json:"result"`
}

type TweetUserLegacy struct {
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
}

type TweetResults struct {
	Result struct {
		RestID string `json:"rest_id"`
		Core   struct {
			UserResults UserResults `json:"user_results"`
		} `json:"core"`
		Legacy TweetResultLegacy `json:"legacy"`
		Views  TweetResultViews  `json:"views"`
	} `json:"result"`
}

type TweetResultViews struct {
	Count int64 `json:"count"`
}

func (obj *TweetResultViews) UnmarshalJSON(data []byte) error {
	type Alias TweetResultViews
	aux := &struct {
		*Alias
		Count string `json:"count"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.Count != "" {
		count, err := strconv.ParseInt(aux.Count, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot unmarshal %s into an int64", aux.Count)
		}
		obj.Count = count
	}

	return nil
}

type TweetResultLegacy struct {
	CreatedAt time.Time `json:"created_at"`
	Entities  struct {
		UserMentions []UserMention `json:"user_mention"`
		Hashtags     []Hashtag     `json:"hashtags"`
		Symbols      []Symbol      `json:"symbols"`
		URLs         []URL         `json:"urls"`
	} `json:"entities"`
	FullText             string `json:"full_text"`
	InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
	QuotedStatusIDStr    string `json:"quoted_status_id_str"`

	FavoriteCount int64 `json:"favorite_count"`
	QuoteCount    int64 `json:"quote_count"`
	ReplyCount    int64 `json:"reply_count"`
	RetweetCount  int64 `json:"retweet_count"`
	IsQuoteStatus bool  `json:"is_quote_status"`
}

func (obj *TweetResultLegacy) UnmarshalJSON(data []byte) error {
	type Alias TweetResultLegacy
	aux := &struct {
		*Alias
		CreatedAt string `json:"created_at"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	createdAt, err := time.Parse(time.RubyDate, aux.CreatedAt)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %s into a time.Time (%s)", aux.CreatedAt, time.RubyDate)
	}
	obj.CreatedAt = createdAt

	return nil
}

type Error struct {
	Message string `json:"message"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Code    int    `json:"code"`
}

// ========= Responses

// SearchTimelineResponse is response from SearchTimeline API
// count: 20
// rate limit 50 per 15 minutes
type SearchTimelineResponse struct {
	Data struct {
		SearchByRawQuery struct {
			SearchTimeline struct {
				Timeline struct {
					Instructions []SearchTimelineInstruction `json:"instructions"`
				} `json:"timeline"`
			} `json:"search_timeline"`
		} `json:"search_by_raw_query"`
	} `json:"data"`
	Errors []Error `json:"errors"`
}

// RetweetersResponse is response from Retweeters API
// count: 100
// rate limit 500 per 15 minutes
type RetweetersResponse struct {
	Data struct {
		RetweetersTimeline struct {
			Timeline struct {
				Instructions []RetweetersInstruction `json:"instructions"`
			} `json:"timeline"`
		} `json:"retweeters_timeline"`
	} `json:"data"`
	Errors []Error `json:"errors"`
}

// FavoritersResponse is response from Favoriters API
// count: 100
// rate limit 500 per 15 minutes
type FavoritersResponse struct {
	Data struct {
		FavoritersTimeline struct {
			Timeline struct {
				Instructions []FavoritersInstruction `json:"instructions"`
			} `json:"timeline"`
		} `json:"favoriters_timeline"`
	} `json:"data"`
	Errors []Error `json:"errors"`
}

type FollowingResponse struct {
	Data struct {
		User struct {
			Result struct {
				Timeline struct {
					Timeline struct {
						Instructions []FollowingInstruction `json:"instructions"`
					} `json:"timeline"`
				} `json:"timeline"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
	Errors []Error `json:"errors"`
}

// ========= Instructions

type Instruction[T any] struct {
	Type    string `json:"type"`
	Entries []T    `json:"entries"`
}

type (
	SearchTimelineInstruction struct {
		Instruction[SearchTimelineEntry]
		Entry *InstructionEntry `json:"entry"`
	}
	RetweetersInstruction Instruction[RetweetersEntry]
	FavoritersInstruction Instruction[FavoritersEntry]
	FollowingInstruction  Instruction[FollowingEntry]
)

// ========= Entries

type Entry[T any] struct {
	EntryID   string `json:"entryId"`
	SortIndex int64  `json:"sortIndex"`
	Content   T      `json:"content"`
}
type (
	InstructionEntry struct {
		Content InstructionEntryContent `json:"content"`
	}
	SearchTimelineEntry Entry[SearchTimelineEntryContent]
	RetweetersEntry     Entry[RetweetersEntryContent]
	FavoritersEntry     Entry[FavoritersEntryContent]
	FollowingEntry      Entry[FollowingEntryContent]
)

func (obj *SearchTimelineEntry) UnmarshalJSON(data []byte) error {
	type Alias SearchTimelineEntry
	aux := &struct {
		*Alias
		SortIndex string `json:"sortIndex"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	sortIndex, err := strconv.ParseInt(aux.SortIndex, 10, 64)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %s into an int64", aux.SortIndex)
	}
	obj.SortIndex = sortIndex

	return nil
}

func (obj *RetweetersEntry) UnmarshalJSON(data []byte) error {
	type Alias RetweetersEntry
	aux := &struct {
		*Alias
		SortIndex string `json:"sortIndex"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	sortIndex, err := strconv.ParseInt(aux.SortIndex, 10, 64)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %s into an int64", aux.SortIndex)
	}
	obj.SortIndex = sortIndex

	return nil
}

func (obj *FavoritersEntry) UnmarshalJSON(data []byte) error {
	type Alias FavoritersEntry
	aux := &struct {
		*Alias
		SortIndex string `json:"sortIndex"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	sortIndex, err := strconv.ParseInt(aux.SortIndex, 10, 64)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %s into an int64", aux.SortIndex)
	}
	obj.SortIndex = sortIndex

	return nil
}

func (obj *FollowingEntry) UnmarshalJSON(data []byte) error {
	type Alias FollowingEntry
	aux := &struct {
		*Alias
		SortIndex string `json:"sortIndex"`
	}{
		Alias: (*Alias)(obj),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	sortIndex, err := strconv.ParseInt(aux.SortIndex, 10, 64)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %s into an int64", aux.SortIndex)
	}
	obj.SortIndex = sortIndex

	return nil
}

// ========= EntryContent

type EmptyEntryContent struct {
	EntryType  string `json:"entryType"`
	CursorType string `json:"cursorType"`
	Value      string `json:"value"`
}

type EntryContent[T any] struct {
	EmptyEntryContent
	ItemContent *T `json:"itemContent"` // nullable
}

type (
	InstructionEntryContent    EmptyEntryContent
	SearchTimelineEntryContent EntryContent[SearchTimelineEntryItemContent]
	RetweetersEntryContent     EntryContent[RetweetersEntryItemContent]
	FavoritersEntryContent     EntryContent[FavoritersEntryItemContent]
	FollowingEntryContent      EntryContent[FollowingEntryItemContent]
)

// ========= EntryContentItem

type SearchTimelineEntryItemContent struct {
	ItemType     string       `json:"itemType"`
	TweetResults TweetResults `json:"tweet_results"`
}

type RetweetersEntryItemContent struct {
	ItemType    string      `json:"itemType"`
	UserResults UserResults `json:"user_results"`
}

type FavoritersEntryItemContent struct {
	ItemType    string      `json:"itemType"`
	UserResults UserResults `json:"user_results"`
}

type FollowingEntryItemContent struct {
	ItemType    string      `json:"itemType"`
	UserResults UserResults `json:"user_results"`
}
