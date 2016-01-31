package main

import (
    "close/worker"
    "close/workers"
)

func main() {
    workers.Options.Parse()

    worker.Main(workers.Options)
}
