package udp

import (
    "fmt"
    "log"
    "net"
)

type ReceiverConfig struct {
    ListenAddr        string
}

type Receiver struct {
    udpAddr net.UDPAddr
    udpConn *net.UDPConn

    stats struct {
        recvErrors      uint
        recvPackets     uint
        recvBytes       uint

        packetErrors    uint
        packetCount     uint
        packetSkip      uint
    }
}

func NewReceiver(config ReceiverConfig) (*Receiver, error) {
    receiver := &Receiver{

    }

    if err := receiver.init(config); err != nil {
        return nil, err
    } else {
        return receiver, nil
    }
}

func (self *Receiver) init(config ReceiverConfig) error {
    if udpAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr); err != nil {
        return fmt.Errorf("Resolve ListenAddr %v: %v", config.ListenAddr, err)
    } else if udpConn, err := net.ListenUDP("udp", udpAddr); err != nil {
        return fmt.Errorf("Listen UDP %v: %v", udpAddr, err)
    } else {
        self.udpAddr = *udpAddr
        self.udpConn = udpConn
    }

    return nil
}

func (self *Receiver) recv() (Packet, error) {
    var packet Packet
    buf := make([]byte, packetSize)

    if recvSize, srcAddr, err := self.udpConn.ReadFromUDP(buf); err != nil {
        self.stats.recvErrors++

        return packet, err
    } else {
        self.stats.recvPackets++
        self.stats.recvBytes += uint(recvSize)

        if err := packet.Payload.Unpack(buf[:recvSize]); err != nil {
            self.stats.packetErrors++

            return packet, err
        } else {

            packet.SrcIP = srcAddr.IP
            packet.SrcPort = uint16(srcAddr.Port)
            packet.DstIP = self.udpAddr.IP // XXX
            packet.DstPort = uint16(self.udpAddr.Port)

            return packet, nil
        }
    }
}

func (self *Receiver) Run() error {
    for {
        var state Payload

        log.Printf("Recv...\n")

        if packet, err := self.recv(); err != nil {
            log.Printf("Recv error: %v\n", err)
        } else {
            if packet.Payload.Start != state.Start {
                state.Start = packet.Payload.Start
                state.Seq = packet.Payload.Seq
                self.stats.packetCount = 0

                log.Printf("Recv Start @%v from seq=%v\n", state.Start, state.Seq)

            } else if packet.Payload.Seq > state.Seq {
                self.stats.packetCount++

                log.Printf("Recv %8d/%8d @ %d\n", self.stats.packetCount, packet.Payload.Seq - state.Seq)

                state.Seq = packet.Payload.Seq
            } else {
                self.stats.packetSkip++

                log.Printf("Skip at seq=%v < %v\n", packet.Payload.Seq, state.Seq)
            }
        }
    }

    return nil
}
