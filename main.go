package main

import (
	"encoding/json"
	"fmt"
	"github.com/bearfrieze/nimbus/nimbus"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	pollFrequency   = 15
	itemLimit       = 50
	workerCount     = 20
	invalidDuration = time.Hour * 24 * 7
)

var (
	ca      *nimbus.Cache
	db      *gorm.DB
	client  *http.Client
	queued  map[string]bool = make(map[string]bool)
	channel chan string     = make(chan string, workerCount)
)

func fetchFeed(url string) (*nimbus.Feed, error) {
	log.Printf("Fetching %s\n", url)
	r, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch %s: %s", url, err)
	}
	data, _ := ioutil.ReadAll(r.Body)
	return nimbus.NewFeed(url, data)
}

func saveFeed(feed *nimbus.Feed) error {

	dbFeed := nimbus.Feed{URL: feed.URL}
	dbDuplicate := nimbus.Feed{Sum: feed.Sum}
	dbFeedFound := !db.Where(&dbFeed).First(&dbFeed).RecordNotFound()
	dbDuplicateFound := !db.Where(&dbDuplicate).First(&dbDuplicate).RecordNotFound()

	if dbDuplicateFound && !(dbFeedFound && dbFeed.ID == dbDuplicate.ID) {
		createAlias(&dbFeed, &dbDuplicate, dbFeedFound)
		return fmt.Errorf("Duplicate %s found, alias created", dbDuplicate.URL)
	}

	if !dbFeedFound {
		log.Printf("Creating %s\n", feed.URL)
		db.Create(&feed)
		return nil
	}

	feed.ID = dbFeed.ID
	db.Omit("Items", "CreatedAt").Save(&feed)

	db.Model(&dbFeed).Related(&dbFeed.Items)

	// Compare items to existing items
	// Update existing items and create new ones
	dbItems := make(map[string]int, len(dbFeed.Items))
	for _, dbItem := range dbFeed.Items {
		dbItems[dbItem.GUID] = dbItem.ID
	}
	for _, item := range feed.Items {
		dbID, exists := dbItems[item.GUID]
		if !exists {
			item.FeedID = dbFeed.ID
			db.Create(&item)
			continue
		}
		item.ID = dbID
		db.Omit("GUID", "FeedID", "PublishedAt", "CreatedAt").Save(&item)
	}
	return nil
}

func createAlias(alias *nimbus.Feed, original *nimbus.Feed, delete bool) {

	if alias.URL == original.URL {
		return
	}
	log.Printf("Creating alias %s for %s\n", alias.URL, original.URL)
	db.Create(&nimbus.Alias{Alias: alias.URL, Original: original.URL})

	ca.SetAlias(alias.URL, original.URL)

	if delete {
		deleteFeed(alias)
	}
}

func deleteFeed(feed *nimbus.Feed) {
	log.Printf("Deleting %s\n", feed.URL)
	db.Where(&nimbus.Alias{Original: feed.URL}).Delete(nimbus.Alias{})
	db.Where(&nimbus.Item{FeedID: feed.ID}).Delete(nimbus.Item{})
	db.Delete(feed)
}

func worker() {
	for {
		url := <-channel
		delete(queued, url)
		pollFeed(url)
	}
}

func pollFeed(url string) {

	log.Printf("Polling %s\n", url)

	feed, err := fetchFeed(url)
	if err != nil {
		log.Printf("Marking %s as invalid: %s", url, err)
		db.Create(&nimbus.Invalid{URL: url, Error: err.Error()})
		db.Model(&nimbus.Feed{}).Where("url = ?", url).UpdateColumn("next_poll_at", time.Now().Add(invalidDuration))
		return
	}

	err = saveFeed(feed)
	if err != nil {
		log.Printf("Failed to save %s: %s\n", url, err)
		return
	}

	ca.SetFeed(getFeed(url))

	log.Printf("Polled %s, next poll at %v\n", url, feed.NextPollAt)
}

func queueFeed(url string) {
	if _, exists := queued[url]; exists {
		log.Printf("Already polling %s\n", url)
		return
	}
	queued[url] = true
	channel <- url
}

