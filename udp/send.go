package udp

import (
    "close/config"
    "fmt"
    "log"
    "net"
    "os"
    "close/stats"
    "time"
    "close/worker"
)

const SOURCE_PORT uint = 0
const SOURCE_PORT_BITS uint = 0

type SendConfig struct {
    DestAddr        string  `long:"dest-addr" value-name:"HOST:PORT" description:"Fixed destination address" required:"yes"`
    SourceNet       string  `long:"source-net" value-name:"HOST/MASK" description:"Use raw IP socket with given randomized range of source addresses"`
    SourcePort      uint    `long:"source-port" value-name:"0-65535" description:"Use raw IP socket with given randomized source port"`
    SourcePortBits  uint    `long:"source-port-bits" value-name:"0-16" description:"Randomize low-order bits of source-port"`

    ID              string  `json:"id" long:"id"`       // 64-bit hex ID, or random
    Rate            uint    `json:"rate" long:"rate"`   // 0 - unrated
    Count           uint    `json:"count" long:"count"` // 0 - infinite
    Size            uint    `json:"size" long:"size"`   // target size of UDP payload
}

func (self SendConfig) Worker() (worker.Worker, error) {
    return NewSend(self)
}

type SendStats struct {
    ID              ID              // send id, to correlated with RecvStats
    Start           time.Time       // stats were reset
    Time            time.Time       // stats were updated

    Config          SendConfig
    Rate            RateStats
    Send            SockStats       // Send.Bytes includes IP+UDP+Payload
}

func (self SendStats) StatsID() stats.ID {
    return stats.ID{
        Type:       "udp_send",
        Instance:   self.ID.String(),
    }
}

func (self SendStats) StatsTime() time.Time {
    return self.Time
}

func (self SendStats) Duration() time.Duration {
    return self.Time.Sub(self.Start)
}

func (self SendStats) CalcRate() float64 {
    return float64(self.Rate.Count) / self.Duration().Seconds()
}

// Return rate-loop utilization between 0..1, with 1.0 being fully utilized (unable to keep up with rate)
func (self SendStats) RateUtil() float64 {
    return 1.0 - self.Rate.SleepDuration.Seconds() / self.Duration().Seconds()
}

// Return the actual rate vs configured rate as a proportional error, with 1.0 being the most accurate
func (self SendStats) RateError() float64 {
    return self.CalcRate() / float64(self.Config.Rate)
}

func (self SendStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // gauges
        "rate_config": self.Config.Rate,
        "rate": self.CalcRate(),
        "rate_error": self.RateError(),
        "rate_util": self.RateUtil(),

        // counters
        "rate_underruns": self.Rate.UnderrunCount,
        "send_packets": self.Send.Packets,
        "send_bytes": self.Send.Bytes,
        "send_errors": self.Send.Errors,
    }
}

func (self SendStats) String() string {
    // pps rate of sent packets; may be lower than Rate() in the case of Send.Errors > 0
    sendRate := float64(self.Send.Packets) / self.Duration().Seconds()

    // achieved througput with IP+UDP headers
    sendMbps := float64(self.Send.Bytes) / 1000 / 1000 * 8 / self.Duration().Seconds()

    return fmt.Sprintf("send %9d with %5d underruns @ %10.2f/s = %8.2fMb/s +%5d errors @ %6.2f%% rate %6.2f%% util",
        self.Send.Packets, self.Rate.UnderrunCount,
        sendRate, sendMbps, self.Send.Errors,
        self.RateError() * 100,
        self.RateUtil() * 100,
    )
}

type Send struct {
    config      SendConfig
    log         *log.Logger

    dstAddr     net.IPAddr
    dstIP       net.IP
    dstPort     uint16
    srcAddr     net.IP
    srcAddrBits uint
    srcPort     RandPort
    id          ID

    sockSend  SockSend

    configChan  chan config.Config

    statsChan       chan stats.Stats
    statsInterval   time.Duration

}

func NewSend(config SendConfig) (*Send, error) {
    send := &Send{
        log:    log.New(os.Stderr, "udp.Send: ", 0),
    }

    if err := send.apply(config); err != nil {
        return nil, err
    }

    return send, nil
}

func (self *Send) String() string {
    return fmt.Sprintf("%v -> %v", self.id, self.dstAddr)
}

func (self *Send) Config() config.Config {
    return &self.config
}

