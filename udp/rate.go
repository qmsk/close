package udp

import (
    "time"
)

type RateStats struct {
    SleepDuration   time.Duration   // total time slept
    UnderrunCount   uint            // count of timing underruns (no sleep)
    Count           uint            // count of timing ticks
}

type RateClock struct {
    rateChan    chan time.Time
    stats       RateStats

    start       time.Time
    rate        uint
    stop        uint                // stop once offset
    offset      uint
}

func (self *RateClock) run() {
    for ; self.stop == 0 || self.offset < self.stop; self.offset++ {
        if self.rate != 0 {
            // scheduled time for next packet
            targetClock := time.Duration(self.offset) * time.Second / time.Duration(self.rate)
            currentClock := time.Since(self.start)

            skew := targetClock - currentClock

            if skew > 0 {
                // slow down
                time.Sleep(skew)

                self.stats.SleepDuration += skew
            } else {
                // catch up
                self.stats.UnderrunCount++
            }
        }

        self.rateChan <- time.Now()

        self.stats.Count++
    }
}

func (self *RateClock) Start(rate uint, count uint) chan time.Time {
    self.rateChan = make(chan time.Time)

    self.Set(rate, count)

    // start
    go self.run()

    return self.rateChan
}

// change running timer
func (self *RateClock) Set(rate uint, count uint) {
    // XXX: unsafe if running...
    self.start = time.Now()
    self.rate = rate
    self.stop = count
    self.offset = 0
}

func (self *RateClock) Stop() {
    // XXX
    self.stop = self.offset
}

func (self *RateClock) takeStats() RateStats {
    // XXX: safe?
    stats := self.stats
    self.stats = RateStats{}

    return stats
}

