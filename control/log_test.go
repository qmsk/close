package control

import (
    "fmt"
    "math/rand"
    "sync"
    "testing"
    "time"
)

const FOLLOWER_COUNT = 100
const MESSAGE_COUNT = 100

func TestLogsFollowers (t *testing.T) {
    var waitGroup sync.WaitGroup

    logs, err := NewLogs()
    if err != nil {
        t.Fatal(err)
    }

    // add followers
    waitGroup.Add(1)
    go func() {
        defer waitGroup.Done()
        for count := 0; count <= FOLLOWER_COUNT; count++ {
            time.Sleep(time.Duration(rand.Float32() * float32(time.Second)))

            // create in this goroutine
            lf := logFollower{
                logs:       logs,
                writeChan:  nil, // just drops
            }

            t.Logf("subscribe %v\n", lf)
            logs.subscribe(lf)

            // unsub later from different goroutine
            waitGroup.Add(1)
            go func(lf logFollower) {
                defer waitGroup.Done()

                time.Sleep(time.Duration(rand.Float32() * 10.0 * float32(time.Second)))

                t.Logf("unsubscribe %v\n", lf)
                logs.unsubscribe(lf)
            }(lf)
        }
    }()

    // write messages
    waitGroup.Add(1)
    go func() {
        defer waitGroup.Done()
        for count := 0; count <= MESSAGE_COUNT; count++ {
            time.Sleep(time.Duration(rand.Float32() * float32(time.Second)))

            logMsg := LogMsg{fmt.Sprintf("Message %d", count)}

            t.Logf("Write %v\n", logMsg)
            logs.write(logMsg)
        }
    }()

    t.Logf("Running...\n")
    waitGroup.Wait()
}

