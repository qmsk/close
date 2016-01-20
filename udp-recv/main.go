package main

import (
    "github.com/jessevdk/go-flags"
    "log"
    "os"
    "close/stats"
    "close/udp"
)

type Options struct {
    Stats       stats.WriterOptions     `group:"Stats Writer"`
    UDPRecv     *udp.RecvConfig         `group:"UDP Receiver"`
}

func main() {
    // udp Recv
    udpRecv, err := udp.NewRecv()
    if err != nil {
        log.Fatalf("udp.NewRecv: %v\n", err)
    } else {
        log.Printf("udp.NewRecv: %v\n", udpRecv)
    }

    options := Options{
        UDPRecv:    udpRecv.Config(),
    }

    parser := flags.NewParser(&options, flags.Default)

    // flags
    if args, err := parser.Parse(); err != nil {
        log.Fatalf("flags.Parser: Parse: %v\n", err)
        os.Exit(1)
    } else if len(args) > 0 {
        log.Printf("flags Parser.Parser: extra arguments: %v\n", args)
        parser.WriteHelp(os.Stderr)
        os.Exit(1)
    }

    // stats
    if options.Stats.Empty() {
        log.Printf("Skip stats")
    } else if statsWriter, err := stats.NewWriter(options.Stats); err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", options.Stats, err)
    } else if err := udpRecv.StatsWriter(statsWriter); err != nil {
        log.Fatalf("udp.Recv %v: StatsWriter %v: %v\n", udpRecv, statsWriter, err)
    } else {
        log.Printf("upd.Recv %v: StatsWriter %v\n", udpRecv, statsWriter)
    }

    // run
    log.Printf("Run...\n",)

    if err := udpRecv.Run(); err != nil {
        log.Fatalf("udp.Recv %v: Run: %v\n", udpRecv, err)
    } else {
        log.Printf("udp.Recv %v: Run: done\n", udpRecv)
    }
}
