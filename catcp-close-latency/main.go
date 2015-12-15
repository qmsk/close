package main

import (
	"close/ping"
	"close/statsd"
	"fmt"
	"time"
	"log"
	"flag"
	"os"
)

func collectLatency(src string, p *ping.Pinger, c *statsd.Client) {
	go func() {
		for {
			latency := <-p.RTT
			latency_s := latency.Seconds()
			metric := fmt.Sprintf("%s.latency.%s", src, p.DstName)
			c.SendTiming(metric, latency_s)
		}
	}()

	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		p.Latency()
	}
}

func main() {
	// TODO Some random value
	var defaultSrc = "unknown"
	if hostname, ok := os.Hostname(); ok != nil {
		defaultSrc = hostname
	}

	var statsdURL = flag.String("statsd", "statsd.docker.catcp:8125", "URL to statsd daemon")
	var pingTarget = flag.String("ping", "8.8.8.8", "target host to send ICMP echos to")
	var pingSource = flag.String("source", defaultSrc, "source to use in the metric name to send ICMP echos from")

	flag.Parse()

	c, err := statsd.NewClient(*statsdURL)
	if err != nil {
		log.Panicf("Could not connect to statsd server\n")
	}
	defer c.Close()

	p, err := ping.NewPinger(*pingTarget)
	if err != nil {
		log.Panicf("Could not create a new Pinger\n")
	}
	defer p.Close()

	collectLatency(*pingSource, p, c)
}
