package nimbus

import "time"

type Invalid struct {
	ID        int
	URL       string `sql:"unique_index"`
	Error     string
	CreatedAt time.Time
}
