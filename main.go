package main

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"golang.org/x/net/html/charset"
	"io/ioutil"
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
	ID        int
	Title     string `xml:"title" json:"title"`
	URL       string `json:"url" sql:"unique_index"`
	Items     []Item `xml:"item" json:"items"`
	TTL       int    `xml:"ttl" json:"ttl"`
	Sum       string `sql:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Item struct {
	ID        int
	ChannelID int     `sql:"index"`
	Title     string  `xml:"title" json:"title"`
	Link      string  `xml:"link" json:"link"`
	PubDate   string  `xml:"pubDate" json:"pubDate"`
	Unix      *int64  `json:"unix"`
	GUID      *string `xml:"guid" json:"guid" sql:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Alias struct {
	ID        int
	Alias     string `sql:"unique_index"`
	Original  string `sql:"index"`
	CreatedAt time.Time
}

type Invalid struct {
	ID        int
	URL       string `sql:"unique_index"`
	Error     string
	CreatedAt time.Time
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

func fetchChannel(url string) (*Channel, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to load channel '%s': %s", url, err)
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	var f Feed
	if err := decoder.Decode(&f); err != nil {
		return nil, fmt.Errorf("Failed to decode channel '%s': %s", url, err)
	}

	c := f.Channel
	if c.Title == "" || len(c.Items) <= 0 {
		return nil, fmt.Errorf("Invalid channel '%s'", url)
	}

	// Enrich
	c.URL = url
	data, _ := ioutil.ReadAll(resp.Body)
	c.Sum = fmt.Sprintf("%x", sha256.Sum256(data))
	for key, item := range c.Items {
		if item.GUID == nil {
			guid := fmt.Sprintf("%s:%s:%s", url, item.PubDate, item.Title)
			item.GUID = &guid
		}
		c.Items[key] = item
	}
	for key, item := range c.Items {
		if item.PubDate == "" {
			item.PubDate = time.Now().Format(time.RFC1123Z)
		}
		t, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			log.Printf("Failed to parse time for channel '%s': %s\n", url, err)
			break
		}
		unix := t.Unix()
		item.Unix = &unix
		c.Items[key] = item
	}

	return &c, nil
}

func saveChannel(channel *Channel, db *gorm.DB) error {

	dbChannel := Channel{URL: channel.URL}
	dbDuplicate := Channel{Sum: channel.Sum}
	dbChannelFound := !db.Where(&dbChannel).First(&dbChannel).RecordNotFound()
	dbDuplicateFound := !db.Where(&dbDuplicate).First(&dbDuplicate).RecordNotFound()

	if dbDuplicateFound {
		createAlias(&dbChannel, &dbDuplicate, dbChannelFound, db)
		return fmt.Errorf("Duplicate channel '%s' found, alias created", dbDuplicate.URL)
	}

	if !dbChannelFound {
		log.Printf("Creating channel '%s'\n", channel.URL)
		db.Create(&channel)
		return nil
	}

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
		db.Omit("GUID", "ChannelID").Save(&item)
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

	timeout := channel.Timeout()
	log.Printf("Polled '%s', %d minutes until next poll\n", url, timeout)
	time.Sleep(time.Duration(timeout) * time.Minute)
	pollChannel(url, db)
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

	json, err := json.Marshal(channel)
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
	db.SingularTable(true)
	db.AutoMigrate(&Channel{}, &Item{}, &Alias{}, &Invalid{})
	return &db
}

func main() {

	db := getDB()

	// Start polling channels
	var urls []string
	db.Model(&Channel{}).Pluck("URL", &urls)
	for _, url := range urls {
		go pollChannel(url, db)
		log.Printf("Started polling '%s'\n", url)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, db)
	})

	port := os.Getenv("PORT")
	log.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
