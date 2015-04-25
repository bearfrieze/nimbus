package main

import(
    "time"
)

type Feed struct {
    Channel Channel `xml:"channel"`
}

type Channel struct {
    ID        int       `json:"-"`
    Title     string    `xml:"title" json:"title"`
    URL       string    `json:"url" sql:"unique_index"`
    Items     []Item    `xml:"item" json:"items"`
    TTL       int       `xml:"ttl" json:"-"`
    Sum       string    `json:"-" sql:"index"`
    PollAt    time.Time `json:"poll_at" sql:"index"`
    CreatedAt time.Time `json:"-"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Item struct {
    ID        int       `json:"-"`
    ChannelID int       `json:"-" sql:"index"`
    Title     string    `xml:"title" json:"title"`
    Link      string    `xml:"link" json:"link"`
    PubDate   string    `xml:"pubDate" json:"pubDate"`
    Unix      *int64    `json:"unix"`
    GUID      *string   `xml:"guid" json:"guid" sql:"index"`
    CreatedAt time.Time `json:"-"`
    UpdatedAt time.Time `json:"-"`
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