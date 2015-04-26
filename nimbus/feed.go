package nimbus

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"github.com/bearfrieze/nimbus/atom"
	"github.com/bearfrieze/nimbus/rss"
	"github.com/kennygrant/sanitize"
	"time"
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
	URL         string    `json:"url"`
	GUID        string    `json:"guid" sql:"index"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

func (f Feed) Timeout() time.Duration {

	count := len(f.Items) - 1
	delta := f.Items[0].PublishedAt.Sub(f.Items[count].PublishedAt)
	frequency := time.Duration(delta / time.Duration(count))
	timeout := frequency / time.Duration(2)

	if timeout < minTimeout {
		return minTimeout
	} else if timeout > maxTimeout {
		return maxTimeout
	}

	return timeout
}

func NewFeed(url string, data []byte) (*Feed, error) {
	var f *Feed
	rf, re := rss.NewFeed(data)
	if re == nil {
		f = NewFeedFromRSS(rf)
	}
	af, ae := atom.NewFeed(data)
	if ae == nil {
		f = NewFeedFromAtom(af)
	}
	if f == nil {
		return nil, fmt.Errorf("Feed is not RSS: %s\nFeed is not Atom: %s\n", re, ae)
	}
	for key, item := range f.Items {
		item.Title = sanitize.HTML(item.Title)
		if item.GUID == "" {
			item.GUID = fmt.Sprintf(
				"%x:%x:%d",
				md5.Sum([]byte(url)),
				md5.Sum([]byte(item.Title)),
				item.PublishedAt.Unix(),
			)
			fmt.Println(item.GUID)
		}
		f.Items[key] = item
		f.UpdatedAt = time.Now()
	}
	f.URL = url
	f.NextPollAt = time.Now().Add(f.Timeout())
	f.UpdatedAt = time.Now()
	return f, nil
}

func NewFeedFromRSS(rf *rss.Feed) *Feed {

	rc := rf.Channel

	items := make([]Item, len(rc.Items))
	for key, ri := range rc.Items {
		items[key] = Item{
			Title:       ri.Title,
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
		items[key] = Item{
			Title:       entry.Title,
			URL:         entry.Links[0].Href,
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
