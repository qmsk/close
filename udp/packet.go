package udp

import (
    "encoding/binary"
    "bytes"
    "net"
)

var packetSize int = binary.Size(Payload{})

type Packet struct {
    SrcIP       net.IP
    SrcPort     uint16
    DstIP       net.IP
    DstPort     uint16

    Payload     Payload
}

type Payload struct {
    Start       int64
    Seq         uint64
}

func (self Payload) Pack() []byte {
    buffer := new(bytes.Buffer)

    if err := binary.Write(buffer, binary.BigEndian, self); err != nil {
        panic(err)
    }

    return buffer.Bytes()
}

func (self *Payload) Unpack(buf []byte) error {
    reader := bytes.NewReader(buf)

    if err := binary.Read(reader, binary.BigEndian, self); err != nil {
        return err
    }

    return nil
}
