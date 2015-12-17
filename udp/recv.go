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

    stats           RecvStats
    statsChan       chan RecvStats
    statsInterval   time.Duration
}

type RecvStats struct {
    Time            time.Time       // stats were reset
    Duration        time.Duration   // stats were collected

    PacketStart     uint64          // first packet
    PacketSeq       uint64          // most recent packet
    PacketErrors    uint            // invalid packets
    PacketCount     uint            // in-sequence packets
    PacketSkips     uint            // skipped in-sequence packets
    PacketDups      uint            // out-of-sequence packets

    Recv            SockStats
}

// check if we have any received packets to report on
func (self RecvStats) Valid() bool {
    return (self.PacketSeq > 0)
}

// proportion of delivered packets, not counting reordered or duplicated packets
// this ratio only applies when .Valid()
func (self RecvStats) PacketWin() float64 {
    return float64(self.PacketCount) / float64(self.PacketSeq - self.PacketStart)
}

// proportion of lost or reordered packets, not counting duplicates
// this ratio only applies when .Valid()
func (self RecvStats) PacketLoss() float64 {
    return float64(self.PacketSkips) / float64(self.PacketSeq - self.PacketStart)
}

func (self RecvStats) String() string {
    packetOffset := self.PacketSeq - self.PacketStart
    packetRate := float64(self.Recv.Packets) / self.Duration.Seconds()
    packetThroughput := float64(self.Recv.Bytes) / 1000 / 1000 * 8 / self.Duration.Seconds()
    packetLoss := 1.0 - float64(self.PacketCount) / float64(packetOffset)

    return fmt.Sprintf("%5.2f: recv %10d / %10d = %10.2f/s %8.2fMb/s @ %6.2f%% loss",
        self.Duration.Seconds(),
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

func (self *Recv) GiveStats(interval time.Duration) chan RecvStats {
    self.statsChan = make(chan RecvStats)
    self.statsInterval = interval

    return self.statsChan
}

func (self *Recv) Run() error {
    var payload Payload

    self.stats = RecvStats{
        Time:   time.Now(),
    }

    for {
        if packet, err := self.sockRecv.recv(); err != nil {
            log.Printf("Recv error: %v\n", err)
        } else {
            if packet.Payload.Start != payload.Start {
                log.Printf("Start from %v: %v\n", packet.Payload.Seq, packet.Payload.Start)

                payload = packet.Payload

                self.stats.PacketStart = payload.Seq
                self.stats.PacketSeq = payload.Seq
                self.stats.PacketCount++

            } else if packet.Payload.Seq > payload.Seq {
                self.stats.PacketSeq = packet.Payload.Seq
                self.stats.PacketSkips += uint(packet.Payload.Seq - payload.Seq - 1) // normally 0 if delivered in sequence
                self.stats.PacketCount++

                payload.Seq = packet.Payload.Seq

            } else {
                log.Printf("Skip %v <= %v\n", packet.Payload.Seq, payload.Seq)

                self.stats.PacketDups++
            }
        }

        // stats
        self.stats.Duration = time.Since(self.stats.Time)

        if self.statsChan != nil && self.stats.Duration >= self.statsInterval {
            self.stats.Recv = self.sockRecv.takeStats()
            self.statsChan <- self.stats

            self.stats = RecvStats{
                Time:           time.Now(),

                PacketStart:    payload.Seq,
            }
        }
    }

    return nil
}
