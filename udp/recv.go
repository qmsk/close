package udp

import (
    "fmt"
    "log"
    "net"
    "time"
)

type RecvConfig struct {
    ListenAddr        string
}

type Recv struct {
    udpAddr     net.UDPAddr
    udpConn     *net.UDPConn

    statsChan   chan RecvStats
    stats       RecvStats
}

type RecvStats struct {
    StartTime       time.Time
    StartSeq        uint64

    RecvErrors      uint
    RecvPackets     uint
    RecvBytes       uint    // only includes Payload

    PacketTime      time.Time
    PacketSeq       uint64
    PacketErrors    uint
    PacketCount     uint
    PacketSkip      uint
}

func (self RecvStats) String() string {
    clock := self.PacketTime.Sub(self.StartTime)
    packetOffset := self.PacketSeq - self.StartSeq
    packetRate := float64(self.RecvPackets) / clock.Seconds()
    packetThroughput := float64(self.RecvBytes) / 1000 / 1000 * 8 / clock.Seconds()
    packetLoss := 1.0 - float64(self.PacketCount) / float64(packetOffset)

    return fmt.Sprintf("%8.2f: recv %8d / %8d = %8.2f/s %8.2fMb/s @ %6.2f%% loss",
        clock.Seconds(),
        self.PacketCount, packetOffset,
        packetRate, packetThroughput,
        packetLoss * 100.0,
    )
}

func NewRecv(config RecvConfig) (*Recv, error) {
    receiver := &Recv{

    }

    if err := receiver.init(config); err != nil {
        return nil, err
    } else {
        return receiver, nil
    }
}

func (self *Recv) init(config RecvConfig) error {
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

func (self *Recv) recv() (Packet, error) {
    var packet Packet
    buf := make([]byte, PACKET_MTU)

    if recvSize, srcAddr, err := self.udpConn.ReadFromUDP(buf); err != nil {
        self.stats.RecvErrors++

        return packet, err
    } else {
        self.stats.RecvPackets++
        self.stats.RecvBytes += uint(recvSize)

        if err := packet.Payload.Unpack(buf[:recvSize]); err != nil {
            self.stats.PacketErrors++

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

func (self *Recv) GiveStats() chan RecvStats {
    self.statsChan = make(chan RecvStats)

    return self.statsChan
}

func (self *Recv) Run() error {
    var payload Payload

    for {
        if packet, err := self.recv(); err != nil {
            log.Printf("Recv error: %v\n", err)
        } else {
            self.stats.PacketTime = time.Now()

            if packet.Payload.Start != payload.Start {
                log.Printf("Start from %v: %v\n", packet.Payload.Seq, packet.Payload.Start)

                // reset
                self.stats = RecvStats{
                    StartTime:  self.stats.PacketTime,
                    StartSeq:   packet.Payload.Seq,
                }

            } else if packet.Payload.Seq > payload.Seq {
                self.stats.PacketSeq = packet.Payload.Seq
                self.stats.PacketCount++

            } else {
                log.Printf("Skip %v <= %v\n", packet.Payload.Seq, payload.Seq)

                self.stats.PacketSkip++
            }

            payload = packet.Payload
        }

        if self.statsChan != nil {
            self.statsChan <- self.stats
        }
    }

    return nil
}
