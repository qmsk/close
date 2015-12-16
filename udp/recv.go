package udp

import (
    "fmt"
    "log"
    "time"
)

type RecvConfig struct {
    ListenAddr        string
}

type Recv struct {
    sockRecv    SockRecv

    statsChan   chan RecvStats
    stats       RecvStats
}

type RecvStats struct {
    StartTime       time.Time
    StartSeq        uint64

    PacketTime      time.Time
    PacketSeq       uint64
    PacketErrors    uint
    PacketCount     uint
    PacketSkip      uint

    Recv            SockStats
}

func (self RecvStats) String() string {
    clock := self.PacketTime.Sub(self.StartTime)
    packetOffset := self.PacketSeq - self.StartSeq
    packetRate := float64(self.Recv.Packets) / clock.Seconds()
    packetThroughput := float64(self.Recv.Bytes) / 1000 / 1000 * 8 / clock.Seconds()
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
    sockUDP := &SockUDP{}
    if err := sockUDP.initListen(config.ListenAddr); err != nil {
        return err
    }

    self.sockRecv = sockUDP

    return nil
}

func (self *Recv) GiveStats() chan RecvStats {
    self.statsChan = make(chan RecvStats)

    return self.statsChan
}

func (self *Recv) Run() error {
    var payload Payload

    for {
        if packet, err := self.sockRecv.recv(); err != nil {
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
            self.stats.Recv = self.sockRecv.getStats()
            self.statsChan <- self.stats
        }
    }

    return nil
}
