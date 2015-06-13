package nimbus

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
				Teaser:      "Suit says median age at Google is 29, way below national averages.",
				GUID:        "https://arstechnica.com/?p=652799",
				URL:         "http://feeds.arstechnica.com/~r/arstechnica/index/~3/WIXuV8OMhBc/",
				PublishedAt: time.Date(2015, 04, 25, 17, 0, 3, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("http://feeds.arstechnica.com/arstechnica/index?format=xml", "rss/arstechnica", &expected, t)
	tc.Test(t)
}

func TestRSSXKCD(t *testing.T) {
	expected := Feed{
		Title: "xkcd.com",
		Items: []Item{
			Item{
				Title:       "Win by Induction",
				Teaser:      "",
				GUID:        "http://xkcd.com/1516/",
				URL:         "http://xkcd.com/1516/",
				PublishedAt: time.Date(2015, 04, 24, 04, 0, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("http://xkcd.com/rss.xml", "rss/xkcd", &expected, t)
	tc.Test(t)
}

func TestAtomSlashdot(t *testing.T) {
	expected := Feed{
		Title: "Slashdot",
		Items: []Item{
			Item{
				Title:       "Declassified Report From 2009 Questions Effectiveness of NSA Spying",
				Teaser:      `schwit1 writes: With debate gearing up over the coming expiration of the Patriot Act surveillance law, the Obama administration on Saturday unveiled a 6-year-old report examining the once-secret program code-named Stellarwind, which collected information on Americans' calls and emails. The report was from the inspectors general of various intelligence and law enforcement agencies. They found that while many senior intelligence officials believe the program filled a gap by increasing access to international communications, others including FBI agents, CIA analysts and managers "had difficulty evaluating the precise contribution of the [the surveillance system] to counterterrorism efforts because it was most often viewed as one source among many available analytic and intelligence-gathering tools in these efforts." "The report said that the secrecy surrounding the program made it less useful. Very few working-level C.I.A. analysts were told about it. ... Another part of the newly disclosed report provides an explanation for a change in F.B.I. rules during the Bush administration. Previously, F.B.I. agents had only two types of cases: "preliminary" and "full" investigations. But the Bush administration created a third, lower-level type called an "assessment." This development, it turns out, was a result of Stellarwind.Read more of this story at Slashdot.`,
				GUID:        "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
				URL:         "http://news.slashdot.org/story/15/04/26/0347222/declassified-report-from-2009-questions-effectiveness-of-nsa-spying?utm_source=atom1.0mainlinkanon&utm_medium=feed",
				PublishedAt: time.Date(2015, 04, 26, 8, 53, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("http://rss.slashdot.org/slashdot/slashdotMainatom?format=xml", "atom/slashdot", &expected, t)
	tc.Test(t)
}

func TestAtomTheVerge(t *testing.T) {
	expected := Feed{
		Title: "The Verge -  All Posts",
		Items: []Item{
			Item{
				Title:       "These old school electric bicycles look like a 1950s dream",
				Teaser:      `For cycling enthusiasts whose interests fall somewhere between a 10-speed and a motorcycle, electric bicycles are a pretty solid alternative. Unfortunately, sometimes e-bikes look like this. But these new retro-styled ones from Vintage Electric look like something an extra in Jaws would ride around before all the bad stuff stars happening.The bikes are from the Cruz line: they're kind of campy, kind of beachy, and vintage in style only. Vintage Electric claims the bikes can reach a speed of 36 MPH in "Race Mode" with a range of 30 miles and a recharge time of just two hours. The company says the 3,000 watt, 3-phase brushless motor and 52 volt battery should last around 30,000 miles. The bicycles are available now for...Continue reading…`,
				GUID:        "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
				URL:         "http://www.theverge.com/2015/4/26/8495991/electric-bicycles-vintage-electric-cruz",
				PublishedAt: time.Date(2015, 04, 26, 2, 1, 2, 0, time.FixedZone("UTC", -4*60*60)),
			},
		},
	}
	tc := NewTestCase("http://www.theverge.com/rss/full.xml", "atom/theverge", &expected, t)
	tc.Test(t)
}

func TestAtomXKCD(t *testing.T) {
	expected := Feed{
		Title: "xkcd.com",
		Items: []Item{
			Item{
				Title:       "Win by Induction",
				Teaser:      "",
				GUID:        "http://xkcd.com/1516/",
				URL:         "http://xkcd.com/1516/",
				PublishedAt: time.Date(2015, 04, 24, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	tc := NewTestCase("http://xkcd.com/atom.xml", "atom/xkcd", &expected, t)
	tc.Test(t)
}

type TestCase struct {
	Actual   *Feed
	Expected *Feed
}

func NewTestCase(url string, name string, expected *Feed, t *testing.T) TestCase {
	path := fmt.Sprintf("test_fixtures/%s.xml", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read data: %s", err)
	}
	actual, err := NewFeed(url, data)
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
	expect(a.Teaser, e.Teaser, t)
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
