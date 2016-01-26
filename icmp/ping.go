package icmp

import (
    "golang.org/x/net/icmp"
    "close/stats"
    "close/config"
    "os"
    "log"
    "fmt"
    "time"
    "close/worker"
)

type PingConfig struct {
    Target      string              `json:"target" long:"target"`
    Proto       string              `json:"proto" long:"protocol" value-name:"ipv4|ipv6" default:"ipv4"`
    ID          int                 `json:"id" long:"id"`
    Interval    time.Duration       `json:"interval" long:"interval" value-name:"<count>(ns|us|ms|s|m|h)" default:"1s"`
}

func (self PingConfig) Worker() (worker.Worker, error) {
    return NewPinger(self)
}

type PingStats struct {
    ID              int
    Time            time.Time       // ping request was sent out

    RTT             time.Duration
}

func (self PingStats) StatsID() stats.ID {
    // use default Instance:
    return stats.ID{
        Type:       "icmp_ping",
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

    conn        *Conn

    configC     chan  config.Config
    statsC      chan  stats.Stats
    receiverC   chan  pingResult
}

func NewPinger(config PingConfig) (*Pinger, error) {
    p := &Pinger{
        log:        log.New(os.Stderr, "ping: ", 0),
    }

    // start
    if err := p.apply(config); err != nil {
        return nil, err
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

// Apply configuration to state
// TODO: teardown old state?
func (p *Pinger) apply(config PingConfig) error {
    if config.ID == 0 {
        // XXX: this is going to be 1 when running within docker..
        config.ID = os.Getpid()
    }

    if conn, err := NewConn(config); err != nil {
        return err
    } else {
        p.conn = conn
    }

    p.receiverC = make(chan pingResult)

    // good
    p.config = config

    go p.receiver(p.receiverC, p.conn)

    return nil
}

// mainloop
func (p *Pinger) Run() error {
    if p.statsC != nil {
        defer close(p.statsC)
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
    p.conn.IcmpConn.Close()
}

func (p *Pinger) send(id uint16, seq uint16) error {
    wm := p.conn.NewMessage(id, seq)

    if err := p.conn.Write(wm); err != nil {
        return fmt.Errorf("icmp.PacketConn %v: WriteTo %v: %v", p.conn.IcmpConn, p.conn.TargetAddr, err)
    }

    return nil
}

func (p *Pinger) receiver(receiverC chan pingResult, conn *Conn) {
    defer close(receiverC)

    icmpProto := conn.Proto.ianaProto
    icmpConn := conn.IcmpConn

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
