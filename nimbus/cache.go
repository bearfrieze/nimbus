package nimbus

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

type Cache struct {
	pool redis.Pool
}

func NewCache(server string) *Cache {

	// http://godoc.org/github.com/garyburd/redigo/redis#Pool
	pool := redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &Cache{pool: pool}
}

func (c Cache) Flush() {
	conn := c.pool.Get()
	defer conn.Close()
	log.Println("Flushing cache...")
	conn.Do("FLUSHDB")
	log.Println("Done flushing cache")
}

func (c Cache) Close() {
	c.pool.Close()
}

func (c Cache) SetFeed(url string, feed *Feed, queued bool) {
	conn := c.pool.Get()
	defer conn.Close()
	var value string
	if feed == nil {
		value = fmt.Sprintf("%b", queued)
	} else {
		marshalled, err := json.Marshal(feed)
		if err != nil {
			log.Printf("Unable to marshal feed '%s': %s", url, err)
			return
		}
		value = string(marshalled)
	}
	_, err := conn.Do("SET", url, value)
	if err != nil {
		log.Printf("Failed to set feed '%s': %s", url, err)
	}
}

func (c Cache) SetAlias(alias string, original string) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", "aliases", alias, original)
	if err != nil {
		log.Printf("Failed to set alias '%s' of '%s': %s", alias, original, err)
	}
}

func (c Cache) SetInvalid(url string) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SADD", "invalids", url)
	if err != nil {
		log.Printf("Failed to set invaid '%s': %s", url, err)
	}
}

func (c Cache) RemoveInvalid(url string) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SREM", "invalids", url)
	if err != nil {
		log.Printf("Failed to set invaid '%s': %s", url, err)
	}
}

func (c Cache) GetFeeds(urls []string) (map[string]*json.RawMessage, []string) {

	conn := c.pool.Get()
	defer conn.Close()

	response := make(map[string]*json.RawMessage)

	// Get aliases
	for _, url := range urls {
		conn.Send("HGET", "aliases", url)
	}
	conn.Flush()

	// Save aliases and check if invalid
	for i, _ := range urls {
		alias, _ := redis.String(conn.Receive())
		if alias != "" {
			urls[i] = alias
		}
		conn.Send("SISMEMBER", "invalids", urls[i])
	}
	conn.Flush()

	// Note which feeds are invalid and get the ones that aren't
	invalids := make(map[string]bool)
	for _, url := range urls {
		invalid, _ := redis.Int(conn.Receive())
		if invalid == 1 {
			invalids[url] = true
			continue
		}
		conn.Send("GET", url)
	}
	conn.Flush()

	missing := make([]string, 0)
	for _, url := range urls {
		var value string
		if _, exists := invalids[url]; exists {
			value = "false"
		} else {
			var err error
			value, err = redis.String(conn.Receive())
			if err != nil {
				value = "true"
				missing = append(missing, url)
			}
		}
		rm := json.RawMessage(value)
		response[url] = &rm
	}

	return response, missing
}
