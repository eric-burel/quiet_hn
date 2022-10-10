// Package hn implements a really basic Hacker News client
package hn

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
)

const (
	apiBase = "https://hacker-news.firebaseio.com/v0"
)

// Client is an API client used to interact with the Hacker News API
type Client struct {
	// unexported fields...
	apiBase string
}

// Making the Client zero value useful without forcing users to do something
// like `NewClient()`
func (c *Client) defaultify() {
	if c.apiBase == "" {
		c.apiBase = apiBase
	}
}

// TopItems returns the ids of roughly 450 top items in decreasing order. These
// should map directly to the top 450 things you would see on HN if you visited
// their site and kept going to the next page.
//
// TopItmes does not filter out job listings or anything else, as the type of
// each item is unknown without further API calls.
func (c *Client) TopItems() ([]int, error) {
	c.defaultify()
	resp, err := http.Get(fmt.Sprintf("%s/topstories.json", c.apiBase))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ids []int
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// GetItem will return the Item defined by the provided ID.
func (c *Client) GetItem(id int) (Item, error) {
	c.defaultify()
	var item Item
	resp, err := http.Get(fmt.Sprintf("%s/item/%d.json", c.apiBase, id))
	if err != nil {
		return item, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&item)
	if err != nil {
		return item, err
	}
	return item, nil
}

func (c *Client) GetValidItem(id int, ch chan *ParsedItem) {
	fmt.Printf("Get item %d\n", id)
	hnItem, err := c.GetItem(id)
	item := ParseHNItem(hnItem)
	if item.isStoryLink() {
		// ignore this item
		ch <- nil
	}
	if err != nil {
		ch <- nil
	}
	ch <- &item
}

func (c *Client) GetItems(ids []int, numStories int) ([]ParsedItem, error) {
	// Channel can be buffered, we want to send only 30 stories
	// this means checking the story validity in the goroutine
	ch := make(chan *ParsedItem, numStories)

	// Get a slice of 30 ids * 1.25
	// Trigger a goroutine per id
	// Append to stories
	// Done if 30 stories, otherwise get a new slice of missing count + 5

	stories_map := make(map[int]ParsedItem)

	slice_start := 0
	for len(stories_map) < numStories && slice_start < len(ids) {
		slice_end := int(math.Min(float64(slice_start+35), float64(len(ids))))
		ids_slice := ids[slice_start:slice_end]
		//slice_length := slice_end - slice_start
		fmt.Printf("Looking for ids in slice %d:%d\n", slice_start, slice_end)

		for _, id := range ids_slice {
			go c.GetValidItem(id, ch)
		}

		for range ids_slice {
			item := <-ch
			if item != nil {
				fmt.Printf("Got an item %s", item.Title)
				stories_map[item.ID] = *item
			}
			//stories = append(stories, item)
		}

		slice_start = slice_end
	}
	// at this point, we may have
	// - too few stories if we ran out of ids
	// - too many stories if the stories we fetched were all valid

	// ids are already sorted => just build a map of fetched stories per id
	var stories []ParsedItem
	// for each id READ IN ORDER, check if the story is in map, if yes, add to stories,
	// stop when we have found all ids of the map
	// => we end up with a sorted list of stories
	for i := 0; i < len(ids) && len(stories) < len(stories_map); i++ {
		id := ids[i]
		story, has := stories_map[id]
		if has {
			stories = append(stories, story)
		}

	}

	if len(stories) < numStories {
		return stories, errors.New("not enough stories")
	}
	if len(stories) > numStories {
		// TODO remove additional stories
		stories = stories[0:numStories]
	}
	return stories, nil
}

// Item represents a single item returned by the HN API. This can have a type
// of "story", "comment", or "job" (and probably more values), and one of the
// URL or Text fields will be set, but not both.
//
// For the purpose of this exercise, we only care about items where the
// type is "story", and the URL is set.
type Item struct {
	By          string `json:"by"`
	Descendants int    `json:"descendants"`
	ID          int    `json:"id"`
	Kids        []int  `json:"kids"`
	Score       int    `json:"score"`
	Time        int    `json:"time"`
	Title       string `json:"title"`
	Type        string `json:"type"`

	// Only one of these should exist
	Text string `json:"text"`
	URL  string `json:"url"`
}
