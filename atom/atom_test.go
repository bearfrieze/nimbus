package atom

import (
	"fmt"
	"io/ioutil"
	"testing"
	"reflect"
)

func TestSlashdot(t *testing.T) {
	expected := Feed{
		Title:   "Slashdot",
		Updated: "2015-04-26T09:21:17+00:00",
		Entries: []Entry{
			Entry{
				Title: "Declassified Report From 2009 Questions Effectiveness of NSA Spying",
				ID:    "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
				Links: []Link{
					Link{
						Href: "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
					},
				},
				Updated: "2015-04-26T08:53:00+00:00",
			},
		},
	}
	tc := NewTestCase("slashdot", &expected, t)
	tc.Test(t)
}

func TestTheVerge(t *testing.T) {
	expected := Feed{
		Title:   "The Verge -  All Posts",
		Updated: "2015-04-26T02:01:02-04:00",
		Entries: []Entry{
			Entry{
				Title: "These old school electric bicycles look like a 1950s dream",
				ID:    "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
				Links: []Link{
					Link{
						Href: "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
						Rel:  "alternate",
					},
				},
				Published: "2015-04-26T02:01:02-04:00",
				Updated: "2015-04-26T02:01:02-04:00",
			},
		},
	}
	tc := NewTestCase("theverge", &expected, t)
	tc.Test(t)
}

func TestXKCD(t *testing.T) {
	expected := Feed{
		Title:   "xkcd.com",
		Updated: "2015-04-24T00:00:00Z",
		Entries: []Entry{
			Entry{
				Title: "Win by Induction",
				ID:    "http://xkcd.com/1516/",
				Links: []Link{
					Link{
						Href: "http://xkcd.com/1516/",
						Rel:  "alternate",
					},
				},
				Updated: "2015-04-24T00:00:00Z",
			},
		},
	}
	tc := NewTestCase("xkcd", &expected, t)
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
	tc.TestEntry(t)
	tc.TestLink(t)
}

func (tc TestCase) TestFeed(t *testing.T) {
	expect(tc.Actual.Title, tc.Expected.Title, t)
	expect(tc.Actual.Updated, tc.Expected.Updated, t)
}

func (tc TestCase) TestEntry(t *testing.T) {
	actual := tc.Actual.Entries[0]
	expected := tc.Expected.Entries[0]
	expect(actual.Title, expected.Title, t)
	expect(actual.ID, expected.ID, t)
	expect(actual.Updated, expected.Updated, t)
}

func (tc TestCase) TestLink(t *testing.T) {
	actual := tc.Actual.Entries[0].Links[0]
	expected := tc.Expected.Entries[0].Links[0]
	expect(actual.Href, expected.Href, t)
	expect(actual.Rel, expected.Rel, t)
}

// https://github.com/codegangsta/gin/blob/master/lib/helpers_test.go
func expect(a interface{}, e interface{}, t *testing.T) {
    if a != e {
        t.Errorf("Expected %v (type %v) - Got %v (type %v)", e, reflect.TypeOf(e), a, reflect.TypeOf(a))
    }
}
