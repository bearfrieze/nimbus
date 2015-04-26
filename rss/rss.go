package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
)

type Feed struct {
	Channel Channel `xml:"channel"`
	Raw     []byte  `xml:",innerxml"`
}

type Channel struct {
	Title         string `xml:"title"`
	TTL           int    `xml:"ttl"`
	LastBuildDate string `xml:"lastBuildDate"`
	PubDate       string `xml:"pubDate"`
	Items         []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func NewFeed(data []byte) (*Feed, error) {

	if !IsFeed(data) {
		return nil, fmt.Errorf("Not an RSS feed")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	var f Feed
	if err := decoder.Decode(&f); err != nil {
		return nil, fmt.Errorf("Failed to decode feed: %s", err)
	}
    if len(f.Channel.Items) == 0 {
        return nil, fmt.Errorf("Feed has no items")
    }
	return &f, nil
}

func IsFeed(data []byte) bool {

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		token, _ := decoder.Token()
		if se, ok := token.(xml.StartElement); ok {
            // fmt.Printf("%+v\n\n", se)
			return se.Name.Local == "rss"
		}
	}
	return false
}
