package udp

type SockStats struct {
    Errors      uint
    Packets     uint
    Bytes       uint    // only includes Payload
}

type SockSend interface {
    getStats() SockStats
    send(packet Packet) error
}

type SockRecv interface {
    getStats() SockStats
    recv() (Packet, error)
}
