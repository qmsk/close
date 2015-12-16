package statsd

import (
    "net"
    "log"
    "fmt"
)

type Client struct {
    conn net.Conn
}

func NewClient(dst string) (*Client, error) {
    c := &Client {
    }

    udpAddr, err := net.ResolveUDPAddr("udp", dst)
    if err != nil {
        return nil, err
    }

    c.conn, err = net.DialUDP(udpAddr.Network(), nil, udpAddr)
    if err != nil {
        return nil, err
    }

    return c, nil
}

func (c *Client) Close() {
    c.conn.Close()
}

func (c *Client) SendTiming(name string, timing float64) error {
    pkt := []byte( fmt.Sprintf("%s:%.3f|ms", name, timing) )
    return c.send(pkt)
}

func (c *Client) send(buf []byte) error {
    n, err := c.conn.Write(buf)
    if n != len(buf) {
        log.Print("Did not send the whole message")
    }
    return err
}
