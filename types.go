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