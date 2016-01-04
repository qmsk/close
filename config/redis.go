package config

import (
    "fmt"
    "path"
    "gopkg.in/redis.v3"
    "time"
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
    self.prefix = path.Clean(options.Prefix)

    self.redisClient = redis.NewClient(&options.Redis)

    if _, err := self.redisClient.Ping().Result(); err != nil {
        return err
    }

    return nil
}

func (self *Redis) path(parts...string) string {
    return path.Join(append([]string{self.prefix}, parts...)...)
}

func (self *Redis) registerModule(module string) error {
    if err := self.redisClient.SAdd(self.path(), module).Err(); err != nil {
        return err
    }

    return nil
}

// Return a new Sub for the given name
func (self *Redis) Sub(options SubOptions) (*Sub, error) {
    sub := &Sub{redis: self}

    if err := sub.init(options); err != nil {
        return nil, err
    }

    return sub, nil
}

// List all modules
func (self *Redis) ListModules() ([]string, error) {
    return self.redisClient.SMembers(self.path()).Result()
}

// List all Subs, for given module
func (self *Redis) List(module string) (subs []*Sub, err error) {
    start := time.Now().Add(-SUB_TTL)

    members, err := self.redisClient.ZRangeByScore(self.path(module, ""), redis.ZRangeByScore{Min: fmt.Sprintf("%v", start.Unix()), Max: "+inf"}).Result()
    if err != nil {
        return nil, err
    }

    for _, subPath := range members {
        subOptions := SubOptions{Module: module, ID: path.Base(subPath)}

        sub := &Sub{redis: self}
        sub.init(subOptions)

        subs = append(subs, sub)
    }

    return
}
