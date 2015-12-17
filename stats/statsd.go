package stats

import (
    "log"
    "net"
    "fmt"
)

const STATSD_SERVER = "statsd:8125"

type StatsdConfig struct {
    Server      string
    Debug       bool
}

type Client struct {
    conn    net.Conn
    debug   bool
}

func NewClient(config StatsdConfig) (*Client, error) {
    if config.Server == "" {
        config.Server = STATSD_SERVER
    }

    c := &Client {
        debug:  config.Debug,
    }

    if conn, err := net.Dial("udp", config.Server); err != nil {
        return nil, err
    } else {
        c.conn = conn
    }

    return c, nil
}

func (c *Client) String() string {
    return c.conn.RemoteAddr().String()
}

func (c *Client) send(name, value, suffix string) error {
    msg := fmt.Sprintf("%v:%v|%v\n", name, value, suffix)

    if c.debug {
        log.Printf("statsd %v: send %v\n", c, msg)
    }

    if n, err := c.conn.Write([]byte(msg)); err != nil {
        return err
    } else if n != len(msg) {
        return fmt.Errorf("Short write")
    } else {
        return nil
    }
}

func (c *Client) SendCounter(name string, value uint) error {
    return c.send(name, fmt.Sprintf("%d", value), "c")
}
func (c *Client) SendTiming(name string, value float64) error {
    return c.send(name, fmt.Sprintf("%f", value), "ms")
}
func (c *Client) SendGauge(name string, value uint) error {
    return c.send(name, fmt.Sprintf("%d", value), "g")
}
func (c *Client) SendGaugef(name string, value float64) error {
    return c.send(name, fmt.Sprintf("%f", value), "g")
}
func (c *Client) SendSet(name string, value string) error {
    return c.send(name, fmt.Sprintf("%s", value), "s")
}

func (c *Client) Close() {
    c.conn.Close()
}
