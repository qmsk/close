package udp

import (
    "encoding/binary"
    "fmt"
)

const PORT uint = 1337 // used as destination port
const PAYLOAD_SIZE = 16 // XXX: or smaller with varint?

type Payload struct {
    // Chosen by the sender, kept the same for all consecutive packets in the same sequence.
    // The receiver requires this to be unique per sender for reliable sequence tracking.
    // Required since a sender may transmit from different randomized source addresses, possibly overlapping with other senders.
    ID          uint64

    // The sender sends packets with consective sequence numbers, starting from zero.
    // Used by the sender to count received/missed/reordered packets, per ID.
    Seq         uint64
}

func (self Payload) Pack(dataSize uint) []byte {
    if dataSize < PAYLOAD_SIZE {
        dataSize = PAYLOAD_SIZE
    }

    // with trailing zeros
    buf := make([]byte, dataSize)

    binary.BigEndian.PutUint64(buf[0:8], self.ID)
    binary.BigEndian.PutUint64(buf[8:16], self.Seq)

    return buf
}

func (self *Payload) Unpack(buf []byte) error {
    if len(buf) < PAYLOAD_SIZE {
        return fmt.Errorf("Short payload")
    }

    self.ID = binary.BigEndian.Uint64(buf[0:8])
    self.Seq = binary.BigEndian.Uint64(buf[8:16])

    return nil
}
