package icmp

import (
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
    SendTime        time.Time       // ping request was sent out
    RecvTime        time.Time
}

func (self PingStats) StatsID() stats.ID {
    // use default Instance:
    return stats.ID{
        Type:       "icmp_ping",
    }
}

func (self PingStats) StatsTime() time.Time {
    return self.SendTime
}

func (self PingStats) RTT() time.Duration {
    return self.RecvTime.Sub(self.SendTime)
}

func (self PingStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // timing
        "rtt": self.RTT().Seconds(),
    }
}

func (self PingStats) String() string {
    return fmt.Sprintf("rtt=%.2fms",
        self.RTT().Seconds() * 1000,
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

    configC     chan  config.ConfigPush
    statsC      chan  stats.Stats
    receiverC   chan  Ping
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

    if conn, err := NewConn(config.Proto, config.Target); err != nil {
        return err
    } else {
        p.conn = conn
    }

    p.receiverC = make(chan Ping)

    // good
    p.config = config

    go p.receiver(p.receiverC, p.conn)

    return nil
}

func (p *Pinger) configPush(configPush config.ConfigPush) (config.Config, error) {
    config := p.config // copy

    if err := configPush.Unmarshal(&config); err != nil {
        return nil, err
    }

    p.log.Printf("config: %#v\n", config)

    // TODO: apply()

    return nil, fmt.Errorf("Not implemented")
}

// mainloop
func (p *Pinger) Run() error {
    if p.statsC != nil {
        defer close(p.statsC)
    }
    defer p.log.Printf("stopped\n")

    timerChan := time.Tick(p.config.Interval)

    // state
    var packets = make(map[uint16]Ping) // inflight
    var packet = Ping{
        ID:     uint16(p.config.ID),
        Seq:    0,
        Data:   []byte("HELLO 1"),
    }

    for {
        select {
        case <-timerChan:
            packet.Seq++

            // Send, updating .Time
            if err := p.conn.Send(&packet); err != nil {
                p.log.Printf("send %v: %v\n", packet, err)
            } else {
                packets[packet.Seq] = packet
            }

        case recvPacket, ok := <-p.receiverC:
            if !ok {
                p.log.Printf("recv closed\n")
                return nil
            }

            sendPacket, ok := packets[recvPacket.Seq]
            if !ok {
                p.log.Printf("recv bad seq: %#v\n", recvPacket)
                continue
            }

            delete(packets, recvPacket.Seq)

            // stat
            if p.statsC != nil {
                p.statsC <- PingStats{
                    ID:         int(sendPacket.ID),
                    SendTime:   sendPacket.Time,
                    RecvTime:   recvPacket.Time,
                }
            }

        case configPush, open := <-p.configC:
            if !open {
                p.log.Printf("config closed\n")
                return nil
            } else {
                configPush.ApplyFunc(p.configPush)
            }

//      case <-expiryTicker.C:
        }
    }
}

func (p *Pinger) Stop() {
    p.log.Printf("stopping...\n")

    // causes recevier() to close(receiverC)
    if err := p.conn.Close(); err != nil {
        p.log.Fatalf("icmpConn.Close: %v\n", err)
    }
}

func (p *Pinger) receiver(receiverC chan Ping, conn *Conn) {
    defer close(receiverC)

    for {
        if ping, err := conn.Recv(); err != nil {
            p.log.Printf("icmp.PacketConn %v: ReadFrom: %v\n", conn, err)

            // TODO: quit if the connection is closed, report other errors?
            return
        } else {
            receiverC <- ping
        }
    }
}
