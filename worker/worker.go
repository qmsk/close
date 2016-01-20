package worker

import (
    "close/config"
    "close/stats"
)

type Worker interface {
    Config() config.Config

    StatsWriter(statsWriter *stats.Writer) error
    ConfigSub(configSub *config.Sub) error

    Run() error
    Stop()
}
