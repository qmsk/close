package main

import (
    "flag"
    "log"
    "time"
    "close/udp"
)

var (
    receiverConfig  udp.RecvConfig
    showStats       bool
)

func init() {
    flag.StringVar(&receiverConfig.ListenAddr, "listen-addr", "0.0.0.0:1337",
        "host:port")

    flag.BoolVar(&showStats, "show-stats", false,
        "display stats")
}

func logStats(statsChan chan udp.RecvStats) {
    statsTime := time.Now()

    for stats := range statsChan {
        logTime := time.Now()

        if logTime.Sub(statsTime).Seconds() > 1.0 {
            statsTime = logTime

            log.Println(stats)
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
    if showStats {
        go logStats(udpRecv.GiveStats())
    }

    // run
    log.Printf("Run...\n",)

    if err := udpRecv.Run(); err != nil {
        log.Fatalf("udp.Recv.Run: %v\n", err)
    } else {
        log.Printf("udp.Recv.Run: done\n")
    }
}
