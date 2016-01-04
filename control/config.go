package control

import (
    "close/config"
    "log"
)

func (self *Manager) ConfigList() (map[string]config.Config, error) {
    list := make(map[string]config.Config)

    modules, err := self.configRedis.ListModules()
    if err != nil {
        return list, err
    }

    for _, module := range modules {
        subs, err := self.configRedis.List(module)
        if err != nil {
            return list, err
        }

        for _, configSub := range subs {
            if subConfig, err := configSub.Get(); err != nil {
                log.Printf("Manager.List Sub.Get %v: %v\n", configSub, err)
                continue
            } else {
                list[configSub.String()] = subConfig
            }
        }
    }

    return list, nil
}
