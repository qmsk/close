package udp

import (
    "fmt"
    "github.com/google/gopacket"
    "golang.org/x/net/ipv4"
    "github.com/google/gopacket/layers"
    "net"
    "time"
)

const SOURCE_PORT uint = 0
const SOURCE_PORT_BITS uint = 0
const IP_TTL uint8 = 64

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

    SendErrors      uint
    SendPackets     uint
    SendBytes       uint    // includes IP+UDP+Payload
}

func (self SendStats) String() string {
    clock := self.RateClock
    sendRate := float64(self.SendPackets) / clock.Seconds()
    util := 1.0 - self.RateSleep.Seconds() / clock.Seconds()

    return fmt.Sprintf("%8.2f: send %8d @ %8.2f/s = %8.2fMb/s @ %5.2f%% rate %5.2f%% util",
        clock.Seconds(),
        self.SendPackets, sendRate,
        float64(self.SendBytes) / 1000 / 1000 * 8,
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

    ipConn      *net.IPConn
    rawConn     *ipv4.RawConn

    rate        uint
    size        uint

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

// probe the source address the kernel would select for the given destination
func probeSource(udpAddr *net.UDPAddr) (*net.UDPAddr, error) {
    if udpConn, err := net.DialUDP("udp", nil, udpAddr); err != nil {
        return nil, err
    } else {
        switch addr := udpConn.LocalAddr().(type) {
        case *net.UDPAddr:
            return addr, nil
        default:
            return nil, fmt.Errorf("Unknown address: %#v", addr)
        }
    }
}

func (self *Send) init(config SendConfig) error {
    // resolve
    if ipAddr, err := net.ResolveIPAddr("ip", config.DestAddr); err != nil {
        return fmt.Errorf("Resolve DestAddr%v: %v", config.DestAddr, err)
    } else {
        self.dstAddr = *ipAddr
        self.dstIP = ipAddr.IP
    }

    self.dstPort = uint16(config.DestPort)

    if config.SourceNet == "" {
        if srcAddr, err := probeSource(&net.UDPAddr{IP: self.dstIP, Port: int(self.dstPort)}); err != nil {
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

    // setup
    if ip4 := self.dstIP.To4(); ip4 != nil {
        if ipConn, err := net.ListenIP("ip4:udp", nil); err != nil {
            return fmt.Errorf("ListenIP: %v", err)
        } else {
            self.ipConn = ipConn
        }

        if rawConn, err := ipv4.NewRawConn(self.ipConn); err != nil {
            return fmt.Errorf("NewRawConn: %v", err)
        } else {
            self.rawConn = rawConn
        }

    } else if ip6 := self.dstIP.To16(); ip6 != nil {
        return fmt.Errorf("TODO: IPv6")
    } else {
        return fmt.Errorf("Invalid IP family")
    }

    // config
    self.rate = config.Rate
    self.size = config.Size

    return nil
}

type SerializableNetworkLayer interface {
    gopacket.NetworkLayer
    gopacket.SerializableLayer
}

// serialize and send from gopacket layers
func (self *Send) sendLayers(ip SerializableNetworkLayer, udp *layers.UDP, payload *gopacket.Payload) error {
    // serialize
    serializeBuffer := gopacket.NewSerializeBuffer()
    serializeOptions := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

    if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
        return err
    }

    if err := gopacket.SerializeLayers(serializeBuffer, serializeOptions,
        ip,
        udp,
        payload,
    ); err != nil {
        return err
    }

    // send
    if send, err := self.ipConn.WriteToIP(serializeBuffer.Bytes(), &self.dstAddr); err != nil {
        self.stats.SendErrors++
    } else {
        self.stats.SendPackets++
        self.stats.SendBytes += uint(send)
    }

    return nil
}

// serialize and send from Packet
func (self *Send) sendPacket(packet Packet) error {
    // packet structure
    ip := layers.IPv4{
        Version:    4,
        TTL:        IP_TTL,
        Protocol:   layers.IPProtocolUDP,

        SrcIP:      packet.SrcIP,
        DstIP:      packet.DstIP,
    }
    udp := layers.UDP{
        SrcPort:    layers.UDPPort(packet.SrcPort),
        DstPort:    layers.UDPPort(packet.DstPort),
    }
    payload := gopacket.Payload(packet.Payload.Pack(packet.PayloadSize))

    return self.sendLayers(&ip, &udp, &payload)
}

func (self *Send) GiveStats() chan SendStats {
    self.statsChan = make(chan SendStats)

    return self.statsChan
}

// Generate a sequence of *Packet
func (self *Send) run(rate uint, size uint) error {
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

        if err := self.sendPacket(packet); err != nil {
            return err
        }

        payload.Seq++

        // stats
        if self.statsChan != nil {
            self.statsChan <- self.stats
        }
    }
}

func (self *Send) Run() error {
    // TODO: reconfigure
    return self.run(self.rate, self.size)
}
