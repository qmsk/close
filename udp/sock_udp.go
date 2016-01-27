package udp

// Trivial net.UDPConn based socket. Supports send and recv

import (
    "fmt"
    "net"
)

type SockUDP struct {
    udpAddr     *net.UDPAddr
    udpConn     *net.UDPConn

    stats       *SockStats
}

func (self *SockUDP) init() {
    self.stats = &SockStats{}
}

func (self *SockUDP) String() string {
    return fmt.Sprintf("%v", self.udpAddr)
}

func (self *SockUDP) initDial(addr string) error {
    self.init()

    if udpAddr, err := net.ResolveUDPAddr("udp", addr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", addr, err)
    } else if udpConn, err := net.DialUDP("udp", nil, udpAddr); err != nil {
        return fmt.Errorf("Dial UDP %v: %v", udpAddr, err)
    } else {
        self.udpAddr = udpAddr
        self.udpConn = udpConn
    }

    return nil
}

func (self *SockUDP) initListen(addr string) error {
    self.init()

    if udpAddr, err := net.ResolveUDPAddr("udp", addr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", addr, err)
    } else if udpConn, err := net.ListenUDP("udp", udpAddr); err != nil {
        return fmt.Errorf("Listen UDP %v: %v", udpAddr, err)
    } else {
        self.udpAddr = udpAddr
        self.udpConn = udpConn
    }

    return nil
}

func (self *SockUDP) probeSource() (*net.UDPAddr, error) {
    return self.udpConn.LocalAddr().(*net.UDPAddr), nil
}

func (self *SockUDP) recv() (Packet, error) {
    var packet Packet
    buf := make([]byte, PACKET_MTU)

    if recvSize, srcAddr, err := self.udpConn.ReadFromUDP(buf); err != nil {
        self.stats.Errors++

        return packet, err
    } else {
        self.stats.Packets++
        self.stats.Bytes += uint(recvSize)

        if err := packet.Payload.Unpack(buf[:recvSize]); err != nil {
            return packet, err
        } else {
            packet.SrcIP = srcAddr.IP
            packet.SrcPort = uint16(srcAddr.Port)
            packet.DstIP = self.udpAddr.IP // XXX
            packet.DstPort = uint16(self.udpAddr.Port)
            packet.PayloadSize = uint(recvSize)

            return packet, nil
        }
    }
}

func (self *SockUDP) recvChan() chan Packet {
    recvChan := make(chan Packet)

    go func() {
        for {
            if packet, err := self.recv(); err != nil {
                // XXX: wut
            } else {
                recvChan <- packet
            }
        }
    }()

    return recvChan
}

func (self *SockUDP) send(packet Packet) error {
    payload := packet.Payload.Pack(packet.PayloadSize)

    if sendSize, err := self.udpConn.Write(payload); err != nil {
        self.stats.Errors++
    } else {
        self.stats.Packets++
        self.stats.Bytes += uint(sendSize)
    }

    return nil
}

func (self *SockUDP) useStats(stats *SockStats) {
    self.stats = stats
}
