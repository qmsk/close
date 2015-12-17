package udp

import (
    "fmt"
    "net"
    "time"
)

const SOURCE_PORT uint = 0
const SOURCE_PORT_BITS uint = 0

type SendConfig struct {
    DestAddr        string // host
    DestPort        uint
    SourceNet       string // host/mask
    SourcePort      uint
    SourcePortBits  uint

    Rate            uint    // 0 - unrated
    Size            uint    // target size of UDP payload
}

type SendStats struct {
    Time            time.Time       // stats were reset
    Duration        time.Duration   // stats were collected

    Rate            uint            // configured target rate
    RateSleep       time.Duration   // total time slept
    RateUnderrun    uint            // count of timing underruns (no sleep)
    RateCount       uint            // count of timing ticks

    // Send.Bytes includes IP+UDP+Payload
    Send            SockStats
}

func (self SendStats) String() string {
    sendRate := float64(self.Send.Packets) / self.Duration.Seconds()
    sendMbps := float64(self.Send.Bytes) / 1000 / 1000 * 8 / self.Duration.Seconds()

    rate := float64(self.RateCount) / self.Duration.Seconds() / float64(self.Rate)
    util := 1.0 - self.RateSleep.Seconds() / self.Duration.Seconds()

    return fmt.Sprintf("%5.2f: send %9d @ %10.2f/s = %8.2fMb/s +%5d errors @ %6.2f%% rate %6.2f%% util",
        self.Duration.Seconds(),
        self.Send.Packets, sendRate,
        sendMbps,
        self.Send.Errors,
        rate * 100,
        util * 100,
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

    rate        uint
    size        uint
    count       uint

    stats           SendStats
    statsChan       chan SendStats
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
        self.initUDP(config)
    } else {
        self.initIP(config)
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
    self.rate = config.Rate
    self.size = config.Size

    return nil
}

// init with SockUDP sender
func (self *Send) initUDP(config SendConfig) error {
    sockUDP := &SockUDP{}
    if err := sockUDP.initDial(fmt.Sprintf("%v:%v", config.DestAddr, config.DestPort)); err != nil {
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
    if err := sock.init(fmt.Sprintf("%v:%v", config.DestAddr, config.DestPort)); err != nil {
        return err
    }

    self.dstIP = sock.udpAddr.IP
    self.dstPort = uint16(sock.udpAddr.Port)

    self.sockSend = sock

    return nil
}

func (self *Send) GiveStats(interval time.Duration) chan SendStats {
    self.statsChan = make(chan SendStats)
    self.statsInterval = interval

    return self.statsChan
}

// Generate a sequence of *Packet
func (self *Send) run(rate uint, size uint, count uint) error {
    startTime := time.Now()

    // reset stats
    self.stats = SendStats{
        Time:   startTime,
        Rate:   rate,
    }
    payload := Payload{
        Start:  uint64(startTime.Unix()),
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
                self.stats.RateUnderrun++
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
                Time:       time.Now(),
                Rate:       rate,
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
