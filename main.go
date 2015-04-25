package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/kennygrant/sanitize"
	_ "github.com/lib/pq"
	"golang.org/x/net/html/charset"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	minTimeout    = 60
	maxTimeout    = 60 * 24
	pollFrequency = 15
)

var (
	formats = []string{time.RFC822, time.RFC822Z, time.RFC1123, time.RFC1123Z}
)

func (c Channel) Frequency() int {

	var count, sum int64
	for i := 0; i < len(c.Items)-1; i++ {
		if c.Items[i].Unix == nil || c.Items[i+1].Unix == nil {
			continue
		}
		sum += *c.Items[i].Unix - *c.Items[i+1].Unix
		count++
	}
	// Avoid division by zero...
	if count == 0 {
		return minTimeout
	}
	return int(sum / count / 60)
}

func (c Channel) Timeout() int {

	timeout := 0
	if c.TTL != 0 {
		timeout = c.TTL
	} else {
		timeout = c.Frequency() / 2
	}

	if timeout < minTimeout {
		return minTimeout
	} else if timeout > maxTimeout {
		return maxTimeout
	}

	return timeout
}

func fetchChannel(url string) (*Channel, error) {

	r, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to load channel '%s': %s", url, err)
	}
	data, _ := ioutil.ReadAll(r.Body)

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	var f Feed
	if err := decoder.Decode(&f); err != nil {
		return nil, fmt.Errorf("Failed to decode channel '%s': %s", url, err)
	}

	c := f.Channel
	if c.Title == "" || len(c.Items) <= 0 {
		return nil, fmt.Errorf("Invalid channel '%s'", url)
	}

	for key, item := range c.Items {

		// Add GUIDs if missing
		if item.GUID == nil {
			guid := fmt.Sprintf("%s:%s:%s", url, item.PubDate, item.Title)
			item.GUID = &guid
		}

		// Enrich publication dates
		if item.PubDate == "" {
			item.PubDate = time.Now().Format(time.RFC1123Z)
		}

		// Add unix timestamps
		var t time.Time
		for _, format := range formats {
			t, err = time.Parse(format, item.PubDate)
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Printf("Failed to parse time for channel '%s': %s\n", url, err)
		}
		unix := t.Unix()
		item.Unix = &unix

		// Sanitize text
		fmt.Println(item.Title)
		item.Title = sanitize.HTML(item.Title)
		fmt.Println(item.Title)

		c.Items[key] = item
	}

	c.URL = url
	c.Sum = fmt.Sprintf("%x", sha256.Sum256(data))
	c.PollAt = time.Now().Add(time.Duration(c.Timeout()) * time.Minute)

	return &c, nil
}

func saveChannel(channel *Channel, db *gorm.DB) error {

	dbChannel := Channel{URL: channel.URL}
	dbDuplicate := Channel{Sum: channel.Sum}
	dbChannelFound := !db.Where(&dbChannel).First(&dbChannel).RecordNotFound()
	dbDuplicateFound := !db.Where(&dbDuplicate).First(&dbDuplicate).RecordNotFound()

	if dbDuplicateFound && !(dbChannelFound && dbChannel.ID == dbDuplicate.ID) {
		createAlias(&dbChannel, &dbDuplicate, dbChannelFound, db)
		return fmt.Errorf("Duplicate channel '%s' found, alias created", dbDuplicate.URL)
	}

	if !dbChannelFound {
		log.Printf("Creating channel '%s'\n", channel.URL)
		db.Create(&channel)
		return nil
	}

	channel.ID = dbChannel.ID
	channel.UpdatedAt = time.Now()
	db.Omit("Items", "CreatedAt").Save(&channel)

	db.Model(&dbChannel).Related(&dbChannel.Items)

	// Compare items to existing items
	// Update existing items and create new ones
	dbItems := make(map[string]int, len(dbChannel.Items))
	for _, dbItem := range dbChannel.Items {
		dbItems[*dbItem.GUID] = dbItem.ID
	}
	for _, item := range channel.Items {
		dbID, exists := dbItems[*item.GUID]
		if !exists {
			item.ChannelID = dbChannel.ID
			db.Create(&item)
			continue
		}
		item.ID = dbID
		item.UpdatedAt = time.Now()
		db.Omit("GUID", "ChannelID", "PubDate", "Unix", "CreatedAt").Save(&item)
	}
	return nil
}

