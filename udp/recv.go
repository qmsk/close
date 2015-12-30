package udp

import (
    "fmt"
    "log"
    "close/stats"
    "time"
)

type RecvConfig struct {
    ListenAddr        string
}

type Recv struct {
    sockRecv    SockRecv

    statsChan       chan stats.Stats
    statsInterval   time.Duration

    recvState       map[uint64]*RecvState
}

type RecvState struct {
    id              uint64
    seq             uint64
    stats           RecvStats
}

type RecvStats struct {
    ID              uint64          // from recv ID
    Time            time.Time       // stats were init/reset
    Duration        time.Duration   // stats were collected

    PacketTime      time.Time       // time of most recent packet
    PacketStart     uint64          // stats for packets following this seq (non-inclusive)
    PacketSeq       uint64          // most recent packet
    PacketSize      uint            // total size of received packets
    PacketErrors    uint            // invalid packets
    PacketCount     uint            // in-sequence packets
    PacketSkips     uint            // skipped in-sequence packets
    PacketDups      uint            // out-of-sequence packets
}

func (self RecvStats) StatsInstance() string {
    return fmt.Sprintf("%016x", self.ID)
}

func (self RecvStats) StatsTime() time.Time {
    return self.Time
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

func (self RecvStats) StatsFields() map[string]interface{} {
    fields := map[string]interface{} {
        "packet_errors": self.PacketErrors,
        "packets": self.PacketCount,
        "packet_skips": self.PacketSkips,
        "packet_dups": self.PacketDups,
    }

    if self.Valid() {
        fields["packet_win"] = self.PacketWin()
        fields["packet_loss"] = self.PacketLoss()
    }

    return fields
}

func (self RecvStats) String() string {
    packetOffset := self.PacketSeq - self.PacketStart
    packetRate := float64(self.PacketCount) / self.Duration.Seconds()
    packetThroughput := float64(self.PacketSize) / 1000 / 1000 * 8 / self.Duration.Seconds()
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
        recvState: make(map[uint64]*RecvState),
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

func (self *Recv) GiveStats(interval time.Duration) chan stats.Stats {
    self.statsChan = make(chan stats.Stats)
    self.statsInterval = interval

    return self.statsChan
}

func (self *Recv) Run() error {
    recvChan := self.sockRecv.recvChan()
    statsTick := time.Tick(self.statsInterval)

    for {
        select {
        case packet := <-recvChan:
            if recvState, exists := self.recvState[packet.Payload.ID]; !exists {
                self.recvState[packet.Payload.ID] = self.makeState(packet)
            } else {
                recvState.handlePacket(packet)
            }
        case <- statsTick:
            for id, recvState := range self.recvState {
                if recvState.alive() {
                    self.statsChan <- recvState.takeStats()
                } else {
                    // cleanup
                    delete(self.recvState, id)
                }
            }
        }
    }

    return nil
}

// First packet received for Payload.ID
func (self *Recv) makeState(packet Packet) *RecvState {
    log.Printf("Start from %v:%v: %8x@%v\n", packet.SrcIP, packet.SrcPort, packet.Payload.ID, packet.Payload.Seq)

    // skip first packet
    return &RecvState{
        id:     packet.Payload.ID,
        seq:    packet.Payload.Seq,
        stats:  RecvStats{
            ID:             packet.Payload.ID,
            Time:           time.Now(),
            PacketStart:    packet.Payload.Seq,
        },
    }
}

func (self *RecvState) handlePacket(packet Packet) {
    self.stats.PacketTime = time.Now()

    if packet.Payload.Seq > self.seq {
        self.stats.PacketSeq = packet.Payload.Seq
        self.stats.PacketSkips += uint(packet.Payload.Seq - self.seq - 1) // normally 0 if delivered in sequence
        self.stats.PacketCount++
        self.stats.PacketSize += packet.PayloadSize

        self.seq = packet.Payload.Seq

    } else {
        log.Printf("Skip %v <= %v\n", packet.Payload.Seq, self.seq)

        self.stats.PacketDups++
    }
}

func (self *RecvState) alive() bool {
    return self.stats.PacketSeq > self.stats.PacketStart
}

func (self *RecvState) takeStats() RecvStats {
    stats := self.stats
    stats.Duration = time.Since(stats.Time)

    self.stats = RecvStats{
        ID:             self.id,
        Time:           time.Now(),

        PacketStart:    self.seq,
    }

    return stats
}