func (self *Send) apply(config SendConfig) error {
    if config.SourceNet == "" && config.SourcePortBits == 0 {
        if err := self.initUDP(config); err != nil {
            return err
        }
    } else {
        if err := self.initIP(config); err != nil {
            return err
        }
    }

    // initialize real source
    self.srcPort.init(0) // TODO: seed

    if srcAddr, err := self.sockSend.probeSource(); err != nil {
        return err
    } else {
        self.srcAddr = srcAddr.IP
        self.srcAddrBits = 0
        self.srcPort.SetPort(uint(srcAddr.Port))
    }

    // generate ID from source addr, unless given
    if config.ID == "" {
        if id, err := genID(self.srcAddr, self.srcPort.Port()); err != nil {
            return fmt.Errorf("genID %v %v: %v", self.srcAddr, self.srcPort.Port(), err)
        } else {
            config.ID = id.String()
            self.id = id
        }
    } else {
        if id, err := parseID(config.ID); err != nil {
            return fmt.Errorf("parseID %v: %v", config.ID, err)
        } else {
            self.id = id
        }
    }

    // randomize source
    if config.SourceNet == "" {

    } else if _, ipNet, err := net.ParseCIDR(config.SourceNet); err != nil {
        return fmt.Errorf("Parse SourceNet %v: %v", config.SourceNet, err)
    } else {
        maskSet, maskBits := ipNet.Mask.Size()

        self.srcAddr = ipNet.IP
        self.srcAddrBits = uint(maskBits - maskSet)
    }

    if config.SourcePort != 0 {
        self.srcPort.SetPort(config.SourcePort)
    }
    if config.SourcePortBits > 0 {
        self.srcPort.SetRandom(config.SourcePortBits)
    }

    // config
    self.config = config

    return nil
}

// init with SockUDP sender
func (self *Send) initUDP(config SendConfig) error {
    sockUDP := &SockUDP{}
    if err := sockUDP.initDial(config.DestAddr); err != nil {
        return err
    }

    self.dstIP = sockUDP.udpAddr.IP
    self.dstPort = uint16(sockUDP.udpAddr.Port)

    self.sockSend = sockUDP

    return nil
}

// init with SockIP sender
func (self *Send) initIP(config SendConfig) error {
    // setup dest
    sock := &SockSyscall{}
    if err := sock.init(config.DestAddr); err != nil {
        return err
    }

    self.dstIP = sock.udpAddr.IP
    self.dstPort = uint16(sock.udpAddr.Port)

    self.sockSend = sock

    return nil
}

func (self *Send) StatsWriter(statsWriter *stats.Writer) error {
    self.statsChan = statsWriter.IntervalStatsWriter()

    return nil
}

// pull runtime configuration from config source
func (self *Send) ConfigSub(configSub *config.Sub) error {
    // copy for updates
    updateConfig := self.config

    if configChan, err := configSub.Start(&updateConfig); err != nil {
        return err
    } else {
        self.configChan = configChan
    }

    return nil
}

// Generate a sequence of *Packet
//
// Returns once complete, or error
func (self *Send) Run() error {
    payload := Payload{
        ID:     self.id,
        Seq:    0,
    }

    // stats
    stats := SendStats{
        ID:     payload.ID,
        Start:  time.Now(),
        Config: self.config,
    }

    self.sockSend.useStats(&stats.Send)

    // Rate
    var rateClock RateClock

    rateClock.init()
    rateClock.useStats(&stats.Rate)

    rateTick := rateClock.Start(self.config.Rate, self.config.Count)

    for {
        stats.Time = time.Now()

        select {
        case _, running := <-rateTick:
            if !running {
                return nil
            }

            // send
            packet := Packet{
                SrcIP:      self.srcAddr,
                SrcPort:    self.srcPort.Port(),
                DstIP:      self.dstIP,
                DstPort:    self.dstPort,

                Payload:        payload,
                PayloadSize:    self.config.Size,
            }

            if err := self.sockSend.send(packet); err != nil {
                return err
            }

            payload.Seq++

        case self.statsChan <- stats:
            // reset
            stats.Start = stats.Time
            stats.Rate = RateStats{}
            stats.Send = SockStats{}

        case configConfig := <-self.configChan:
            config := configConfig.(*SendConfig)

            self.log.Printf("config: %v\n", config)

            if config.Rate != self.config.Rate || config.Count != self.config.Count {
                self.log.Printf("config rate=%d count=%d\n", config.Rate, config.Count)

                rateClock.Set(config.Rate, config.Count)

                self.config.Rate = config.Rate
                self.config.Count = config.Count
            }

            if config.Size != self.config.Size {
                self.log.Printf("config size=%d\n", config.Size)

                self.config.Size = config.Size
            }
        }
    }
}

func (self *Send) Stop() {
    self.log.Printf("stopping...\n")

    // XXX:
    panic("stop")
}
