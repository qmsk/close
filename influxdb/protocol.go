package influxdb

import (
    "time"
    "fmt"
    "strings"
)

type point struct {
    measurement string
    tags        map[string]string
    fields      map[string]string
    timestamp   time.Time
}

func CreatePoint(measurement string) (*point) {
    return &point{
        measurement: measurement,
        timestamp: time.Now(),
    }
}

func (p *point) AddTag(key string, value string) {
    if p.tags == nil {
        p.tags = make(map[string]string)
    }
    p.tags[key] = value
}

func (p *point) AddField(key string, value string) {
    if p.fields == nil {
        p.fields = make(map[string]string)
    }
    p.fields[key] = value
}

func (p point) Serialize() string {
    tags := []string{}
    for k, v := range p.tags {
        tags = append(tags, fmt.Sprintf("%s=%s", k, v))
    }
    tagsRes := strings.Join(tags, ",")

    fields := []string{}
    for k, v := range p.fields {
        fields = append(fields, fmt.Sprintf("%s=%s", k, v))
    }
    fieldsRes := strings.Join(fields, ",")

    return fmt.Sprintf("%s %s %s %d", p.measurement, tagsRes, fieldsRes, p.timestamp.UnixNano())
}
