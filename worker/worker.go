package worker

import (
    "close/config"
    "close/stats"
)

type Worker interface {
    Config() config.Config

    StatsWriter(statsWriter *stats.Writer) error
    ConfigSub(configRedis *config.Redis, options config.SubOptions) error

    Run() error
    Stop()
}
