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
    // return stats and reset
    takeStats() SockStats

    send(packet Packet) error

    // return source address for the destination
    probeSource() (*net.UDPAddr, error)
}

type SockRecv interface {
    // return stats and reset
    takeStats() SockStats

    recv() (Packet, error)
}
