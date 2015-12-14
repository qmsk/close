package statsd

import (
	"net"
	"log"
	"bytes"
	"strconv"
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

func (c *Client) SendFoo() error {
	return c.send([]byte("foo_latency:300|ms\n"))
}

func (c *Client) SendTiming(name string, timing int) error {
	var buffer bytes.Buffer
	buffer.WriteString(name)
	buffer.WriteString(":")
	buffer.Write([]byte(strconv.Itoa(timing)))
	buffer.WriteString("|ms")
	return c.send(buffer.Bytes())
}

func (c *Client) send(buf []byte) error {
	n, err := c.conn.Write(buf)
	if n != len(buf) {
		log.Print("Did not send the whole message")
	}
	return err
}
