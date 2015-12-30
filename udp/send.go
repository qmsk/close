package udp

import (
    "fmt"
    "net"
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

    ID              string  // 64-bit ID, or random
    Rate            uint    // 0 - unrated
    Size            uint    // target size of UDP payload
}

type SendStats struct {
    ID              uint64          // send id, to correlated with RecvStats
    Time            time.Time       // stats were reset
    Duration        time.Duration   // stats were collected

    ConfigRate      uint            // configured target rate
    RateSleep       time.Duration   // total time slept
    RateUnderruns   uint            // count of timing underruns (no sleep)
    RateCount       uint            // count of timing ticks

    // Send.Bytes includes IP+UDP+Payload
    Send            SockStats
}

func (self SendStats) StatsTime() time.Time {
    return self.Time
}

func (self SendStats) Rate() float64 {
    return float64(self.RateCount) / self.Duration.Seconds()
}

// Return rate-loop utilization between 0..1, with 1.0 being fully utilized (unable to keep up with rate)
func (self SendStats) RateUtil() float64 {
    return 1.0 - self.RateSleep.Seconds() / self.Duration.Seconds()
}

// Return the actual rate vs configured rate as a proportional error, with 1.0 being the most accurate
func (self SendStats) RateError() float64 {
    return self.Rate() / float64(self.ConfigRate)
}

func (self SendStats) StatsInstance() string {
    return fmt.Sprintf("%016x", self.ID)
}

func (self SendStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // gauges
        "rate_config": self.ConfigRate,
        "rate": self.Rate(),
        "rate_error": self.RateError(),
        "rate_util": self.RateUtil(),

        // counters
        "rate_underruns": self.RateUnderruns,
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
        self.Send.Packets, self.RateUnderruns,
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

    id          uint64
    rate        uint
    size        uint
    count       uint

    stats           SendStats
    statsChan       chan stats.Stats
    statsInterval   time.Duration
}

func NewSend(config SendConfig) (*Send, error) {
    sender := &Send{

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

    // id
    if config.ID == "" {
        // generate from dst
        self.id = uint64(rand.Int63())
    } else if id, err := strconv.ParseUint(config.ID, 16, 64); err != nil {
        return fmt.Errorf("Parse ID %v: %v", config.ID, err)
    } else {
        self.id = id
    }

    // config
    self.rate = config.Rate
    self.size = config.Size

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

func (self *Send) GiveStats(interval time.Duration) chan stats.Stats {
    self.statsChan = make(chan stats.Stats)
    self.statsInterval = interval

    return self.statsChan
}

// Generate a sequence of *Packet
func (self *Send) run(rate uint, size uint, count uint) error {
    startTime := time.Now()

    // reset stats
    self.stats = SendStats{
        ID:         self.id,
        Time:       startTime,

        ConfigRate: rate,
    }
    payload := Payload{
        ID:     self.id,
        Seq:    0,
    }

    for {
        // rate-limiting?
        if rate != 0 {
            // scheduled time for next packet
            packetClock := time.Duration(payload.Seq) * time.Second / time.Duration(rate)

            rateClock := time.Since(startTime)
            rateSkew := packetClock - rateClock

            if rateSkew > 0 {
                // slow down
                time.Sleep(rateSkew)

                self.stats.RateSleep += rateSkew
            } else {
                // catch up
                self.stats.RateUnderruns++
            }

            self.stats.RateCount++
        }

        // send
        packet := Packet{
            SrcIP:      self.srcAddr,
            SrcPort:    self.srcPort.Port(),
            DstIP:      self.dstIP,
            DstPort:    self.dstPort,

            Payload:        payload,
            PayloadSize:    size,
        }

        if err := self.sockSend.send(packet); err != nil {
            return err
        }

        payload.Seq++

        // stats
        self.stats.Duration = time.Since(self.stats.Time)

        if self.statsChan != nil && self.stats.Duration >= self.statsInterval {
            self.stats.Send = self.sockSend.takeStats()
            self.statsChan <- self.stats

            // reset
            self.stats = SendStats{
                ID:         self.id,
                Time:       time.Now(),

                ConfigRate: rate,
            }
        }

        // end?
        if count > 0 && payload.Seq > uint64(count) {
            break
        }
    }

    return nil
}

func (self *Send) Run() error {
    // TODO: reconfigure
    return self.run(self.rate, self.size, self.count)
}
