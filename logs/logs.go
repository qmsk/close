package logs

import (
    "io"
    "log"
    "os"
)

// Logs distributes LogMsg's from logWriter's to active logFollower's
type Logs struct {
    writeChan           chan LogMsg
    subscribeChan       chan chan LogMsg
    unsubscribeChan     chan chan LogMsg
}

func New() (*Logs, error) {
    self := &Logs{
        writeChan:          make(chan LogMsg),
        subscribeChan:      make(chan chan LogMsg),
        unsubscribeChan:    make(chan chan LogMsg),
    }

    go self.run()

    return self, nil
}

// Create and return new Logger whose output is distributed by Logs to all followers
func (self Logs) Logger(prefix string) *log.Logger {
    logWriter := logWriter{writeChan: self.writeChan}

    // also copy to stderr
    writer := io.MultiWriter(logWriter, os.Stderr)

    return log.New(writer, prefix, 0)
}

// Manage Logs state
// This will never exit
func (self Logs) run() {
    // internal state
    var history []LogMsg
    subscribers := make(map[chan LogMsg]bool)

    for {
        select {
        case msg := <-self.writeChan:
            history = append(history, msg)

            for writeChan, _ := range subscribers {
                writeChan <- msg
            }

        case writeChan := <-self.subscribeChan:
            subscribers[writeChan] = true

            // send history
            for _, msg := range history {
                writeChan <- msg
            }

        case writeChan := <-self.unsubscribeChan:
            delete(subscribers, writeChan)

            // close the chan, such that logFollower.run() will exit
            close(writeChan)
        }
    }
}
