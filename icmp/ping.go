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
    Target      string              `json:"target" long:"target"`
    ID          int                 `json:"id" long:"id"`
    Interval    time.Duration       `json:"interval" long:"interval" value-name:"<count>(ns|us|ms|s|m|h)" default:"1s"`
}

type PingStats struct {
    ID              int
    Time            time.Time       // ping request was sent out

    RTT             time.Duration
}

func (self PingStats) StatsID() stats.ID {
    return stats.ID{
        Type:       "icmp_ping",
        Instance:   fmt.Sprintf("%d", self.ID),
    }
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

type pingResult struct {
    ID    uint16
    Seq   uint16
    Time  time.Time
}

type Pinger struct {
    config      PingConfig

    log         *log.Logger

    targetAddr  *net.UDPAddr
    icmpConn    *icmp.PacketConn

    configC     chan config.Config
    statsC      chan  stats.Stats
    receiverC   chan  pingResult

}

func NewPinger() (*Pinger, error) {
    p := &Pinger{
        config:     PingConfig{
            ID:         os.Getpid(),
        },
        log:        log.New(os.Stderr, "ping: ", 0),
    }

    return p, nil
}

func (p *Pinger) String() string {
    return fmt.Sprintf("Ping %v", p.config.Target)
}

func (p *Pinger) Config() config.Config {
    return &p.config
}

func (p *Pinger) StatsWriter(statsWriter *stats.Writer) error {
    p.statsC = statsWriter.StatsWriter()

    return nil
}

func (p *Pinger) ConfigSub(configSub *config.Sub) error {
    // copy for updates
    pingConfig := p.config

    if configChan, err := configSub.Start(&pingConfig); err != nil {
        return err
    } else {
        p.configC = configChan

        return nil
    }
}

func (p *Pinger) resolveTarget(target string) (*net.UDPAddr, error) {
    if target == "" {
        return nil, fmt.Errorf("No target given")
    }

    if ipAddr, err := net.ResolveIPAddr("ip", target); err != nil {
        return nil, err
    } else {
        // unprivileged icmp mode uses SOCK_DGRAM
        udpAddr := &net.UDPAddr{IP: ipAddr.IP, Zone: ipAddr.Zone}

        return udpAddr, nil
    }
}

func (p *Pinger) icmpListen(targetAddr *net.UDPAddr) (*icmp.PacketConn, error) {
    if ip4 := targetAddr.IP.To4(); ip4 != nil {
        return icmp.ListenPacket("udp4", "")
    } else {
        return nil, fmt.Errorf("Unsupported address: %v", targetAddr)
    }
}

// Apply configuration to state
// TODO: teardown old state?
func (p *Pinger) apply(config PingConfig) error {
    if targetAddr, err := p.resolveTarget(config.Target); err != nil {
        return fmt.Errorf("resolveTarget: %v", err)
    } else {
        p.targetAddr = targetAddr
    }

    if icmpConn, err := p.icmpListen(p.targetAddr); err != nil {
        return fmt.Errorf("icmpListen: %v", err)
    } else {
        p.icmpConn = icmpConn
    }

    p.receiverC = make(chan pingResult)

    // good
    p.config = config

    go p.receiver(p.receiverC, p.icmpConn)

    return nil
}

// mainloop
func (p *Pinger) Run() error {
    if p.statsC != nil {
        defer close(p.statsC)
    }

    // start
    if err := p.apply(p.config); err != nil {
        return err
    }

    defer p.log.Printf("stopped\n")

    // state
    var id = uint16(p.config.ID)
    var seq uint16
    timerChan := time.Tick(p.config.Interval)
    startTimes  := make(map[uint16]time.Time)

    for {
        select {
        case <-timerChan:
            seq++

            if err := p.send(id, seq); err != nil {
                return err
            } else {
                startTimes[seq] = time.Now()
            }

        case result, ok := <-p.receiverC:
            if !ok {
                return nil
            }
            if startTime, ok := startTimes[result.Seq]; ok {
                rtt := result.Time.Sub(startTime)

                if p.statsC != nil {
                    p.statsC <- PingStats{
                        ID:         p.config.ID,
                        Time:       startTime,
                        RTT:        rtt,
                    }
                }

                delete(startTimes, result.Seq)
            }

        case configConfig := <-p.configC:
            config := configConfig.(*PingConfig)

            p.log.Printf("config: %v\n", config)

            // TODO: apply()

//      case <-expiryTicker.C:
        }
    }
}

func (p *Pinger) Stop() {
    p.log.Printf("stopping...\n")

    // causes recevier() to close(receiverC)
    p.icmpConn.Close()
}

// Now this is called only from manager, so it's okay it's not thread safe
func (p *Pinger) send(id uint16, seq uint16) error {
    wm := icmp.Message {
        Type: ipv4.ICMPTypeEcho,
        Code: 0,
        Body: &icmp.Echo {
            ID:     int(id),
            Seq:    int(seq),
            Data:   []byte("HELLO 1"),
        },
    }

    wb, err := wm.Marshal(nil)
    if err != nil {
        return fmt.Errorf("icmp Message.Marshal: %v", err)
    }

    n, err := p.icmpConn.WriteTo(wb, p.targetAddr)
    if n != len(wb) || err != nil {
        return fmt.Errorf("icmp.PacketConn %v: WriteTo %v: %v", p.icmpConn, p.targetAddr, err)
    }

    return nil
}

func (p *Pinger) receiver(receiverC chan pingResult, icmpConn *icmp.PacketConn) {
    defer close(receiverC)

    // IANA ICMP v4 protocol is 1
    // TODO: get protocol from icmpConn, for IPv6?
    icmpProto := 1

    for {
        buf := make([]byte, 1500)
        if readSize, _, err := icmpConn.ReadFrom(buf); err != nil {
            p.log.Printf("icmp.PacketConn %v: ReadFrom: %v\n", icmpConn, err)

            // quit if the connection is closed
            return
        } else {
            buf = buf[:readSize]
        }

        recvTime := time.Now()

        if icmpMessage, err := icmp.ParseMessage(icmpProto, buf); err != nil {
            p.log.Printf("icmp.ParseMessage: %v\n", err)
            continue
        } else if icmpEcho, ok := icmpMessage.Body.(*icmp.Echo); ok {
            receiverC <- pingResult{
                ID:     uint16(icmpEcho.ID),
                Seq:    uint16(icmpEcho.Seq),
                Time:   recvTime,
            }
        }
    }
}
