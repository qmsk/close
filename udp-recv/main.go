package main

import (
    "flag"
    "log"
    "close/udp"
)

var (
    receiverConfig  udp.RecvConfig
)

func init() {
    flag.StringVar(&receiverConfig.ListenAddr, "listen-addr", "0.0.0.0:1337",
        "host:port")
}

func stats(statsChan chan udp.RecvStats) {
    for stats := range statsChan {
        log.Println(stats)
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
    go stats(udpRecv.GiveStats())

    log.Printf("Run...\n",)
    if err := udpRecv.Run(); err != nil {
        log.Fatalf("udp.Recv.Run: %v\n", err)
    } else {
        log.Printf("udp.Recv.Run: done\n")
    }

}
