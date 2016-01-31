// +build !workertest

package workers

import (
    "close/icmp"
    "close/udp"
    "close/worker"
)

var Options worker.Options

func init() {
    Options.Register("icmp_ping", &icmp.PingConfig{})
    Options.Register("udp_send", &udp.SendConfig{})
}
