package docker

import (
    "testing"
)

var testParseStatus = []struct{
    status  string
    state   string
    exit    int
    since   string
}{
    {"Up About a minute",           "running",  0,      "About a minute"},
    {"Up 9 minutes",                "running",  0,      "9 minutes"},
    {"Exited (0) 2 days ago",       "exited",   0,      "2 days"},
    {"Exited (137) 4 weeks ago",    "exited",   137,    "4 weeks"},
}

func TestParseStatus(t *testing.T) {
    for _, test := range testParseStatus {
        state, exit, since := parseStatus(test.status)

        if state != test.state {
            t.Errorf("%v: state %#v != %#v", test.status, state, test.state)
        }
        if exit != test.exit {
            t.Errorf("%v: exit %#v != %#v", test.status, exit, test.exit)
        }
        if since != test.since {
            t.Errorf("%v: since %#v != %#v", test.status, since, test.since)
        }
    }
}
