package workers

import (
    "close/dns"
)

func init() {
    Options.Register("dns", &dns.Config{})
}
