package main

import (
    "github.com/qmsk/close/worker"
    "github.com/qmsk/close/workers"
)

func main() {
    workers.Options.Parse()

    worker.Main(workers.Options)
}
