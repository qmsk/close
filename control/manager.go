package control

import (
    "close/config"
)

type Manager struct {
    configRedis *config.Redis
}

func New(configRedis *config.Redis) *Manager {
    return &Manager{
        configRedis:    configRedis,
    }
}
