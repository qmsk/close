package workers

import (
    "github.com/qmsk/close/icmp"
)

func init() {
    Options.Register("icmp_ping", &icmp.PingConfig{})
}
