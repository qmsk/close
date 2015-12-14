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
			latency := <-p.RTT
			latency_s := latency.Seconds()
			fmt.Printf( "latency: %f\n", latency_s )
			c.SendTiming("latency-ya", latency_s)
		}
	}()

	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
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
