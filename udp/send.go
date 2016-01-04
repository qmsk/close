package udp

import (
    "close/config"
    "fmt"
    "log"
    "net"
    "os"
    "math/rand"
    "close/stats"
    "strconv"
    "time"
)

const SOURCE_PORT uint = 0
const SOURCE_PORT_BITS uint = 0

type SendConfig struct {
    DestAddr        string // host:port (or host)
    SourceNet       string // host/mask
    SourcePort      uint
    SourcePortBits  uint

    ID              uint    // 64-bit ID, or random
    Rate            uint    // 0 - unrated
    Count           uint    // 0 - infinite
    Size            uint    // target size of UDP payload
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
    return fmt.Sprintf("%016x", self.ID)
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
    configChan  chan SendConfig

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

    // source
    self.srcPort.init(0) // TODO: seed

    if config.SourceNet == "" {
        if srcAddr, err := self.sockSend.probeSource(); err != nil {
            return err
        } else {
            self.srcAddr = srcAddr.IP
            self.srcAddrBits = 0
            self.srcPort.SetPort(uint(srcAddr.Port))
        }
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
    if config.ID == 0 {
        // generate from dst
        config.ID = uint(rand.Int63())
    }

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
    return fmt.Sprintf("%016x", self.config.ID)
}

func (self *Send) GiveStats(interval time.Duration) chan stats.Stats {
    self.statsChan = make(chan stats.Stats)
    self.statsInterval = interval

    return self.statsChan
}

func (self *SendConfig) update(configMap map[string]string) error {
    for key, value := range configMap {
        switch key {
        case "id":
            continue
        case "rate":
            if rate, err := strconv.ParseUint(value, 10, 32); err != nil {
                return err
            } else {
                self.Rate = uint(rate)
            }
        case "count":
            if count, err := strconv.ParseUint(value, 10, 32); err != nil {
                return err
            } else {
                self.Count = uint(count)
            }
        case "size":
            if size, err := strconv.ParseUint(value, 10, 32); err != nil {
                return err
            } else {
                self.Size = uint(size)
            }
        default:
            return fmt.Errorf("Unknown field: %v", key)
        }
    }

    return nil
}

func (self *Send) configFrom(readChan chan map[string]string) {
    config := self.config

    for configMap := range readChan {
        if err := config.update(configMap); err != nil {
            self.log.Printf("config update %v: %v", configMap, err)
        } else {
            self.log.Printf("config update %v", configMap)
            self.configChan <- config
        }
    }
}

func (self *Send) ConfigFrom(configRedis *config.Redis) (*config.Sub, error) {
    configMap := config.Config{
        "rate":     fmt.Sprintf("%v", self.config.Rate),
        "count":    fmt.Sprintf("%v", self.config.Count),
        "size":     fmt.Sprintf("%v", self.config.Size),
    }

    if configSub, err := configRedis.Sub(config.SubOptions{"udp_send", self.ID()}, configMap); err != nil {
        return nil, err
    } else if readChan, err := configSub.Read(); err != nil {
        return nil, err
    } else {
        self.configChan = make(chan SendConfig)

        go self.configFrom(readChan)

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

        case config := <-self.configChan:
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
