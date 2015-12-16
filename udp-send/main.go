package main

import (
    "flag"
    "log"
    "close/udp"
)

var (
    senderConfig    udp.SendConfig
    showStats       bool
)

func init() {
    flag.UintVar(&senderConfig.DestPort, "port", udp.PORT,
        "port")
    flag.StringVar(&senderConfig.SourceNet, "source-net", "",
        "addr/prefixlen")
    flag.UintVar(&senderConfig.SourcePort, "source-port", udp.SOURCE_PORT,
        "port")
    flag.UintVar(&senderConfig.SourcePortBits, "source-port-bits", udp.SOURCE_PORT_BITS,
        "fixed bits of port")

    flag.UintVar(&senderConfig.Rate, "rate", 0,
        "rate /s")
    flag.UintVar(&senderConfig.Size, "size", 0,
        "bytes")

    flag.BoolVar(&showStats, "show-stats", false,
        "display stats")
}

func stats(statsChan chan udp.SendStats) {
    for stats := range statsChan {
        log.Println(stats)
    }
}

func main() {
    flag.Parse()

    if destAddr := flag.Arg(0); destAddr == "" {
        log.Fatalf("Usage: [options] <dest-addr>\n")
    } else {
        senderConfig.DestAddr = destAddr
    }

    udpSend, err := udp.NewSend(senderConfig)
    if err != nil {
        log.Fatalf("udp.NewSend %v: %v\n", senderConfig, err)
    } else {
        log.Printf("udp.NewSend %v: %+v\n", senderConfig, udpSend)
    }

    // stats
    if showStats {
        go stats(udpSend.GiveStats())
    }

    // run
    log.Printf("Run...\n")

    if err := udpSend.Run(); err != nil {
        log.Fatalf("udp.Send.Run: %v\n", err)
    } else {
        log.Printf("udp.Send: done\n")
    }
}
