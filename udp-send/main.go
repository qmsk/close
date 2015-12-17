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
    sendConfig      udp.SendConfig
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

    flag.StringVar(&sendConfig.SourceNet, "source-net", "",
        "addr/prefixlen")
    flag.UintVar(&sendConfig.SourcePort, "source-port", udp.SOURCE_PORT,
        "port")
    flag.UintVar(&sendConfig.SourcePortBits, "source-port-bits", udp.SOURCE_PORT_BITS,
        "fixed bits of port")

    flag.UintVar(&sendConfig.Rate, "rate", 0,
        "rate /s")
    flag.UintVar(&sendConfig.Size, "size", 0,
        "bytes")

    flag.BoolVar(&showStats, "show-stats", false,
        "display stats")
}

func logStats(s *stats.Stats, statsChan chan udp.SendStats) {
    for stats := range statsChan {
        if showStats {
            log.Println(stats)
        }

        s.SendGauge("rate-config", stats.ConfigRate)
        s.SendGaugef("rate", stats.Rate())
        s.SendGaugef("rate-error", stats.RateError())
        s.SendGaugef("rate-util", stats.RateUtil())
        s.SendCounter("rate-underruns", stats.RateUnderruns)

        s.SendCounter("send-packets", stats.Send.Packets)
        s.SendCounter("send-bytes", stats.Send.Bytes)
        s.SendCounter("send-errors", stats.Send.Errors)
    }
}

func main() {
    flag.Parse()

    if destAddr := flag.Arg(0); destAddr == "" {
        log.Fatalf("Usage: [options] <dest-host>:<dest-port>>\n")
    } else {
        sendConfig.DestAddr = destAddr
    }

    if statsConfig.Instance == "" {
        statsConfig.Instance = sendConfig.DestAddr
    }

    s, err := stats.New(statsConfig)
    if err != nil {
        log.Fatalf("stats.New %v: %v\n", statsConfig, err)
    } else {
        log.Printf("stats.New %v: %v\n", statsConfig, s)
    }

    udpSend, err := udp.NewSend(sendConfig)
    if err != nil {
        log.Fatalf("udp.NewSend %v: %v\n", sendConfig, err)
    } else {
        log.Printf("udp.NewSend %v: %+v\n", sendConfig, udpSend)
    }

    // stats
    if showStats {
        go logStats(s, udpSend.GiveStats(s.Interval))
    }

    // run
    log.Printf("Run...\n")

    if err := udpSend.Run(); err != nil {
        log.Fatalf("udp.Send.Run: %v\n", err)
    } else {
        log.Printf("udp.Send: done\n")
    }
}
