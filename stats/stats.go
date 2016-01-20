package stats

import (
    "time"
)

type ID struct {
    Type        string
    Hostname    string
    Instance    string
}

type Stats interface {
    StatsID()       ID
    StatsTime()     time.Time
    StatsFields()   map[string]interface{}
    String()        string
}