func pollFeeds(now *time.Time) {

	var urls []string
	var nextPoll = now.Add((pollFrequency + 1) * time.Second)
	db.Model(&nimbus.Feed{}).Where("next_poll_at < ?", nextPoll).Pluck("url", &urls)

	for _, url := range urls {
		go queueFeed(url)
	}
}

func cleanInvalid(now *time.Time) {
	var invalids []nimbus.Invalid
	aWeekAgo := time.Now().Add(-(invalidDuration - pollFrequency))
	db.Where("created_at < ?", aWeekAgo).Find(&invalids)
	for _, invalid := range invalids {
		db.Delete(&invalid)
		ca.RemoveInvalid(invalid.URL)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("Unsupported method '%s'\n", r.Method), 501)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var urls []string
	err := decoder.Decode(&urls)
	if err != nil {
		log.Printf("Unable to decode request: %s\n", err)
		http.Error(w, err.Error(), 400)
		return
	}

	response, missing := ca.GetFeeds(urls)
	json, err := json.Marshal(&response)
	if err != nil {
		log.Printf("Unable to marshal response: %s\n", err)
	}

	for _, url := range missing {
		go queueFeed(url)
	}

	w.Write(json)
}

func getFeed(url string) (*nimbus.Feed, bool) {
	if len(url) == 0 {
		return nil, false
	}
	feed := nimbus.Feed{URL: url}
	if db.Where(&feed).First(&feed).RecordNotFound() {
		invalid := nimbus.Invalid{URL: url}
		if !db.Where(&invalid).First(&invalid).RecordNotFound() {
			return nil, false
		}
		go queueFeed(url)
		return nil, true
	}
	db.Model(&feed).Order("published_at desc").Limit(itemLimit).Related(&feed.Items)
	return &feed, true
}

func newDb() *gorm.DB {

	args := fmt.Sprintf("sslmode=disable host=%s port=%s dbname=%s user=%s password=%s", os.Getenv("PGHOST"), os.Getenv("PGPORT"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"))
	log.Printf("Connecting to postgres: %s\n", args)
	db, err := gorm.Open("postgres", args)
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	db.DB()
	db.DB().SetMaxOpenConns(workerCount)
	db.DB().SetMaxIdleConns(workerCount / 2)
	db.SingularTable(true)
	db.AutoMigrate(&nimbus.Feed{}, &nimbus.Item{}, &nimbus.Alias{}, &nimbus.Invalid{})
	return &db
}

func fillCache() {

	log.Println("Filling cache with feeds...")
	var urls []string
	db.Model(&nimbus.Feed{}).Pluck("url", &urls)
	log.Printf("There are %d feeds", len(urls))
	for i, url := range urls {
		ca.SetFeed(getFeed(url))
		if i%100 == 0 {
			log.Printf("%d feeds filled into cache\n", i)
		}
	}
	log.Println("Done filling cache with feeds")

	log.Println("Filling cache with aliases...")
	var aliases []nimbus.Alias
	db.Find(&aliases)
	log.Printf("There are %d aliases", len(aliases))
	for _, alias := range aliases {
		ca.SetAlias(alias.Alias, alias.Original)
	}
	log.Println("Done filling cache with aliases")

	log.Println("Filling cache with invalids...")
	var invalids []nimbus.Invalid
	db.Find(&invalids)
	log.Printf("There are %d invalids", len(invalids))
	for _, invalid := range invalids {
		ca.SetInvalid(invalid.URL)
	}
	log.Println("Done filling cache with invalids")
}

func main() {

	db = newDb()
	defer db.Close()

	ca = nimbus.NewCache(fmt.Sprintf("%s:%s", os.Getenv("REDISHOST"), os.Getenv("REDISPORT")))
	defer ca.Close()
	ca.Flush()
	fillCache()

	// Make custom http client with timeout
	client = &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	// Start polling feeds
	go func() {
		for now := range time.Tick(pollFrequency * time.Second) {
			log.Printf("Queue length: %d\n", len(queued))
			go pollFeeds(&now)
			go cleanInvalid(&now)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	})

	port := os.Getenv("PORT")
	log.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
