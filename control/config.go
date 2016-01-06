package control

import (
    "close/config"
    "fmt"
    "log"
)

type ConfigItem struct {
    config.SubOptions

    TTL     float64 `json:"ttl"` // seconds
}

func (self *Manager) ConfigList(filter config.SubOptions) (configs []ConfigItem, err error) {
    var modules []string

    if filter.Module == "" {
        if listModules, err := self.configRedis.ListModules(); err != nil {
            return nil, fmt.Errorf("config.Redis %v: ListModules: %v", self.configRedis, err)
        } else {
            modules = listModules
        }
    } else {
        modules = []string{filter.Module}
    }

    for _, module := range modules {
        subs, err := self.configRedis.List(module)
        if err != nil {
            return nil, fmt.Errorf("config.Redis %v: List %v: %v", self.configRedis, module, err)
        }

        for _, configSub := range subs {
            configItem := ConfigItem{SubOptions:configSub.Options()}

            if ttl, err := configSub.Check(); err != nil {
                log.Printf("Manager.List Sub.Get %v: %v\n", configSub, err)
                continue
            } else {
                configItem.TTL = ttl.Seconds()
            }

            configs = append(configs, configItem)
        }
    }

    return configs, nil
}

func (self *Manager) ConfigGet(subOptions config.SubOptions) (config.Config, error) {
    if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return nil, fmt.Errorf("config.Redis %v: Sub %v: %v", self.configRedis, subOptions, err)
    } else if subConfig, err := configSub.Get(); err != nil {
        return nil, fmt.Errorf("config.Sub %v: Get: %v", configSub, err)
    } else {
        log.Printf("config.Sub %v: Get: %v\n", configSub, subConfig)

        return subConfig, nil
    }
}

func (self *Manager) ConfigPush(subOptions config.SubOptions, pushConfig config.Config) error {
    if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return fmt.Errorf("config.Redis %v: Sub %v: %v", self.configRedis, subOptions, err)
    } else if err := configSub.Push(pushConfig); err != nil {
        return fmt.Errorf("config.Sub %v: Push %v: %v", configSub, pushConfig, err)
    } else {
        log.Printf("config.Sub %v: Push %v\n", configSub, pushConfig)

        return nil
    }
}
