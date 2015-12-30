package main

import (
    "close/ping"
    "close/stats"
    "time"
    "log"
    "flag"
    "os"
)

var (
    statsConfig   stats.Config
    pingConfig    ping.PingConfig
)

func init() {
    statsConfig.Type = "icmp_latency"

    flag.StringVar(&statsConfig.InfluxDB.Addr, "influxdb-addr", "http://influxdb:8086",
        "influxdb http://... address")
    flag.StringVar(&statsConfig.InfluxDBDatabase, "influxdb-database", stats.INFLUXDB_DATABASE,
        "influxdb database name")
    flag.StringVar(&statsConfig.Hostname, "stats-hostname", os.Getenv("STATS_HOSTNAME"),
        "hostname to uniquely identify this source")
    flag.StringVar(&statsConfig.Instance, "stats-instance", os.Getenv("STATS_INSTANCE"),
        "instance to uniquely identify the target")
    flag.Float64Var(&statsConfig.Interval, "stats-interval", stats.INTERVAL,
        "stats interval")
    flag.BoolVar(&statsConfig.Print, "stats-print", false,
        "display stats on stdout")

    flag.StringVar(&config.Target, "ping", "8.8.8.8",
        "target host to send ICMP echos to")
}

func collectLatency(p *ping.Pinger, w *stats.Writer) {
    w.WriteFrom(p)

    ticker := time.NewTicker(time.Second)
    for {
        <-ticker.C
        p.Latency()
    }
}

func main() {
    flag.Parse()

    // stats
    statsWriter, err := stats.NewWriter(statsConfig)
    if err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.NewWriter %v: %v\n", statsConfig, statsWriter)
    }

    p, err := ping.NewPinger(pingConfig)
    if err != nil {
        log.Panicf("ping.NewPinger %v: %v\n", pingConfig, err)
    } else {
        log.Printf("ping.NewPinger %v: %v\n", pingConfig, p)
    }
    defer p.Close()

    collectLatency(p, statsWriter)
}
