package udp

import (
    "close/config"
    "fmt"
    "log"
    "net"
    "os"
    "close/stats"
    "time"
)

const SOURCE_PORT uint = 0
const SOURCE_PORT_BITS uint = 0

type SendConfig struct {
    Instance        string  `json:"-"`

    DestAddr        string // host:port (or host)
    SourceNet       string // host/mask
    SourcePort      uint
    SourcePortBits  uint

    ID              uint64  `json:"id"`     // 64-bit ID, or random
    Rate            uint    `json:"rate"`   // 0 - unrated
    Count           uint    `json:"count"`  // 0 - infinite
    Size            uint    `json:"size"`   // target size of UDP payload
}

type SendStats struct {
    ID              uint64          // send id, to correlated with RecvStats
    Time            time.Time       // stats were reset
    Duration        time.Duration   // stats were collected

    Config          SendConfig
    Rate            RateStats

    // Send.Bytes includes IP+UDP+Payload
    Send            SockStats
}

func (self SendStats) StatsTime() time.Time {
    return self.Time
}

func (self SendStats) CalcRate() float64 {
    return float64(self.Rate.Count) / self.Duration.Seconds()
}

// Return rate-loop utilization between 0..1, with 1.0 being fully utilized (unable to keep up with rate)
func (self SendStats) RateUtil() float64 {
    return 1.0 - self.Rate.SleepDuration.Seconds() / self.Duration.Seconds()
}

// Return the actual rate vs configured rate as a proportional error, with 1.0 being the most accurate
func (self SendStats) RateError() float64 {
    return self.CalcRate() / float64(self.Config.Rate)
}

func (self SendStats) StatsInstance() string {
    return fmt.Sprintf("%d", self.ID)
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
    sendRate := float64(self.Send.Packets) / self.Duration.Seconds()

    // achieved througput with IP+UDP headers
    sendMbps := float64(self.Send.Bytes) / 1000 / 1000 * 8 / self.Duration.Seconds()

    return fmt.Sprintf("%5.2f: send %9d with %5d underruns @ %10.2f/s = %8.2fMb/s +%5d errors @ %6.2f%% rate %6.2f%% util",
        self.Duration.Seconds(),
        self.Send.Packets, self.Rate.UnderrunCount,
        sendRate, sendMbps, self.Send.Errors,
        self.RateError() * 100,
        self.RateUtil() * 100,
    )
}

type Send struct {
    dstAddr     net.IPAddr
    dstIP       net.IP
    dstPort     uint16
    srcAddr     net.IP
    srcAddrBits uint
    srcPort     RandPort

    sockSend  SockSend

    config      SendConfig
    configChan  chan config.Config

    statsChan       chan stats.Stats
    statsInterval   time.Duration

    log         *log.Logger
}

func NewSend(config SendConfig) (*Send, error) {
    sender := &Send{
        log:    log.New(os.Stderr, "udp.Send: ", 0),
    }

    if err := sender.init(config); err != nil {
        return nil, err
    } else {
        return sender, nil
    }
}

func (self *Send) init(config SendConfig) error {
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
    if config.ID != 0 {

    } else if id, err := genID(self.srcAddr, self.srcPort.Port()); err != nil {
        return fmt.Errorf("genID %v %v: %v", self.srcAddr, self.srcPort.Port(), err)
    } else {
        config.ID = id
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

func (self *Send) ID() string {
    return fmt.Sprintf("%d", self.config.ID)
}

func (self *Send) GiveStats(interval time.Duration) chan stats.Stats {
    self.statsChan = make(chan stats.Stats)
    self.statsInterval = interval

    return self.statsChan
}

// pull runtime configuration from config source
func (self *Send) ConfigFrom(configRedis *config.Redis) (*config.Sub, error) {
    // copy for updates
    updateConfig := self.config

    if configSub, err := configRedis.Sub(config.SubOptions{"udp_send", self.config.Instance}); err != nil {
        return nil, err
    } else if configChan, err := configSub.Start(&updateConfig); err != nil {
        return nil, err
    } else {
        self.configChan = configChan

        return configSub, nil
    }
}

// Generate a sequence of *Packet
//
// Returns once complete, or error
func (self *Send) Run() error {
    var rateClock RateClock

    rateTick := rateClock.Start(self.config.Rate, self.config.Count)

    payload := Payload{
        ID:     uint64(self.config.ID),
        Seq:    0,
    }

    // stats
    statsStart := time.Now()
    statsTick := time.Tick(self.statsInterval)

    for {
        select {
        case sendTime := <-rateTick:
            if sendTime.IsZero() {
                // channel closed
                break
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

        case statsTime := <-statsTick:
            self.statsChan <- SendStats{
                ID:         payload.ID,
                Time:       statsTime,
                Duration:   statsTime.Sub(statsStart),

                Config:     self.config,

                Rate:       rateClock.takeStats(),
                Send:       self.sockSend.takeStats(),
            }

            statsStart = statsTime

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

    return nil
}
