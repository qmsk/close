package main

import (
    "close/icmp"
    "close/udp"
    "close/worker"
)

var options worker.Options

func init() {
    options.Register("icmp_ping", &icmp.PingConfig{})
    options.Register("udp_send", &udp.SendConfig{})
}

func main() {
    options.Parse()

    worker.Main(options)
}
