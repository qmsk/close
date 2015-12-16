package udp

import (
    "encoding/binary"
    "bytes"
    "net"
)

const PORT uint = 1337
const PACKET_MTU = 1500 // XXX: not including IP overhead..?

type Packet struct {
    SrcIP       net.IP
    SrcPort     uint16
    DstIP       net.IP
    DstPort     uint16

    PayloadSize uint
    Payload     Payload
}

type Payload struct {
    Start       int64
    Seq         uint64
}

func (self Payload) Pack(dataSize uint) []byte {
    buffer := new(bytes.Buffer)

    if err := binary.Write(buffer, binary.BigEndian, self); err != nil {
        panic(err)
    }

    bufferLen := uint(buffer.Len())

    if bufferLen < dataSize {
        data := make([]byte, dataSize - bufferLen)

        if err := binary.Write(buffer, binary.BigEndian, data); err != nil {
            panic(err)
        }
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
