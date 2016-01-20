package worker

import (
    "os"
    "os/signal"
)

// worker.Stop() on SIGINT, SIGKILL
// one time only; revert to default signal handling
func stopping (worker Worker) {
    stopChan := make(chan os.Signal, 1)
    signal.Notify(stopChan, os.Interrupt, os.Kill)

    // only once
    defer signal.Stop(stopChan)

    select {
    case <-stopChan:
        worker.Stop()
    }
}
