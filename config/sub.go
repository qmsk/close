package config

import (
    "gopkg.in/redis.v3"
    "time"
)

const TTL = 10 * time.Second

type SubOptions struct {
    Module  string
    ID      string
}

type Sub struct {
    redis   *Redis
    options SubOptions
    path    string

    stopChan    chan bool
}

func (self *Sub) init(options SubOptions) {
    self.options = options
    self.path = self.redis.path(options.Module, options.ID)
}

// register in redis
func (self *Sub) start(config Config) error {
    self.stopChan = make(chan bool)

    if err := self.set(config); err != nil {
        return err
    }

    go self.keepalive()

    return nil
}

// update config in redis
func (self *Sub) set(config Config) error {
    // XXX: HMSet() is dumb and needs the first pair as separate arguments
    var firstField, firstValue string
    var pairs []string

    for key, value := range config {
        if firstField == "" {
            firstField = key
            firstValue = value
        } else {
            pairs = append(pairs, key, value)
        }
    }

    if res := self.redis.redisClient.HMSet(self.path, firstField, firstValue, pairs...); res.Err() != nil {
        return res.Err()
    }

    return nil
}

func (self *Sub) keepalive() {
    path := self.redis.path(self.options.Module, "")
    expire := time.Now().Add(TTL)
    refreshTimer := time.Tick(TTL / 2)

    for {
        // XXX: errors
        self.redis.redisClient.ZAdd(path, redis.Z{Score: float64(expire.Unix()), Member: self.path})
        self.redis.redisClient.ExpireAt(self.path, expire)

        select {
        case t := <-refreshTimer:
            expire = t.Add(TTL)

        case <-self.stopChan:
            self.redis.redisClient.ZRem(path, self.path)
            break
        }
    }
}

func (self *Sub) read(pubsub *redis.PubSub, readChan chan map[string]string) {
    defer close(readChan)

    for {
        if _, err := pubsub.ReceiveMessage(); err != nil {
            break
        }

        if res := self.redis.redisClient.HGetAllMap(self.path); res.Err() != nil {
            break
        } else {
            readChan <- res.Val()
        }
    }
}

func (self *Sub) Read() (chan map[string]string, error) {
    pubsub, err := self.redis.redisClient.Subscribe(self.path)
    if err != nil {
        return nil, err
    }

    readChan := make(chan map[string]string)

    go self.read(pubsub, readChan)

    return readChan, nil
}

func (self *Sub) Stop() error {
    self.stopChan <- true

    self.redis.redisClient.Del(self.path) // XXX

    return nil
}
