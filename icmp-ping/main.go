package main

import (
    "close/icmp"
    "close/stats"
    "close/config"
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
    statsConfig.Type = "icmp_ping"

    flag.StringVar(&statsConfig.InfluxDB.Addr, "influxdb-addr", "http://influxdb:8086",
        "influxdb http://... address")
    flag.StringVar(&statsConfig.InfluxDBDatabase, "influxdb-database", stats.INFLUXDB_DATABASE,
        "influxdb database name")

    flag.StringVar(&statsConfig.Hostname, "stats-hostname", os.Getenv("STATS_HOSTNAME"),
        "hostname to uniquely identify this source")
    flag.BoolVar(&statsConfig.Print, "stats-print", false,
        "display stats on stdout")

    flag.StringVar(&configOptions.Redis.Addr, "config-redis-addr", "",
        "host:port")
    flag.Int64Var(&configOptions.Redis.DB, "config-redis-db", 0,
        "Database to select")
    flag.StringVar(&configOptions.Prefix, "config-prefix", "close",
        "Redis key prefix")

    flag.StringVar(&pingConfig.Instance, "instance", os.Getenv("CLOSE_ID"),
        "type instance")
    flag.Float64Var(&pingConfig.Interval, "interval", 1.0,
        "ping interval")
    flag.StringVar(&pingConfig.Target, "target", "",
        "target host to send ICMP echos to")
}

func main() {
    flag.Parse()

    p, err := ping.NewPinger(pingConfig)
    if err != nil {
        log.Fatalf("ping.NewPinger %v: %v\n", pingConfig, err)
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

    p.Run()
}
