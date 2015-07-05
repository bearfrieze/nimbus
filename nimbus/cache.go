package nimbus

import (
	"encoding/json"
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
		MaxIdle:     20,
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

func (c Cache) Set(url string, value string) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", url, value)
	if err != nil {
		log.Printf("Failed to set feed '%s': %s", url, err)
	}
}

func (c Cache) Expire(url string, seconds int) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("EXPIRE", url, seconds)
	if err != nil {
		log.Printf("Failed to expire feed '%s': %s", url, err)
	}
}

func (c Cache) SetFeed(url string, feed *Feed) {
	marshalled, err := json.Marshal(feed)
	if err != nil {
		log.Printf("Unable to marshal feed '%s': %s", url, err)
		return
	}
	c.Set(url, string(marshalled))
}

func (c Cache) SetAlias(alias string, original string) {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", "aliases", alias, original)
	if err != nil {
		log.Printf("Failed to set alias '%s' of '%s': %s", alias, original, err)
	}
}

func (c Cache) GetFeeds(urls []string) (map[string]*json.RawMessage, []string) {

	conn := c.pool.Get()
	defer conn.Close()

	response := make(map[string]*json.RawMessage)

	for _, url := range urls {
		conn.Send("HGET", "aliases", url)
	}
	conn.Flush()

	for i, _ := range urls {
		alias, _ := redis.String(conn.Receive())
		if alias != "" {
			urls[i] = alias
		}
		conn.Send("GET", urls[i])
	}
	conn.Flush()

	missing := make([]string, 0)
	for _, url := range urls {
		value, err := redis.String(conn.Receive())
		if err != nil {
			value = "true"
			missing = append(missing, url)
		}
		rm := json.RawMessage(value)
		response[url] = &rm
	}

	return response, missing
}
