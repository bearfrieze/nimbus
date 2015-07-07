package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
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
	p, _ := plot.New()
	p.Add(plotter.NewGrid())
	keys := make([]string, 0)
	for i, pair := range pairs {
		key := fmt.Sprintf("%s->%s", pair[0], pair[1])
		keys = append(keys, key)
		deltas := make([]float64, 0)
		for url := range polls {
			for _, poll := range polls[url] {
				start, startExists := poll[pair[0]]
				stop, stopExists := poll[pair[1]]
				if !startExists || !stopExists {
					continue
				}
				delta := stop.Sub(start).Seconds()
				if delta < 0 {
					continue
				}
				deltas = append(deltas, delta)
			}
		}
		fmt.Printf("%s(%d)\n", key, len(deltas))

		if len(deltas) == 0 {
			continue
		}
		sort.Float64s(deltas)
		count := float64(len(deltas))
		var sum float64
		for _, delta := range deltas {
			sum += delta
		}
		fmt.Printf("av: %f\n", sum/count)
		for i := 1; i < 4; i++ {
			fmt.Printf("q%d: %f\n", i, deltas[len(deltas)*i/4])
		}

		selection := deltas[int(count*0.1):int(count*0.9)] // Unfortunate but necessary step, too many outliers
		b, _ := plotter.MakeHorizBoxPlot(vg.Points(20), float64(i), plotter.Values(selection))
		p.Add(b)
	}

	p.NominalY(keys...)
	if err := p.Save(10*vg.Inch, 5*vg.Inch, "boxplot.png"); err != nil {
		panic(err)
	}
}
