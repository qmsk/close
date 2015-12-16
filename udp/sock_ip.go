package udp

// net.IPConn based socket. Supports send with arbitrary source

import (
    "fmt"
    "github.com/google/gopacket"
    "golang.org/x/net/ipv4"
    "github.com/google/gopacket/layers"
    "net"
)

const IP_TTL uint8 = 64

// used for serializing packets with transport-layer checksums
type SerializableNetworkLayer interface {
    gopacket.NetworkLayer
    gopacket.SerializableLayer
}


type SockIP struct {
    ipAddr      net.IPAddr
    udpAddr     net.UDPAddr

    ipConn      *net.IPConn
    rawConn     *ipv4.RawConn

    stats       SockStats
}

func (self *SockIP) init(dstAddr string) error {
    // resolve
    if udpAddr, err := net.ResolveUDPAddr("udp", dstAddr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", dstAddr, err)
    } else {
        self.ipAddr = net.IPAddr{IP: udpAddr.IP, Zone: udpAddr.Zone}
        self.udpAddr = *udpAddr
    }

    // setup
    if ip4 := self.udpAddr.IP.To4(); ip4 != nil {
        if ipConn, err := net.ListenIP("ip4:udp", nil); err != nil {
            return fmt.Errorf("ListenIP: %v", err)
        } else {
            self.ipConn = ipConn
        }

        // sets the IPConn into raw mode (IPv4 headers included)
        if rawConn, err := ipv4.NewRawConn(self.ipConn); err != nil {
            return fmt.Errorf("NewRawConn: %v", err)
        } else {
            self.rawConn = rawConn
        }

    } else if ip6 := self.udpAddr.IP.To16(); ip6 != nil {
        return fmt.Errorf("TODO: IPv6")
    } else {
        return fmt.Errorf("Invalid IP family")
    }

    return nil
}

// probe the source address the kernel would select for our destination
func (self *SockIP) probeSource() (*net.UDPAddr, error) {
    if udpConn, err := net.DialUDP("udp", nil, &self.udpAddr); err != nil {
        return nil, err
    } else {
        return udpConn.LocalAddr().(*net.UDPAddr), nil
    }
}



// serialize and send from gopacket layers
func (self *SockIP) sendLayers(ip SerializableNetworkLayer, udp *layers.UDP, payload *gopacket.Payload) error {
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
    if send, err := self.ipConn.WriteToIP(serializeBuffer.Bytes(), &self.ipAddr); err != nil {
        self.stats.Errors++
    } else {
        self.stats.Packets++
        self.stats.Bytes += uint(send)
    }

    return nil
}

// serialize and send from Packet
func (self *SockIP) send(packet Packet) error {
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

func (self *SockIP) resetStats() {
    self.stats = SockStats{}
}
func (self *SockIP) getStats() SockStats {
    return self.stats
}
