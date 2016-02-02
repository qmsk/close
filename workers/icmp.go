package workers

import (
    "close/icmp"
)

func init() {
    Options.Register("icmp_ping", &icmp.PingConfig{})
}