func createAlias(alias *Channel, original *Channel, delete bool, db *gorm.DB) {
	if alias.URL == original.URL {
		return
	}
	log.Printf("Creating alias '%s' for '%s'\n", alias.URL, original.URL)
	db.Create(&Alias{Alias: alias.URL, Original: original.URL})
	if delete {
		deleteChannel(alias, db)
	}
}

func deleteChannel(channel *Channel, db *gorm.DB) {
	log.Printf("Deleting channel '%s'\n", channel.URL)
	db.Where(&Alias{Original: channel.URL}).Delete(Alias{})
	db.Where(&Item{ChannelID: channel.ID}).Delete(Item{})
	db.Delete(channel)
}

func pollChannel(url string, db *gorm.DB) {

	channel, err := fetchChannel(url)
	if err != nil {
		db.Create(&Invalid{URL: url, Error: err.Error()})
		log.Printf("Marked '%s' as invalid: %s", url, err)
		return
	}

	err = saveChannel(channel, db)
	if err != nil {
		log.Printf("Failed to save channel '%s': %s\n", url, err)
		return
	}

	log.Printf("Polled '%s', next poll at %v\n", url, channel.PollAt)
}

func pollChannels(now *time.Time, db *gorm.DB) {

	var urls []string
	var nextPoll = now.Add((pollFrequency + 1) * time.Second)
	db.Model(&Channel{}).Where("poll_at < ?", nextPoll).Pluck("URL", &urls)

	for _, url := range urls {
		go pollChannel(url, db)
	}
}

func getChannel(url string, db *gorm.DB, repeat bool) (*Channel, bool) {
	channel := Channel{URL: url}
	if db.Where(&channel).First(&channel).RecordNotFound() {
		alias := Alias{Alias: url}
		if !repeat && !db.Where(&alias).First(&alias).RecordNotFound() {
			return getChannel(alias.Original, db, true)
		}
		invalid := Invalid{URL: url}
		if !db.Where(&invalid).First(&invalid).RecordNotFound() {
			return nil, false
		}
		go pollChannel(url, db)
		log.Printf("Started polling '%s'\n", url)
		return nil, true
	}
	db.Model(&channel).Related(&channel.Items)
	return &channel, true
}

func handler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {

	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf("Unsupported method '%s'\n", r.Method), 501)
		return
	}

	url := r.URL.Query().Get("url")
	channel, polling := getChannel(url, db, false)
	if channel == nil {
		fmt.Fprintf(w, "%t\n", polling)
		return
	}

	json, err := json.Marshal(&channel)
	if err != nil {
		log.Printf("Unable to marshal channel '%s': %s\n", url, err)
	}

	fmt.Fprintln(w, string(json))
}

func getDB() *gorm.DB {

	args := fmt.Sprintf("sslmode=disable host=%s port=%s dbname=%s user=%s password=%s", os.Getenv("PGHOST"), os.Getenv("PGPORT"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"))
	log.Printf("Connecting to postgres: %s\n", args)
	db, err := gorm.Open("postgres", args)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	db.DB()
	db.DB().Ping()
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	db.SingularTable(true)
	db.AutoMigrate(&Channel{}, &Item{}, &Alias{}, &Invalid{})
	return &db
}

func main() {

	db := getDB()

	// Start polling channels
	go func() {
		for now := range time.Tick(pollFrequency * time.Second) {
			go pollChannels(&now, db)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, db)
	})

	port := os.Getenv("PORT")
	log.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
