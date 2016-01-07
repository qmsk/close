package main

import (
    "close/ping"
    "close/stats"
    "close/config"
    "time"
    "log"
    "flag"
    "os"
)

var (
    statsConfig     stats.Config
    pingConfig      ping.PingConfig
    configOptions   config.Options
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

    flag.StringVar(&configOptions.Redis.Addr, "config-redis-addr", "",
        "host:port")
    flag.Int64Var(&configOptions.Redis.DB, "config-redis-db", 0,
        "Database to select")
    flag.StringVar(&configOptions.Prefix, "config-prefix", "close",
        "Redis key prefix")

    flag.StringVar(&pingConfig.Target, "ping", "8.8.8.8",
        "target host to send ICMP echos to")
}

func collectLatency(p *ping.Pinger) {
    ticker := time.NewTicker(time.Second)
    for {
        <-ticker.C
        p.Latency()
    }
}

func main() {
    flag.Parse()

    p, err := ping.NewPinger(pingConfig)
    if err != nil {
        log.Panicf("ping.NewPinger %v: %v\n", pingConfig, err)
    } else {
        log.Printf("ping.NewPinger %v: %v\n", pingConfig, p)
    }
    defer p.Close()

    // config
    if configOptions.Redis.Addr == "" {

    } else if configRedis, err := config.NewRedis(configOptions); err != nil {
        log.Fatalf("config.NewRedis %v: %v\n", configOptions, err)
    } else if configSub, err := p.ConfigFrom(configRedis); err != nil {
        log.Fatalf("ping.ConfigFrom %v: %v\n", configRedis, err)
    } else {
        log.Printf("ping.ConfigFrom: %v\n", configSub)
    }

    // stats
    statsWriter, err := stats.NewWriter(statsConfig)
    if err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.NewWriter %v: %v\n", statsConfig, statsWriter)

        statsWriter.WriteFrom(p)
    }


    collectLatency(p)
}
