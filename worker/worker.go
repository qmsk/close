package worker

import (
    "close/config"
    "close/stats"
)

type WorkerConfig interface {
    Worker() (Worker, error)
}

type Worker interface {
    StatsWriter(*stats.Writer) error
    ConfigSub(*config.Sub) error

    Run() error
    Stop()
}
