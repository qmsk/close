package influxdb

import (
    "net/http"
    "strings"
)

const WRITE_URL = "/write?db="

type Client struct {
    httpClient http.Client
    server     string
    database   string
    points     chan point
}

func NewClient(config Config) (*Client, error) {
    c := &Client{
    }
    if err := c.init(config); err != nil {
        return nil, err
    }
    return c, nil
}

func (c *Client) Close() {
}

func (c *Client) init(config Config) error {
    d := config.WithDefaults()
    c.server = d.Server
    c.database = d.Database
    c.points = make(chan point)
    return nil
}

func (c *Client) AddPoint(p point) {
    c.points <- p
}

func (c *Client) sender() {
    for {
        p, ok := <-c.points
        if !ok {
            break
        }
        c.httpClient.Post(c.server + WRITE_URL + c.database, "application/text", 
            strings.NewReader(p.Serialize()))
    }
}
