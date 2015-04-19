package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"golang.org/x/net/html/charset"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	minTimeout = 60
	maxTimeout = 60 * 24
)

type Feed struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	ID    int
	Title string `xml:"title" json:"title"`
	URL   string `json:"url" sql:"unique_index"`
	Items []Item `xml:"item" json:"items"`
	TTL   int    `xml:"ttl" json:"ttl"`
}

type Item struct {
	ID        int
	ChannelID int     `sql:"index"`
	Title     string  `xml:"title" json:"title"`
	Link      string  `xml:"link" json:"link"`
	PubDate   string  `xml:"pubDate" json:"pubDate"`
	Unix      *int64  `json:"unix"`
	GUID      *string `xml:"guid" json:"guid" sql:"unique_index"`
}

func (c Channel) Frequency() int {

	var count, sum int64
	for i := 0; i < len(c.Items)-1; i++ {
		if c.Items[0].Unix == nil || c.Items[1].Unix == nil {
			continue
		}
		sum += *c.Items[0].Unix - *c.Items[1].Unix
		count++
	}
	if count == 0 {
		return minTimeout
	}
	return int(sum / count / 60)
}

func (c Channel) Timeout() int {

	if c.TTL != 0 {
		return c.TTL
	}
	if freq := c.Frequency() / 2; freq > minTimeout {
		if freq > maxTimeout {
			return maxTimeout
		}
		return freq
	}
	return minTimeout
}

func getChannel(url string) (*Channel, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf(`Failed to load channel "%s": %s`, url, err)
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	var f Feed
	if err := decoder.Decode(&f); err != nil {
		return nil, fmt.Errorf(`Failed to decode channel "%s": %s`, url, err)
	}

	c := f.Channel
	if c.Title == "" || len(c.Items) <= 0 {
		return nil, fmt.Errorf(`Channel "%s" is invalid`, url)
	}
	c.URL = url

	// Add GUID's
	for key, item := range c.Items {
		if item.GUID == nil {
			guid := fmt.Sprintf("%s:%s:%s", url, item.PubDate, item.Title)
			c.Items[key].GUID = &guid
		}
	}

	// Add Unix's
	for key, item := range c.Items {
		t, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			log.Printf(`Failed to parse time for channel "%s": %s\n`, url, err)
			break
		}
		unix := t.Unix()
		c.Items[key].Unix = &unix
	}

	return &c, nil
}

func saveChannel(channel *Channel, db *gorm.DB) {

	dbChannel := Channel{URL: channel.URL}
	db.Where(&dbChannel).First(&dbChannel)
	db.Model(&dbChannel).Related(&dbChannel.Items)

	fmt.Println(channel.URL)
	for _, dbItem := range dbChannel.Items {
		fmt.Printf("%s\n", *dbItem.GUID)
	}

	// Compare items to existing items
	// Update existing items and create new ones
	dbItems := make(map[string]*Item, len(dbChannel.Items))
	for _, dbItem := range dbChannel.Items {
		dbItems[*dbItem.GUID] = &dbItem
	}
	for _, item := range channel.Items {
		dbItem, exists := dbItems[*item.GUID]
		if !exists {
			item.ChannelID = dbChannel.ID
			db.Create(&item)
			dbChannel.Items = append(dbChannel.Items, item)
			continue
		}
		item.ID = dbItem.ID
		db.Omit("GUID").Save(&item)
	}
}

func pollChannel(url string, db *gorm.DB, create bool) {

	channel, err := getChannel(url)
	if err != nil {
		log.Println(err)
		return
	}

	if create {
		db.Create(channel)
	} else {
		saveChannel(channel, db)
	}

	timeout := channel.Timeout()
	log.Printf("Polled: %s, Timeout: %d\n", url, timeout)
	time.Sleep(time.Duration(timeout) * time.Minute)
	pollChannel(url, db, false)
}

func handler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {

	if r.Method == "GET" {
		fmt.Fprintf(w, "Ping!\n")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var urls []string
	err := decoder.Decode(&urls)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	channels := make([]Channel, 0, len(urls))
	for _, url := range urls {
		channel := Channel{}
		if db.Where(Channel{URL: url}).First(&channel).RecordNotFound() {
			go pollChannel(url, db, true)
			log.Printf("Polling initiated: %s\n", url)
			continue
		}
		db.Model(&channel).Related(&channel.Items)
		channels = append(channels, channel)
	}

	json, err := json.Marshal(channels)
	if err != nil {
		log.Printf("Unable to marshal channels: %s\n", err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, string(json))
}

func getDB() *gorm.DB {

	args := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"), os.Getenv("PGDATABASE"), os.Getenv("PGHOST"), os.Getenv("PGPORT"))
	log.Printf("Connecting to postgres: %s\n", args)
	db, err := gorm.Open("postgres", args)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	db.DB()
	db.SingularTable(true)
	db.AutoMigrate(&Channel{}, &Item{})
	return &db
}

func main() {

	db := getDB()

	// Start polling channels
	var urls []string
	db.Model(&Channel{}).Pluck("URL", &urls)
	for _, url := range urls {
		go pollChannel(url, db, false)
		log.Printf("Polling initiated: %s\n", url)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, db)
	})

	port := os.Getenv("PORT")
	fmt.Printf("Cumulonimbus is listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
