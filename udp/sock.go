package udp

import (
    "net"
)

type SockStats struct {
    Errors      uint
    Packets     uint
    Bytes       uint    // only includes Payload
}

type SockSend interface {
    resetStats()
    getStats() SockStats
    send(packet Packet) error
    probeSource() (*net.UDPAddr, error)
}

type SockRecv interface {
    resetStats()
    getStats() SockStats
    recv() (Packet, error)
}
