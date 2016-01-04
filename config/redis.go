package config

import (
    "gopkg.in/redis.v3"
    "strings"
)

type Options struct {
    Redis           redis.Options
    Prefix          string
}

type Redis struct {
    options     Options
    prefix      string
    redisClient *redis.Client
}

func NewRedis(options Options) (*Redis, error) {
    self := &Redis{}

    if err := self.init(options); err != nil {
        return nil, err
    } else {
        return self, nil
    }
}

func (self *Redis) init(options Options) error {
    self.prefix = strings.TrimRight(options.Prefix, "/")

    self.redisClient = redis.NewClient(&options.Redis)

    if _, err := self.redisClient.Ping().Result(); err != nil {
        return err
    }

    return nil
}

func (self *Redis) path(parts...string) string {
    return self.prefix + "/" + strings.Join(parts, "/")
}

// Return a new Sub for the given name
func (self *Redis) Sub(options SubOptions) (*Sub, error) {
    sub := &Sub{redis: self}

    if err := sub.init(options); err != nil {
        return nil, err
    }

    return sub, nil
}
