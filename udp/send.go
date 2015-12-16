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
    StartTime       time.Time

    Rate            uint            // configured rate
    RateClock       time.Duration
    RateSleep       time.Duration
    RateUnderrun    uint
    RateCount       uint

    // Send.Bytes includes IP+UDP+Payload
    Send            SockStats
}

func (self SendStats) String() string {
    clock := self.RateClock
    sendRate := float64(self.Send.Packets) / clock.Seconds()
    sendMbps := float64(self.Send.Bytes) / 1000 / 1000 * 8 / clock.Seconds()
    util := 1.0 - self.RateSleep.Seconds() / clock.Seconds()

    return fmt.Sprintf("%8.2f: send %8d @ %8.2f/s = %8.2fMb/s @ %5.2f%% rate %5.2f%% util",
        clock.Seconds(),
        self.Send.Packets, sendRate,
        sendMbps,
        sendRate / float64(self.Rate) * 100,
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

    stats       SendStats
    statsChan   chan SendStats
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
    // setup dest
    sockIP := &SockIP{}
    if err := sockIP.init(fmt.Sprintf("%v:%v", config.DestAddr, config.DestPort)); err != nil {
        return err
    }

    self.dstIP = sockIP.udpAddr.IP
    self.dstPort = uint16(sockIP.udpAddr.Port)
    self.sockSend = sockIP

    // source
    if config.SourceNet == "" {
        if srcAddr, err := sockIP.probeSource(); err != nil {
            return err
        } else {
            self.srcAddr = srcAddr.IP
            self.srcAddrBits = 0
            self.srcPort = makeRandPort(uint(srcAddr.Port))
        }
    } else if _, ipNet, err := net.ParseCIDR(config.SourceNet); err != nil {
        return fmt.Errorf("Parse SourceNet %v: %v", config.SourceNet, err)
    } else {
        maskSet, maskBits := ipNet.Mask.Size()

        self.srcAddr = ipNet.IP
        self.srcAddrBits = uint(maskBits - maskSet)
    }

    if config.SourcePort != 0 {
        self.srcPort = makeRandPort(config.SourcePort)
    }
    if config.SourcePortBits > 0 {
        self.srcPort.SetRandom(config.SourcePortBits, 0) // XXX: seed
    }

    // config
    self.rate = config.Rate
    self.size = config.Size

    return nil
}

func (self *Send) GiveStats() chan SendStats {
    self.statsChan = make(chan SendStats)

    return self.statsChan
}

// Generate a sequence of *Packet
func (self *Send) run(rate uint, size uint, count uint) error {
    startTime := time.Now()

    // reset stats
    self.stats = SendStats{
        StartTime:  startTime,
        Rate:       rate,
    }
    payload := Payload{
        Start:  startTime.Unix(),
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

            self.stats.RateClock = time.Since(startTime)
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
        if self.statsChan != nil {
            self.stats.Send = self.sockSend.getStats()
            self.statsChan <- self.stats
        }

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
