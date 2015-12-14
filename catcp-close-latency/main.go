package main

import (
	"catcp/close/ping"
	"catcp/close/statsd"
	"fmt"
	"time"
	"log"
)

func collectLatency(p *ping.Pinger, c *statsd.Client) {
	go func() {
		for {
			latency_ms := int(<-p.RTT / time.Millisecond)
			fmt.Printf( "latency: %d\n", latency_ms )
			c.SendTiming("latency-ya", latency_ms)
		}
	}()

	ticker := time.NewTicker(time.Second)
	for {
		<- ticker.C
		p.Latency()
	}
}

func main() {
	c, err := statsd.NewClient("statsd.docker.catcp:8125")
	if err != nil {
		log.Panicf("Could not connect to statsd server\n")
	}
	defer c.Close()

	p, err := ping.NewPinger("ya.ru")
	if err != nil {
		log.Panicf("Could not create a new Pinger\n")
	}
	defer p.Close()

	collectLatency(p, c)
}
