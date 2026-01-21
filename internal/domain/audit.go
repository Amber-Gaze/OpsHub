package domain

import "time"

type Audit struct {
    User      string
    Action    string
    CreatedAt time.Time
}
