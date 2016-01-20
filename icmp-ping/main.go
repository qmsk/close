package main

import (
    "close/icmp"
    "close/worker"
    "log"
)


func main() {
    p, err := ping.NewPinger()
    if err != nil {
        log.Fatalf("ping.NewPinger: %v\n", err)
    }

    worker.Main(p)
}
