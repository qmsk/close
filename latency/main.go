package main

import (
    "close/ping"
    "close/statsd"
    "fmt"
    "time"
    "log"
    "flag"
    "os"
)

type LatencyConfig struct {
    Source  string
    Target  string
    Statsd  string
}

var (
    config    LatencyConfig
)

func init() {
    // TODO Some random value
    var defaultSrc = "unknown"
    if hostname, err := os.Hostname(); err == nil {
        defaultSrc = hostname
    }

    flag.StringVar(&config.Statsd, "statsd", "statsd.docker.catcp:8125",
        "URL to statsd daemon")
    flag.StringVar(&config.Target, "ping", "8.8.8.8",
        "target host to send ICMP echos to")
    flag.StringVar(&config.Source, "source", defaultSrc,
        "source to use in the metric name to send ICMP echos from")
}

func collectLatency(config LatencyConfig, p *ping.Pinger, c *statsd.Client) {
    statsC := p.GiveStats()
    go func() {
        for {
            latency := <-statsC
            latency_s := latency.Seconds()
            metric := fmt.Sprintf("%s.latency.%s", config.Source, config.Target)
            c.SendTiming(metric, latency_s)
        }
    }()

    ticker := time.NewTicker(time.Second)
    for {
        <-ticker.C
        p.Latency()
    }
}

func main() {
    flag.Parse()

    c, err := statsd.NewClient(config.Statsd)
    if err != nil {
        log.Panicf("Could not connect to statsd server\n")
    }
    defer c.Close()

    p, err := ping.NewPinger(config.Target)
    if err != nil {
        log.Panicf("Could not create a new Pinger\n")
    }
    defer p.Close()

    collectLatency(config, p, c)
}
