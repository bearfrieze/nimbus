package rss

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestArsTechnica(t *testing.T) {
	expected := Feed{
		Channel: Channel{
			Title:         "Ars Technica",
			LastBuildDate: "Sun, 26 Apr 2015 03:49:47 +0000",
			Items: []Item{
				Item{
					Title:   "64-year-old engineer sues Google for age discrimination",
					GUID:    "https://arstechnica.com/?p=652799",
					Link:    "http://feeds.arstechnica.com/~r/arstechnica/index/~3/WIXuV8OMhBc/",
					PubDate: "Sat, 25 Apr 2015 17:00:03 +0000",
				},
			},
		},
	}
	tc := NewTestCase("arstechnica", &expected, t)
	tc.Test(t)
}

func TestXKCD(t *testing.T) {
	expected := Feed{
		Channel: Channel{
			Title: "xkcd.com",
			Items: []Item{
				Item{
					Title:   "Win by Induction",
					GUID:    "http://xkcd.com/1516/",
					Link:    "http://xkcd.com/1516/",
					PubDate: "Fri, 24 Apr 2015 04:00:00 -0000",
				},
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
	tc.TestChannel(t)
	tc.TestItem(t)
}

func (tc TestCase) TestChannel(t *testing.T) {
	a := tc.Actual.Channel
	e := tc.Expected.Channel
	expect(a.Title, e.Title, t)
	expect(a.TTL, e.TTL, t)
	expect(a.LastBuildDate, e.LastBuildDate, t)
	expect(a.PubDate, e.PubDate, t)
}

func (tc TestCase) TestItem(t *testing.T) {
	a := tc.Actual.Channel.Items[0]
	e := tc.Expected.Channel.Items[0]
	expect(a.Title, e.Title, t)
	expect(a.Link, e.Link, t)
	expect(a.PubDate, e.PubDate, t)
	expect(a.GUID, e.GUID, t)
}

// https://github.com/codegangsta/gin/blob/master/lib/helpers_test.go
func expect(a interface{}, e interface{}, t *testing.T) {
	if a != e {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", e, reflect.TypeOf(e), a, reflect.TypeOf(a))
	}
}
