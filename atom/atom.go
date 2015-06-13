package atom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
)

type Feed struct {
	Title   string  `xml:"title"`
	Updated string  `xml:"updated"`
	Entries []Entry `xml:"entry"`
	Raw     []byte  `xml:",innerxml"`
}

type Entry struct {
	Title     string `xml:"title"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Summary   string `xml:"summary"`
	Content   string `xml:"content"`
	Links     []Link `xml:"link"`
	ID        string `xml:"id"`
}

type Link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func NewFeed(data []byte) (*Feed, error) {

	if !IsFeed(data) {
		return nil, fmt.Errorf("Not an atom feed")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	var f Feed
	if err := decoder.Decode(&f); err != nil {
		return nil, fmt.Errorf("Failed to decode feed: %s", err)
	}
	if len(f.Entries) == 0 {
		return nil, fmt.Errorf("Feed has no entries")
	}
	return &f, nil
}

func IsFeed(data []byte) bool {

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if se, ok := token.(xml.StartElement); ok {
			return se.Name.Local == "feed"
		}
	}
	return false
}
