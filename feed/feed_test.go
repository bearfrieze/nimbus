package feed

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

func TestRSSArsTechnica(t *testing.T) {
	expected := Feed{
		Title: "Ars Technica",
		Items: []Item{
			Item{
				Title:       "64-year-old engineer sues Google for age discrimination",
				GUID:        "https://arstechnica.com/?p=652799",
				URL:         "http://feeds.arstechnica.com/~r/arstechnica/index/~3/WIXuV8OMhBc/",
				PublishedAt: time.Date(2015, 04, 25, 17, 0, 3, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("rss/arstechnica", &expected, t)
	tc.Test(t)
}

func TestRSSXKCD(t *testing.T) {
	expected := Feed{
		Title: "xkcd.com",
		Items: []Item{
			Item{
				Title:       "Win by Induction",
				GUID:        "http://xkcd.com/1516/",
				URL:         "http://xkcd.com/1516/",
				PublishedAt: time.Date(2015, 04, 24, 04, 0, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("rss/xkcd", &expected, t)
	tc.Test(t)
}

func TestAtomSlashdot(t *testing.T) {
	expected := Feed{
		Title: "Slashdot",
		Items: []Item{
			Item{
				Title:       "Declassified Report From 2009 Questions Effectiveness of NSA Spying",
				GUID:        "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
				URL:         "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
				PublishedAt: time.Date(2015, 04, 26, 8, 53, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("atom/slashdot", &expected, t)
	tc.Test(t)
}

func TestAtomTheVerge(t *testing.T) {
	expected := Feed{
		Title: "The Verge -  All Posts",
		Items: []Item{
			Item{
				Title:       "These old school electric bicycles look like a 1950s dream",
				GUID:        "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
				URL:         "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
				PublishedAt: time.Date(2015, 04, 26, 2, 1, 2, 0, time.FixedZone("UTC", -4*60*60)),
			},
		},
	}
	tc := NewTestCase("atom/theverge", &expected, t)
	tc.Test(t)
}

func TestAtomXKCD(t *testing.T) {
	expected := Feed{
		Title: "xkcd.com",
		Items: []Item{
			Item{
				Title:       "Win by Induction",
				GUID:        "http://xkcd.com/1516/",
				URL:         "http://xkcd.com/1516/",
				PublishedAt: time.Date(2015, 04, 24, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("atom/xkcd", &expected, t)
	tc.Test(t)
}

type TestCase struct {
	Actual   *Feed
	Expected *Feed
}

func NewTestCase(name string, expected *Feed, t *testing.T) TestCase {
	path := fmt.Sprintf("test_fixtures/%s.xml", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read data: %s", err)
	}
	actual, err := NewFeed(data)
	if err != nil {
		t.Errorf("Failed to decode data: %s", err)
	}
	return TestCase{actual, expected}
}

func (tc TestCase) Test(t *testing.T) {
	tc.TestFeed(t)
	tc.TestItem(t)
}

func (tc TestCase) TestFeed(t *testing.T) {
	expect(tc.Actual.Title, tc.Expected.Title, t)
}

func (tc TestCase) TestItem(t *testing.T) {
	a := tc.Actual.Items[0]
	e := tc.Expected.Items[0]
	expect(a.Title, e.Title, t)
	expect(a.URL, e.URL, t)
	expect(a.GUID, e.GUID, t)
	expect(a.PublishedAt.Unix(), e.PublishedAt.Unix(), t)
}

// https://github.com/codegangsta/gin/blob/master/lib/helpers_test.go
func expect(a interface{}, e interface{}, t *testing.T) {
	if a != e {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", e, reflect.TypeOf(e), a, reflect.TypeOf(a))
	}
}
