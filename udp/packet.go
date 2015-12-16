package udp

import (
    "fmt"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "net"
)

// used for serializing packets with transport-layer checksums
type SerializableNetworkLayer interface {
    gopacket.NetworkLayer
    gopacket.SerializableLayer
}

const PACKET_MTU = 1500 // XXX: not including IP overhead..?
const PACKET_TTL uint8 = 64

type Packet struct {
    SrcIP       net.IP
    SrcPort     uint16
    DstIP       net.IP
    DstPort     uint16

    PayloadSize uint
    Payload     Payload
}

// Pack into an IP+UDP+Payload packet
func (self *Packet) PackIP() ([]byte, error) {
    var ip SerializableNetworkLayer

    if src4, dst4 := self.SrcIP.To4(), self.DstIP.To4(); src4 != nil && dst4 != nil {
        // packet structure
        ip = &layers.IPv4{
            Version:    4,
            TTL:        PACKET_TTL,
            Protocol:   layers.IPProtocolUDP,

            SrcIP:      src4,
            DstIP:      dst4,
        }
    } else {
        return nil, fmt.Errorf("Unsupported IP address")
    }

    udp := layers.UDP{
        SrcPort:    layers.UDPPort(self.SrcPort),
        DstPort:    layers.UDPPort(self.DstPort),
    }
    payload := gopacket.Payload(self.Payload.Pack(self.PayloadSize))

    // serialize
    serializeBuffer := gopacket.NewSerializeBuffer()
    serializeOptions := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

    if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
        return nil, err
    }

    if err := gopacket.SerializeLayers(serializeBuffer, serializeOptions,
        ip,
        &udp,
        &payload,
    ); err != nil {
        return nil, err
    }

    return serializeBuffer.Bytes(), nil
}
