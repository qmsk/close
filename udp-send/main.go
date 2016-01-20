package main

import (
    "log"
    "close/udp"
    "close/worker"
)

func main() {
    udpSend, err := udp.NewSend()
    if err != nil {
        log.Fatalf("udp.NewSend: %v\n", err)
    }

    worker.Main(udpSend)
}
