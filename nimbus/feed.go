package nimbus

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"github.com/bearfrieze/nimbus/atom"
	"github.com/bearfrieze/nimbus/rss"
	"github.com/kennygrant/sanitize"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	minTimeout = time.Hour
	maxTimeout = time.Hour * time.Duration(24)
)

var (
	timeFormats = []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
	}
	rexRepeatWhitespace = regexp.MustCompile(`\s\s+`)
)

type Feed struct {
	ID         int       `json:"-"`
	Title      string    `json:"title"`
	URL        string    `json:"url" sql:"unique_index"`
	Items      []Item    `json:"items"`
	Sum        string    `json:"-" sql:"index"`
	NextPollAt time.Time `json:"next_poll_at" sql:"index"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Item struct {
	ID          int       `json:"-"`
	FeedID      int       `json:"-" sql:"index"`
	Title       string    `json:"title"`
	Teaser      string    `json:"teaser" sql:"type:text"`
	URL         string    `json:"url"`
	GUID        string    `json:"guid" sql:"index"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

func (f Feed) Timeout() time.Duration {

	var timeout time.Duration

	count := len(f.Items) - 1
	if count > 0 {
		delta := f.Items[0].PublishedAt.Sub(f.Items[count].PublishedAt)
		frequency := time.Duration(delta / time.Duration(count))
		timeout = frequency / time.Duration(2)
	}

	if timeout < minTimeout {
		return minTimeout
	} else if timeout > maxTimeout {
		return maxTimeout
	}

	return timeout
}

func cleanText(text string) string {
	text = sanitize.HTML(text)
	text = strings.Replace(text, "\n", " ", -1)
	text = rexRepeatWhitespace.ReplaceAllLiteralString(text, "")
	if !utf8.ValidString(text) {
		text = ""
	}
	return strings.TrimSpace(text)
}

func limitStringLength(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}

func NewFeed(url string, data []byte) (*Feed, error) {
	var f *Feed
	f, err := NewFeedFromUnknown(data)
	if err != nil {
		return nil, err
	}
	for key, item := range f.Items {
		item.Title = limitStringLength(cleanText(item.Title), 255)
		item.URL = limitStringLength(item.URL, 255)
		item.Teaser = limitStringLength(cleanText(item.Teaser), 1000)
		if item.GUID == "" {
			item.GUID = fmt.Sprintf(
				"%x:%x:%d",
				md5.Sum([]byte(url)),
				md5.Sum([]byte(item.Title)),
				item.PublishedAt.Unix(),
			)
		}
		f.Items[key] = item
		f.UpdatedAt = time.Now()
	}
	f.URL = limitStringLength(url, 255)
	f.NextPollAt = time.Now().Add(f.Timeout())
	f.UpdatedAt = time.Now()
	return f, nil
}

func NewFeedFromUnknown(data []byte) (*Feed, error) {
	rf, re := rss.NewFeed(data)
	if re == nil {
		return NewFeedFromRSS(rf), nil
	}
	af, ae := atom.NewFeed(data)
	if ae == nil {
		return NewFeedFromAtom(af), nil
	}
	return nil, fmt.Errorf("Feed is not RSS: %s, Feed is not Atom: %s", re, ae)
}

func NewFeedFromRSS(rf *rss.Feed) *Feed {

	rc := rf.Channel

	items := make([]Item, len(rc.Items))
	for key, ri := range rc.Items {
		items[key] = Item{
			Title:       ri.Title,
			Teaser:      ri.Description,
			URL:         ri.Link,
			GUID:        ri.GUID,
			PublishedAt: PublishedAt([]string{ri.PubDate}),
		}
	}

	return &Feed{
		Title: rf.Channel.Title,
		Items: items,
		Sum:   Sum(rf.Raw),
	}
}

func NewFeedFromAtom(af *atom.Feed) *Feed {

	items := make([]Item, len(af.Entries))
	for key, entry := range af.Entries {
		var url string
		if len(entry.Links) > 0 {
			url = entry.Links[0].Href
		}
		var teaser string
		if len(entry.Summary) > 0 {
			teaser = entry.Summary
		} else {
			teaser = entry.Content
		}
		items[key] = Item{
			Title:       entry.Title,
			Teaser:      teaser,
			URL:         url,
			GUID:        entry.ID,
			PublishedAt: PublishedAt([]string{entry.Published, entry.Updated}),
		}
	}

	return &Feed{
		Title: af.Title,
		Items: items,
		Sum:   Sum(af.Raw),
	}
}

func PublishedAt(ss []string) time.Time {
	for _, s := range ss {
		for _, f := range timeFormats {
			if t, err := time.Parse(f, s); err == nil {
				return t
			}
		}
	}
	return time.Now()
}

func Sum(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
