package stats

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"
)

const INTERVAL = 1.0

type Config struct {
    Statsd      StatsdConfig

    // The hostname this instance is running on to uniquely identify the source of measurements
    // If multiple instances of the same type are running on a single host, they must have a different hostname
    Hostname    string

    // Type of measurements being sent
    Type        string

    // The target being measured, intended to be aggregated from multiple instances of this type running on different hosts
    Instance    string

    // Collection interval
    Interval    float64 // seconds
}

// Wrap a statsd client to uniquely identify the measurements
type Stats struct {
    config      Config
    client      *Client
    Interval    time.Duration
}

func New(config Config) (*Stats, error) {
    if config.Hostname == "" {
        if hostname, err := os.Hostname(); err != nil {
            return nil, err
        } else {
            config.Hostname = hostname
        }
    }
    if strings.Contains(config.Hostname, ".") {
        log.Printf("statsd-hostname: stripping domain\n")
        config.Hostname = strings.Split(config.Hostname, ".")[0]
    }

    if config.Type == "" {
        panic("Invalid stats-type")
    }

    if config.Instance == "" {
        return nil, fmt.Errorf("Invalid stats-instance")
    } else if strings.ContainsAny(config.Instance, ".:") {
        log.Printf("statsd-instance: escaping\n")
        config.Instance = strings.Map(func(c rune) rune {
            switch(c) {
            case '.', ':':
                return '_'
            default:
                return c
            }
        }, config.Instance)
    }

    stats := &Stats{
        config:     config,
        Interval:   time.Duration(config.Interval * float64(time.Second)),
    }

    if client, err := NewClient(config.Statsd); err != nil {
        return nil, err
    } else {
        stats.client = client
    }

    return stats, nil
}

func (self *Stats) path(field string) string {
    return strings.Join([]string{self.config.Hostname, self.config.Type, self.config.Instance, field}, ".")
}

func (self *Stats) SendCounter(name string, value uint) error {
    return self.client.SendCounter(self.path(name), value)
}
func (self *Stats) SendTiming(name string, value float64) error {
    return self.client.SendTiming(self.path(name), value)
}
func (self *Stats) SendGauge(name string, value uint) error {
    return self.client.SendGauge(self.path(name), value)
}
func (self *Stats) SendGaugef(name string, value float64) error {
    return self.client.SendGaugef(self.path(name), value)
}
func (self *Stats) SendSet(name string, value string) error {
    return self.client.SendSet(self.path(name), value)
}
