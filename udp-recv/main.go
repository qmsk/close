package main

import (
    "flag"
    "log"
    "os"
    "close/stats"
    "close/udp"
)

var (
    statsConfig     stats.Config
    receiverConfig  udp.RecvConfig
)

func init() {
    statsConfig.Type = "udp_recv"

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

    flag.StringVar(&receiverConfig.ListenAddr, "listen-addr", "0.0.0.0:1337",
        "host:port")
}

func main() {
    flag.Parse()

    udpRecv, err := udp.NewRecv(receiverConfig)
    if err != nil {
        log.Fatalf("udp.NewRecv %v: %v\n", receiverConfig, err)
    } else {
        log.Printf("udp.NewRecv %v: %+v\n", receiverConfig, udpRecv)
    }

    // stats
    if statsConfig.Instance == "" {
        statsConfig.Instance = receiverConfig.ListenAddr
    }

    statsWriter, err := stats.NewWriter(statsConfig)
    if err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.NewWriter %v: %v\n", statsConfig, statsWriter)
    }

    statsWriter.WriteFrom(udpRecv)

    // run
    log.Printf("Run...\n",)

    if err := udpRecv.Run(); err != nil {
        log.Fatalf("udp.Recv.Run: %v\n", err)
    } else {
        log.Printf("udp.Recv.Run: done\n")
    }
}
