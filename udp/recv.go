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

    PacketClock     time.Duration   // 
    PacketStart     uint64          // first packet
    PacketSeq       uint64          // most recent packet
    PacketErrors    uint            // invalid packets
    PacketCount     uint            // in-sequence packets
    PacketSkip      uint            // out-of-sequence packets

    Recv            SockStats
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

            } else if packet.Payload.Seq > payload.Seq {
                self.stats.PacketSeq = packet.Payload.Seq
                self.stats.PacketCount++

                payload.Seq = packet.Payload.Seq

            } else {
                log.Printf("Skip %v <= %v\n", packet.Payload.Seq, payload.Seq)

                self.stats.PacketSkip++
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
