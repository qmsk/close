package udp

import (
    "fmt"
    "github.com/google/gopacket"
    "golang.org/x/net/ipv4"
    "github.com/google/gopacket/layers"
    "log"
    "net"
    "time"
)

const SOURCE_PORT uint = 0
const PORT uint = 1337
const PORT_BITS uint = 0
const IP_TTL uint8 = 64

type SenderConfig struct {
    DestAddr        string // host
    DestPort        uint
    SourceNet       string // host/mask
    SourcePort      uint
    SourcePortBits  uint
}

type Sender struct {
    dstAddr     net.IPAddr
    dstIP       net.IP
    dstPort     uint16
    srcAddr     net.IP
    srcAddrBits uint
    srcPort     RandPort

    ipConn      *net.IPConn
    rawConn     *ipv4.RawConn

    stats       struct {
        rateUnderrun    uint
        sendErrors      uint
        sendPackets     uint
        sendBytes       uint
    }
}

func NewSender(config SenderConfig) (*Sender, error) {
    sender := &Sender{

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

func (self *Sender) init(config SenderConfig) error {
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

    return nil
}

type SerializableNetworkLayer interface {
    gopacket.NetworkLayer
    gopacket.SerializableLayer
}

// serialize and send from gopacket layers
func (self *Sender) sendLayers(ip SerializableNetworkLayer, udp *layers.UDP, payload *gopacket.Payload) error {
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
        self.stats.sendErrors++
    } else {
        self.stats.sendPackets++
        self.stats.sendBytes += uint(send)
    }

    return nil
}

// serialize and send from Packet
func (self *Sender) sendPacket(packet Packet) error {
    // packet structure
    ip := layers.IPv4{
        TTL:        IP_TTL,
        Protocol:   layers.IPProtocolUDP,

        SrcIP:      packet.SrcIP,
        DstIP:      packet.DstIP,
    }
    udp := layers.UDP{
        SrcPort:    layers.UDPPort(packet.SrcPort),
        DstPort:    layers.UDPPort(packet.DstPort),
    }
    payload := gopacket.Payload(packet.Payload.Pack())

    return self.sendLayers(&ip, &udp, &payload)
}

// Generate a sequence of *Packet
func (self *Sender) Run(rate uint) error {
    startTime := time.Now()

    payload := Payload{
        Start:  startTime.Unix(),
        Seq:    0,
    }

    for {
        duration := time.Since(startTime)

        // schedule
        packetDuration := time.Duration(payload.Seq) * time.Second / time.Duration(rate)

        if packetDuration > duration {
            time.Sleep(packetDuration - duration)
        } else {
            self.stats.rateUnderrun++
        }

        log.Printf("%5.2f send @%d = %5.2f/s", duration.Seconds(), payload.Seq,
            float64(payload.Seq) / duration.Seconds(),
        )

        packet := Packet{
            SrcIP:      self.srcAddr,
            SrcPort:    self.srcPort.Port(),
            DstIP:      self.dstIP,
            DstPort:    self.dstPort,

            Payload:    payload,
        }

        if err := self.sendPacket(packet); err != nil {
            return err
        }

        payload.Seq++
    }
}
