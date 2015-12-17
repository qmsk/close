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
    showStats       bool
)

func init() {
    statsConfig.Type = "udp"

    flag.StringVar(&statsConfig.Statsd.Server, "statsd-server", os.Getenv("STATSD_SERVER"),
        "host:port address of statsd UDP server")
    flag.BoolVar(&statsConfig.Statsd.Debug, "statsd-debug", false,
        "trace statsd")
    flag.StringVar(&statsConfig.Hostname, "stats-hostname", os.Getenv("STATS_HOSTNAME"),
        "hostname to uniquely identify this source")
    flag.StringVar(&statsConfig.Instance, "stats-instance", os.Getenv("STATS_INSTANCE"),
        "instance to uniquely identify the target")
    flag.Float64Var(&statsConfig.Interval, "stats-interval", stats.INTERVAL,
        "stats interval")


    flag.StringVar(&receiverConfig.ListenAddr, "listen-addr", "0.0.0.0:1337",
        "host:port")

    flag.BoolVar(&showStats, "show-stats", false,
        "display stats")
}

func logStats(s *stats.Stats, statsChan chan udp.RecvStats) {
    for stats := range statsChan {
        if showStats {
            log.Println(stats)
        }

        s.SendCounter("recv-packets", stats.Recv.Packets)
        s.SendCounter("recv-bytes", stats.Recv.Bytes)
        s.SendCounter("recv-errors", stats.Recv.Errors)

        s.SendCounter("packet-errors", stats.PacketErrors)
        s.SendCounter("packets", stats.PacketCount)
        s.SendCounter("packet-skips", stats.PacketSkips)
        s.SendCounter("packet-dups", stats.PacketDups)

        if stats.Valid() {
            s.SendGaugef("packet-win", stats.PacketWin())
            s.SendGaugef("packet-loss", stats.PacketLoss())
        }
    }
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

    s, err := stats.New(statsConfig)
    if err != nil {
        log.Fatalf("stats.New %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.New %v: %v\n", statsConfig, s)
    }

    // stats
    if showStats {
        go logStats(s, udpRecv.GiveStats(s.Interval))
    }

    // run
    log.Printf("Run...\n",)

    if err := udpRecv.Run(); err != nil {
        log.Fatalf("udp.Recv.Run: %v\n", err)
    } else {
        log.Printf("udp.Recv.Run: done\n")
    }
}
