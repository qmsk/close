package stats

import (
    "time"
)

/* StatsWriter identifier; this uniquely identifies an InfluxDB measurement series, which may include multiple fields */
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
