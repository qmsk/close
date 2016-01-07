package ping

import (
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
    "close/stats"
    "close/config"
    "os"
    "log"
    "fmt"
    "time"
    "net"
)

type PingConfig struct {
    Target  string
}

type PingStats struct {
    Target          string
    Time            time.Time       // ping response came in
    RTT             time.Duration
}

func (self PingStats) StatsInstance() string {
    return self.Target
}

func (self PingStats) StatsTime() time.Time {
    return self.Time
}

func (self PingStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // timing
        "rtt": self.RTT,
    }
}

func (self PingStats) String() string {
    return fmt.Sprintf("%.2f s round-trip time",
        self.RTT,
    )
}

func getAddr(dst string) (net.Addr, error) {
    if ips, err := net.LookupIP(dst); err != nil {
        return nil, err
    } else {
        return &net.UDPAddr{IP: ips[0]}, err
    }
}

type pingResult struct {
    Seq   int
    Stop  time.Time
}

type Pinger struct {
    target      string
    dst         net.Addr
    conn        *icmp.PacketConn
    seq         int

    config   PingConfig
    configC  chan config.Config

    statsInterval   time.Duration
    rttC            chan  stats.Stats

    senderC     chan  bool
    receiverC   chan  pingResult

    startTimes  map[int]time.Time

    log         *log.Logger
}

func NewPinger(config PingConfig) (*Pinger, error) {
    p := &Pinger {
        target: config.Target,
        seq: 1,
        log:    log.New(os.Stderr, "ping: ", 0),
    }

    if err := p.init(config); err != nil {
        return nil, err
    } else {
        return p, nil
    }
}

func (p *Pinger) init(config PingConfig) error {
    conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
    if err != nil {
        p.log.Printf("Could not start listening: %s\n", err)
        return err
    }
    p.conn = conn

    udpAddr, err := getAddr(config.Target)
    if err != nil {
        p.log.Printf("Could not resolve remote address: %s\n", err)
        return err
    }

    p.dst = udpAddr
    p.senderC = make(chan bool)
    p.receiverC = make(chan pingResult)
    p.startTimes = make(map[int]time.Time)

    p.config = config

    go p.receiver()
    go p.manager()

    return nil
}

func (p *Pinger) Close() {
    p.conn.Close()
    close(p.rttC)
    close(p.senderC)
    close(p.receiverC)
}

func (p *Pinger) Latency() {
    p.senderC <- true
}

func (p *Pinger) GiveStats(interval time.Duration) chan stats.Stats {
    p.rttC = make(chan stats.Stats)
    p.statsInterval = interval

    return p.rttC
}

func (p *Pinger) ConfigFrom(configRedis *config.Redis) (*config.Sub, error) {
    // copy for updates
    updateConfig := p.config

    if configSub, err := configRedis.Sub(config.SubOptions{"ping", p.target}); err != nil {
        return nil, err
    } else if configChan, err := configSub.Start(&updateConfig); err != nil {
        return nil, err
    } else {
        p.configC = configChan

        return configSub, nil
    }
}

func (p *Pinger) manager() {
    for {
        select {
        case <-p.senderC:
            p.ping()

        case result, ok := <-p.receiverC:
            if !ok {
                break
            }
            if start, ok := p.startTimes[result.Seq]; ok {
                // TODO statsInterval
                if p.rttC != nil {

                    // Could have takeStats interface...
                    s := PingStats{
                        Target: p.target,
                        Time: result.Stop,
                        RTT: result.Stop.Sub(start),
                    }

                    p.rttC <- s
                }
                delete(p.startTimes, result.Seq)
            }

        case configConfig := <-p.configC:
            config := configConfig.(*PingConfig)

            p.log.Printf("config: %v\n", config)
//      case <-expiryTicker.C:
        }
    }
}

// Now this is called only from manager, so it's okay it's not thread safe
func (p *Pinger) ping() {
    p.seq += 1
    seq := p.seq

    wm := icmp.Message {
        Type: ipv4.ICMPTypeEcho,
        Code: 0,
        Body: &icmp.Echo {
            ID: os.Getpid() & 0xffff, Seq: seq,
            Data: []byte("HELLO 1"),
        },
    }

    wb, err := wm.Marshal(nil)
    if err != nil {
        p.log.Printf("Could not marshal ICMP message: %s\n", err)
    }

    p.startTimes[seq] = time.Now()

    n, err := p.conn.WriteTo(wb, p.dst)
    if n != len(wb) || err != nil {
        p.log.Printf("Could not send the packet: %s\n", err)
        delete(p.startTimes, seq)
    }
}

func (p *Pinger) receiver() {
    // If the receiver channel was closed then trying to send on it will panic
    defer func() {
        if r := recover(); r != nil {
            p.log.Println("Recovered in Pinger.receiver()", r)
        }
    }()
    for {
        rb := make([]byte, 1500)
        n, _, err := p.conn.ReadFrom(rb)
        // TODO If the connection is closed quit the loop
        if err != nil {
            p.log.Printf("Could not read the ICMP reply: %s\n", err)
            continue
        }

        stop := time.Now()

        // IANA ICMP v4 protocol is 1
        rm, err := icmp.ParseMessage(1, rb[:n])
        if err != nil {
            p.log.Printf("Could not parse the ICMP reply: %s\n", err)
            continue
        }

        if rm.Type == ipv4.ICMPTypeEchoReply {
            echo := rm.Body.(*icmp.Echo)
            p.receiverC <- pingResult { echo.Seq, stop, }
        }
    }
}
