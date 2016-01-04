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
}

func (self *RateClock) run(rate uint, count uint) {
    startTime := time.Now()
    offset := uint(0)

    for ; count == 0 || offset < count; offset++ {
        if rate != 0 {
            // scheduled time for next packet
            targetClock := time.Duration(offset) * time.Second / time.Duration(rate)
            currentClock := time.Since(startTime)

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

func (self *RateClock) Run(rate uint, count uint) chan time.Time {
    self.rateChan = make(chan time.Time)

    go self.run(rate, count)

    return self.rateChan
}

func (self *RateClock) takeStats() RateStats {
    // XXX: safe?
    stats := self.stats
    self.stats = RateStats{}

    return stats
}

