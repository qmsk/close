package udp

import (
    "encoding/binary"
    "fmt"
    "net"
)

const PORT uint = 1337
const PACKET_MTU = 1500 // XXX: not including IP overhead..?
const PAYLOAD_SIZE = 16 // XXX: or smaller with varint?

type Packet struct {
    SrcIP       net.IP
    SrcPort     uint16
    DstIP       net.IP
    DstPort     uint16

    PayloadSize uint
    Payload     Payload
}

type Payload struct {
    Start       uint64
    Seq         uint64
}

func (self Payload) Pack(dataSize uint) []byte {
    if dataSize < PAYLOAD_SIZE {
        dataSize = PAYLOAD_SIZE
    }

    // with trailing zeros
    buf := make([]byte, dataSize)

    binary.BigEndian.PutUint64(buf[0:8], self.Start)
    binary.BigEndian.PutUint64(buf[8:16], self.Seq)

    return buf
}

func (self *Payload) Unpack(buf []byte) error {
    if len(buf) < PAYLOAD_SIZE {
        return fmt.Errorf("Short payload")
    }

    self.Start = binary.BigEndian.Uint64(buf[0:8])
    self.Seq = binary.BigEndian.Uint64(buf[8:16])

    return nil
}
