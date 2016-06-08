package workers

import (
    "github.com/qmsk/close/dns"
)

func init() {
    Options.Register("dns", &dns.Config{})
}
