package logs

import (
    "fmt"
    "math/rand"
    "sync"
    "testing"
    "time"
)

// subscribe COUNT followers every 0..DELAY seconds
const FOLLOWER_COUNT = 100
const FOLLOWER_DELAY = 1.0 * float32(time.Second)

// ...which will each run for 0..RUNTIME seconds..
const FOLLOWER_RUNTIME = 10.0 * float32(time.Second)

// run COUNT goroutines to write messages...
const WRITER_COUNT = 10

// ...which will each write COUNT messages at an interval of DELAY seconds
const MESSAGE_DELAY = 0.1 * float32(time.Second)
const MESSAGE_COUNT = 1000

func TestLogsFollowers (t *testing.T) {
    var waitGroup sync.WaitGroup

    logs, err := New()
    if err != nil {
        t.Fatal(err)
    }

    // add followers
    waitGroup.Add(1)
    go func() {
        defer waitGroup.Done()

        for count := 0; count <= FOLLOWER_COUNT; count++ {
            time.Sleep(time.Duration(rand.Float32() * FOLLOWER_DELAY))

            // create in this goroutine
            lf := newFollower(logs)
            t.Logf("follower %p\n", lf)

            // mock logFollower.reader() to unsub
            waitGroup.Add(1)
            go func(lf logFollower) {
                defer waitGroup.Done()

                time.Sleep(time.Duration(rand.Float32() * FOLLOWER_RUNTIME))

                t.Logf("unsubscribe %p\n", lf)
                lf.close(nil)
            }(lf)

            // mock logFollower.writer() to consume and count messages
            // sleeps randomly to mock a blocked webSocket writer?
            waitGroup.Add(1)
            go func(lf logFollower) {
                defer waitGroup.Done()

                var count uint
                var dropped uint
                var sleep float64

                for msg := range lf.writeChan {
                    count++
                    dropped += msg.Dropped

                    // random delay
                    _sleep := rand.ExpFloat64() / 1.0

                    sleep += _sleep
                    time.Sleep(time.Duration(_sleep * float64(time.Second)))
                }

                t.Logf("stats %p: %d messages + %d dropped @ %.2f sleep\n", lf, count, dropped, sleep)
            }(lf)
        }
    }()

    // write messages
    writeFunc := func(id uint) {
        defer waitGroup.Done()

        for count := 0; count <= MESSAGE_COUNT; count++ {
            time.Sleep(time.Duration(rand.Float32() * float32(MESSAGE_DELAY)))

            logMsg := LogMsg{Line: fmt.Sprintf("Message %d:%d", id, count)}

            t.Logf("Write %v\n", logMsg)
            logs.writeChan <- logMsg
        }
    }

    for writerID := uint(1); writerID < WRITER_COUNT; writerID++ {
        waitGroup.Add(1)
        go writeFunc(writerID)
    }

    t.Logf("Running...\n")
    waitGroup.Wait()
}

