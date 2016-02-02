package workers

import (
    "close/udp"
)

func init() {
    Options.Register("udp_send", &udp.SendConfig{})
}
