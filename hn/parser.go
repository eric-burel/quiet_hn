package hn

import (
	"net/url"
	"strings"
)

// item is the same as the hn.Item, but adds the Host field
type ParsedItem struct {
	Item
	Host string
}

func ParseHNItem(hnItem Item) ParsedItem {
	ret := ParsedItem{hnItem, ""}
	url, err := url.Parse(ret.Item.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(url.Hostname(), "www.")
	}
	return ret
}

func (item ParsedItem) isStoryLink() bool {
	return item.Type == "story" && item.URL != ""
}
