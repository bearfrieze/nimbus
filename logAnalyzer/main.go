package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const (
	timeLayout = "2006/01/02 15:04:05.000000"
)

type Line struct {
	Time  time.Time
	Event string
	Url   string
}

type Poll map[string]time.Time

var polls = map[string][]Poll{}

func processLine(text string) {
	// 2015/07/06 13:13:10.320882 {"event":"cache","url":"http://feeds.kotaku.com.au/kotakuaustralia"}
	line := Line{}
	stamp := text[0:26] // First part of line is a timestamp
	data := text[27:]   // Second part of line is a json object
	line.Time, _ = time.Parse(timeLayout, stamp)
	json.Unmarshal([]byte(data), &line)
	if _, exists := polls[line.Url]; !exists {
		polls[line.Url] = make([]Poll, 0)
		polls[line.Url] = append(polls[line.Url], Poll{})
	}
	url := polls[line.Url]
	poll := url[len(url)-1]
	poll[line.Event] = line.Time
	if line.Event == "pollEnd" {
		polls[line.Url] = append(polls[line.Url], Poll{})
	}
}

func main() {
	path := os.Args[1]
	file, _ := os.Open(path)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		processLine(scanner.Text())
	}

	pairs := [][]string{
		{"poll", "pollEnd"},
		{"fetch", "fetchFail"},
		{"fetch", "save"},
		{"save", "saveFail"},
		{"save", "cache"},
		{"cache", "pollEnd"},
	}
	deltas := map[string][]int{}
	for _, pair := range pairs {
		join := fmt.Sprintf("%s->%s", pair[0], pair[1])
		deltas[join] = make([]int, 0)
		for url := range polls {
			for _, poll := range polls[url] {
				start, startExists := poll[pair[0]]
				stop, stopExists := poll[pair[1]]
				if !startExists || !stopExists {
					continue
				}
				delta := int(stop.Sub(start).Nanoseconds() / 1000000)
				if delta < 0 {
					continue
				}
				deltas[join] = append(deltas[join], delta)
			}
		}
		sort.Ints(deltas[join])
		sum := 0
		for _, delta := range deltas[join] {
			sum += delta
		}
		count := len(deltas[join])
		average := sum / count
		fmt.Printf("%s(%d)\n", join, count)
		fmt.Printf("av: %d\n", average)
		for i := 1; i < 4; i++ {
			fmt.Printf("q%d: %d\n", i, deltas[join][count*i/4])
		}
	}
}
