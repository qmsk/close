package main

import (
    "flag"
    "log"
    "close/udp"
)

var (
    receiverConfig  udp.ReceiverConfig
)

func init() {
    flag.StringVar(&receiverConfig.ListenAddr, "listen-addr", "0.0.0.0:1337",
        "host:port")
}

func stats(statsChan chan udp.ReceiverStats) {
    for stats := range statsChan {
        log.Println(stats)
    }
}

func main() {
    flag.Parse()

    udpReceiver, err := udp.NewReceiver(receiverConfig)
    if err != nil {
        log.Fatalf("udp.NewReceiver %v: %v\n", receiverConfig, err)
    } else {
        log.Printf("udp.NewReceiver %v: %+v\n", receiverConfig, udpReceiver)
    }

    // stats
    go stats(udpReceiver.GiveStats())

    log.Printf("Run...\n",)
    if err := udpReceiver.Run(); err != nil {
        log.Fatalf("udp.Receiver.Run: %v\n", err)
    } else {
        log.Printf("udp.Receiver.Run: done\n")
    }

}
