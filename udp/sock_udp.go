package udp

// Trivial net.UDPConn based socket. Supports send and recv

import (
    "fmt"
    "net"
)

type SockUDP struct {
    udpAddr     net.UDPAddr
    udpConn     *net.UDPConn

    stats       SockStats
}

func (self *SockUDP) initListen(addr string) error {
    if udpAddr, err := net.ResolveUDPAddr("udp", addr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", addr, err)
    } else if udpConn, err := net.ListenUDP("udp", udpAddr); err != nil {
        return fmt.Errorf("Listen UDP %v: %v", udpAddr, err)
    } else {
        self.udpAddr = *udpAddr
        self.udpConn = udpConn
    }

    return nil
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

func (self *SockUDP) send(packet Packet) error {
    dstAddr := net.UDPAddr{IP: packet.DstIP, Port: int(packet.DstPort)}
    payload := packet.Payload.Pack(packet.PayloadSize)

    if sendSize, err := self.udpConn.WriteToUDP(payload, &dstAddr); err != nil {
        self.stats.Errors++

        return err
    } else {
        self.stats.Packets++
        self.stats.Bytes += uint(sendSize)

        return nil
    }
}

func (self *SockUDP) getStats() SockStats {
    return self.stats
}
