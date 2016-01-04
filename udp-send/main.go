package main

import (
    "close/config"
    "flag"
    "log"
    "os"
    "close/stats"
    "close/udp"
)

var (
    statsConfig     stats.Config
    sendConfig      udp.SendConfig
    configOptions   config.Options
)

func init() {
    statsConfig.Type = "udp_send"

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

    flag.StringVar(&sendConfig.SourceNet, "source-net", "",
        "addr/prefixlen")
    flag.UintVar(&sendConfig.SourcePort, "source-port", udp.SOURCE_PORT,
        "port")
    flag.UintVar(&sendConfig.SourcePortBits, "source-port-bits", udp.SOURCE_PORT_BITS,
        "fixed bits of port")

    flag.StringVar(&configOptions.Redis.Addr, "config-redis-addr", "",
        "host:port")
    flag.Int64Var(&configOptions.Redis.DB, "config-redis-db", 0,
        "Database to select")
    flag.StringVar(&configOptions.Prefix, "config-prefix", "close",
        "Redis key prefix")

    flag.UintVar(&sendConfig.ID, "id", 0,
        "ID (hexadecimal uint64)")
    flag.UintVar(&sendConfig.Rate, "rate", 0,
        "rate /s")
    flag.UintVar(&sendConfig.Size, "size", 0,
        "bytes")
}

func main() {
    flag.Parse()

    if destAddr := flag.Arg(0); destAddr == "" {
        log.Fatalf("Usage: [options] <dest-host>:<dest-port>>\n")
    } else {
        sendConfig.DestAddr = destAddr
    }

    udpSend, err := udp.NewSend(sendConfig)
    if err != nil {
        log.Fatalf("udp.NewSend %v: %v\n", sendConfig, err)
    } else {
        log.Printf("udp.NewSend %v: %+v\n", sendConfig, udpSend)
    }

    // config?
    if configOptions.Redis.Addr == "" {

    } else if configRedis, err := config.NewRedis(configOptions); err != nil {
        log.Fatalf("config.NewRedis %v: %v\n", configOptions, err)
    } else if configSub, err := udpSend.ConfigFrom(configRedis); err != nil {
        log.Fatalf("udp.Send.ConfigFrom %v: %v\n", configRedis, err)
    } else {
        log.Printf("udp.Send.ConfigFrom: %v\n", configSub)
    }

    // stats
    statsWriter, err := stats.NewWriter(statsConfig)
    if err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.NewWriter %v: %v\n", statsConfig, statsWriter)

        statsWriter.WriteFrom(udpSend)
    }

    // run
    log.Printf("Run...\n")

    if err := udpSend.Run(); err != nil {
        log.Fatalf("udp.Send.Run: %v\n", err)
    } else {
        log.Printf("udp.Send: done\n")
    }
}
