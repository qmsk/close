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
    Instance    string              `json:"-"`

    Target      string              `json:"target"`
    ID          int                 `json:"id"`         // 32-bit id, default from pid
    Interval    float64             `json:"interval"`   // seconds
}

type PingStats struct {
    Instance        string
    Time            time.Time       // ping request was sent out

    RTT             time.Duration
}

func (self PingStats) StatsInstance() string {
    return self.Instance
}

func (self PingStats) StatsTime() time.Time {
    return self.Time
}

func (self PingStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // timing
        "rtt": self.RTT.Seconds(),
    }
}

func (self PingStats) String() string {
    return fmt.Sprintf("rtt=%.2fms",
        self.RTT.Seconds() * 1000,
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
    dst         net.Addr
    conn        *icmp.PacketConn

    config   PingConfig
    configC  chan config.Config

    rttC            chan  stats.Stats

    receiverC   chan  pingResult

    log         *log.Logger
}

func NewPinger(config PingConfig) (*Pinger, error) {
    if config.ID == 0 {
        config.ID = os.Getpid() & 0xffff
    }

    p := &Pinger{
        config:     config,
        log:        log.New(os.Stderr, "ping: ", 0),
        receiverC:  make(chan pingResult),
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

    go p.receiver()

    return nil
}

func (p *Pinger) Stop() {
    p.log.Printf("stopping...\n")

    // XXX: assume this trips recevier()?
    p.conn.Close()
}

// interval is ignored
func (p *Pinger) GiveStats(interval time.Duration) chan stats.Stats {
    p.rttC = make(chan stats.Stats)

    return p.rttC
}

func (p *Pinger) ConfigFrom(configRedis *config.Redis) (*config.Sub, error) {
    // copy for updates
    updateConfig := p.config

    if configSub, err := configRedis.NewSub("icmp_ping", p.config.Instance); err != nil {
        return nil, err
    } else if configChan, err := configSub.Start(&updateConfig); err != nil {
        return nil, err
    } else {
        p.configC = configChan

        return configSub, nil
    }
}

// mainloop
func (p *Pinger) Run() {
    defer close(p.rttC)
    defer p.log.Printf("stopped\n")

    var seq int
    timerChan := time.Tick(time.Duration(p.config.Interval * float64(time.Second)))
    startTimes  := make(map[int]time.Time)

    for {
        select {
        case <-timerChan:
            seq++

            if err := p.send(seq); err != nil {
                p.log.Printf("send %d: %v\n", seq, err)
            } else {
                startTimes[seq] = time.Now()
            }

        case result, ok := <-p.receiverC:
            if !ok {
                return
            }
            if start, ok := startTimes[result.Seq]; ok {
                rtt := result.Stop.Sub(start)

                // TODO statsInterval
                if p.rttC != nil {

                    // Could have takeStats interface...
                    s := PingStats{
                        Instance:   p.config.Instance,
                        Time:       start,
                        RTT:        rtt,
                    }

                    p.rttC <- s
                }
                delete(startTimes, result.Seq)
            }

        case configConfig := <-p.configC:
            config := configConfig.(*PingConfig)

            p.log.Printf("config: %v\n", config)
//      case <-expiryTicker.C:
        }
    }
}

// Now this is called only from manager, so it's okay it's not thread safe
func (p *Pinger) send(seq int) error {
    wm := icmp.Message {
        Type: ipv4.ICMPTypeEcho,
        Code: 0,
        Body: &icmp.Echo {
            ID:     p.config.ID,
            Seq:    seq,
            Data:   []byte("HELLO 1"),
        },
    }

    wb, err := wm.Marshal(nil)
    if err != nil {
        return fmt.Errorf("icmp Message.Marshal: %v", err)
    }

    n, err := p.conn.WriteTo(wb, p.dst)
    if n != len(wb) || err != nil {
        return fmt.Errorf("icmp PacketConn.WriteTo: %v", err)
    }

    return nil
}

func (p *Pinger) receiver() {
    defer close(p.receiverC)

    for {
        rb := make([]byte, 1500)
        n, _, err := p.conn.ReadFrom(rb)
        if err != nil {
            p.log.Printf("icmp PacketConn.ReadFrom: %v\n", err)

            // quit if the connection is closed
            break
        }

        stop := time.Now()

        // IANA ICMP v4 protocol is 1
        rm, err := icmp.ParseMessage(1, rb[:n])
        if err != nil {
            p.log.Printf("icmp.ParseMessage: %v\n", err)
            continue
        }

        if rm.Type == ipv4.ICMPTypeEchoReply {
            echo := rm.Body.(*icmp.Echo)
            p.receiverC <- pingResult{echo.Seq, stop}
        }
    }
}
