package worker

import (
    "github.com/qmsk/close/config"
    "github.com/qmsk/close/stats"
)

type WorkerConfig interface {
    Worker() (Worker, error)
}

type Worker interface {
    Run() error
}

type StatsWorker interface {
    StatsWriter(*stats.Writer) error
}

type ConfigWorker interface {
    ConfigSub(*config.Sub) error
}

type StopWorker interface {
    Stop()
}
