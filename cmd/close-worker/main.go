package main

import (
    "github.com/qmsk/close/worker"
)

var Options worker.Options

func main() {
    Options.Parse()

    worker.Main(Options)
}
