package nimbus

import "time"

type Alias struct {
	ID        int
	Alias     string `sql:"unique_index"`
	Original  string `sql:"index"`
	CreatedAt time.Time
}
