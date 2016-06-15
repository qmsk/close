package main

import (
    "github.com/qmsk/close/udp"
)

func init() {
    Options.Register("udp_send", &udp.SendConfig{})
}
