package stats

import (
    "time"
)

type Stats interface {
    StatsTime()     time.Time
    StatsFields()   map[string]interface{}
    String()        string
}

type StatsSource interface {
    GiveStats(interval time.Duration)     chan Stats
}
