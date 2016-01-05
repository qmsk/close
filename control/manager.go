package control

import (
    "close/config"
    "fmt"
    "close/stats"
)

type Options struct {
    StatsReader stats.ReaderConfig
    Config      config.Options
}

type Manager struct {
    configRedis *config.Redis
    statsReader *stats.Reader
}

func New(options Options) (*Manager, error) {
    self := &Manager{}

    if err := self.init(options); err != nil {
        return nil, err
    }

    return self, nil
}

func (self *Manager) init(options Options) error {
    if options.Config.Redis.Addr == "" {
        return fmt.Errorf("missing --config-redis-addr")
    } else if configRedis, err := config.NewRedis(options.Config); err != nil {
        return fmt.Errorf("config.NewRedis %v: %v", options.Config, err)
    } else {
        self.configRedis = configRedis
    }

    if statsReader, err := stats.NewReader(options.StatsReader); err != nil {
        return fmt.Errorf("stats.NewReader %v: %v", options.StatsReader, err)
    } else {
        self.statsReader = statsReader
    }

    return nil
}
