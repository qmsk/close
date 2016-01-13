package control

import (
    "close/config"
    "fmt"
)

type ConfigItem struct {
    config.SubOptions

    TTL     float64 `json:"ttl"` // seconds
}

func (self *Manager) ConfigList(filter config.SubOptions) (configs []ConfigItem, err error) {
    var types []string

    if filter.Type == "" {
        if listTypes, err := self.configRedis.ListTypes(); err != nil {
            return nil, fmt.Errorf("config.Redis %v: ListTypes: %v", self.configRedis, err)
        } else {
            types = listTypes
        }
    } else {
        types = []string{filter.Type}
    }

    for _, subType := range types {
        subs, err := self.configRedis.List(subType)
        if err != nil {
            return nil, fmt.Errorf("config.Redis %v: List %v: %v", self.configRedis, subType, err)
        }

        for _, configSub := range subs {
            configItem := ConfigItem{SubOptions:configSub.Options()}

            if ttl, err := configSub.Check(); err != nil {
                self.log.Printf("Manager.List Sub.Get %v: %v\n", configSub, err)
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
        return subConfig, nil
    }
}

func (self *Manager) ConfigPush(subOptions config.SubOptions, pushConfig config.Config) error {
    if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return fmt.Errorf("config.Redis %v: Sub %v: %v", self.configRedis, subOptions, err)
    } else if err := configSub.Push(pushConfig); err != nil {
        return fmt.Errorf("config.Sub %v: Push %v: %v", configSub, pushConfig, err)
    } else {
        self.log.Printf("config.Sub %v: Push %v\n", configSub, pushConfig)

        return nil
    }
}
