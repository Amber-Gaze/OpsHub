package domain

import "time"

type ConfigHistory struct {
    Project   string
    Group     string
    Key       string
    Value     string
    Revision  int64
    CreatedAt time.Time
}
