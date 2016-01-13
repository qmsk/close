package logs

import (
    "log"
)

const LOGS_BUFFER = 100

// Forward logs sent by Logs, with buffering and dropping to deal with slow followers
// The logic to actually send out messages is implemented by having a goroutine read from writeChan.
// See e.g. ./websocket.go
type logFollower struct {
    logs        *Logs

    // LogMsg's sent by Logs.run()
    msgChan     chan LogMsg

    // whatever writing is consuming these; messages will be dropped if this blocks
    writeChan   chan LogMsg
}

func newFollower(logs *Logs) logFollower {
    follower := logFollower{
        logs:       logs,
        msgChan:    make(chan LogMsg, LOGS_BUFFER),
        writeChan:  make(chan LogMsg),
    }

    go follower.run()

    logs.subscribeChan <- follower.msgChan

    return follower
}

// non-blocking write to follower
func (self logFollower) run() {
    defer close(self.writeChan)

    var drops uint

    for msg := range self.msgChan {
        msg.Dropped = drops

        select {
        case self.writeChan <- msg:
            drops = 0

        default:
            drops++
        }
    }
}

// Close on error
// Should be safe to call from multiple goroutines (both reader() and writer())!
func (self logFollower) close(err error) {
    if err != nil {
        log.Printf("logFollower %v: %v\n", self, err)
    }

    // unsubscribing causes Logs.run() to close the writeChan, which lets logFollower.run() exit
    self.logs.unsubscribeChan <- self.msgChan
}
