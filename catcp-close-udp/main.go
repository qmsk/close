package main

import (
    "flag"
    "log"
    "catcp/close/udp"
)

var (
    senderConfig    udp.SenderConfig
    rate            uint
)

func init() {
    flag.UintVar(&senderConfig.DestPort, "dest-port", udp.PORT,
        "port")
    flag.StringVar(&senderConfig.SourceNet, "source-net", "",
        "addr/prefixlen")
    flag.UintVar(&senderConfig.SourcePort, "source-port", udp.SOURCE_PORT,
        "port")
    flag.UintVar(&senderConfig.SourcePortBits, "source-port-bits", udp.PORT_BITS,
        "fixed bits of port")

    flag.UintVar(&rate, "rate", 1,
        "rate /s")
}

func main() {
    flag.Parse()

    if destAddr := flag.Arg(0); destAddr == "" {
        log.Fatalf("Usage: [options] <dest-addr>\n")
    } else {
        senderConfig.DestAddr = destAddr
    }

    udpSender, err := udp.NewSender(senderConfig)
    if err != nil {
        log.Fatalf("udp.NewSender %v: %v\n", senderConfig, err)
    } else {
        log.Printf("udp.NewSender %v: %+v\n", senderConfig, udpSender)
    }

    log.Printf("Run @%v/s", rate)

    if err := udpSender.Run(rate); err != nil {
        log.Fatalf("udp.Sender.Run: %v\n", err)
    } else {
        log.Printf("udp.Sender: done\n")
    }
}
