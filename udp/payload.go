package udp

import (
    "encoding/binary"
    "bytes"
    "fmt"
    "encoding/hex"
    "net"
    "crypto/sha256"
)

const PORT uint = 1337 // used as destination port
const PAYLOAD_SIZE = 16 // XXX: or smaller with varint?

type ID [8]byte

func (self ID) String() string {
    return hex.EncodeToString(self[:])
}

type Payload struct {
    // Chosen by the sender, kept the same for all consecutive packets in the same sequence.
    // The receiver requires this to be unique per sender for reliable sequence tracking.
    // Required since a sender may transmit from different randomized source addresses, possibly overlapping with other senders.
    ID          ID

    // The sender sends packets with consective sequence numbers, starting from zero.
    // Used by the sender to count received/missed/reordered packets, per ID.
    Seq         uint64
}

// Parse an ID from hexadecimal
func parseID(s string) (id ID, err error) {
    if buf, err := hex.DecodeString(s); err != nil {
        return id, err
    } else {
        copy(id[:], buf[0:8])
        return id, nil
    }
}

// Generate an ID from source address
func genID(ip net.IP, port uint16) (ID, error) {
    hash := sha256.New()

    binary.Write(hash, binary.BigEndian, ip)
    binary.Write(hash, binary.BigEndian, port)

    var hashSum []byte

    hashSum = hash.Sum(hashSum)

    // truncated hash-sum into 64-bit id 
    var id ID

    if err := binary.Read(bytes.NewReader(hashSum), binary.BigEndian, &id); err != nil {
        return id, err
    }

    return id, nil
}

func (self Payload) Pack(dataSize uint) []byte {
    if dataSize < PAYLOAD_SIZE {
        dataSize = PAYLOAD_SIZE
    }

    // with trailing zeros
    buf := make([]byte, dataSize)

    copy(buf[0:8], self.ID[:])
    binary.BigEndian.PutUint64(buf[8:16], self.Seq)

    return buf
}

func (self *Payload) Unpack(buf []byte) error {
    if len(buf) < PAYLOAD_SIZE {
        return fmt.Errorf("Short payload")
    }

    copy(self.ID[:], buf[0:8])
    self.Seq = binary.BigEndian.Uint64(buf[8:16])

    return nil
}
