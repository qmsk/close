package main

import (
    "close/ping"
    "close/statsd"
    "close/influxdb"
    "fmt"
    "time"
    "log"
    "flag"
    "os"
    "strings"
)

type LatencyConfig struct {
    Source  string
    Target  string
    Statsd  string
    InfluxDB  string
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
    flag.StringVar(&config.InfluxDB, "influxdb", "http://influxdb.docker.catcp:8086",
        "URL to influxdb")
    flag.StringVar(&config.Target, "ping", "8.8.8.8",
        "target host to send ICMP echos to")
    flag.StringVar(&config.Source, "source", defaultSrc,
        "source to use in the metric name to send ICMP echos from")
}

func collectLatency(config LatencyConfig, p *ping.Pinger, s *statsd.Client, i *influxdb.Client) {
    statsC := p.GiveStats()
    go func() {
        for {
            latency := <-statsC
            latency_s := latency.Seconds()
            targetName := strings.Replace(config.Target, ".", "_", -1)
            metric := fmt.Sprintf("%s.latency.%s", config.Source, targetName)
            s.SendTiming(metric, latency_s)

            point := influxdb.CreatePoint("latency")
            point.AddTag("source", config.Source)
            point.AddTag("target", config.Target)
            point.AddField("value", fmt.Sprintf("%.3f", latency_s) )

            i.AddPoint(*point)
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

    statsdClient, err := statsd.NewClient(config.Statsd)
    if err != nil {
        log.Panicf("Could not connect to statsd server\n")
    }
    defer statsdClient.Close()

    influxClient, err := influxdb.NewClient(influxdb.Config{
        Server: config.InfluxDB,
    })
    if err != nil {
        log.Panicf("Could not connect to statsd server\n")
    }
    defer statsdClient.Close()

    p, err := ping.NewPinger(config.Target)
    if err != nil {
        log.Panicf("Could not create a new Pinger\n")
    }
    defer p.Close()

    collectLatency(config, p, statsdClient, influxClient)
}
