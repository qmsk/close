package control

import (
    "close/config"
    "fmt"
    "log"
)

func (self *Manager) ConfigList() (map[string]config.Config, error) {
    list := make(map[string]config.Config)

    modules, err := self.configRedis.ListModules()
    if err != nil {
        return list, fmt.Errorf("config.Redis %v: ListModules: %v", self.configRedis, err)
    }

    for _, module := range modules {
        subs, err := self.configRedis.List(module)
        if err != nil {
            return list, fmt.Errorf("config.Redis %v: List %v: %v", self.configRedis, module, err)
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

func (self *Manager) ConfigGet(sub string) (config.Config, error) {
    if subOptions, err := config.ParseSub(sub); err != nil {
        return nil, fmt.Errorf("config.ParseSub %v: %v", sub, err)
    } else if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return nil, fmt.Errorf("config.Redis %v: Sub %v: %v", self.configRedis, subOptions, err)
    } else if subConfig, err := configSub.Get(); err != nil {
        return nil, fmt.Errorf("config.Sub %v: Get: %v", configSub, err)
    } else {
        log.Printf("config.Sub %v: Get: %v\n", configSub, subConfig)

        return subConfig, nil
    }
}

func (self *Manager) ConfigPush(sub string, pushConfig config.Config) error {
    if subOptions, err := config.ParseSub(sub); err != nil {
        return fmt.Errorf("config.ParseSub %v: %v", sub, err)
    } else if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return fmt.Errorf("config.Redis %v: Sub %v: %v", self.configRedis, subOptions, err)
    } else if err := configSub.Push(pushConfig); err != nil {
        return fmt.Errorf("config.Sub %v: Push %v: %v", configSub, pushConfig, err)
    } else {
        log.Printf("config.Sub %v: Push %v\n", configSub, pushConfig)

        return nil
    }
}
