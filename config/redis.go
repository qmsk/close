package config

import (
    "fmt"
    "path"
    "gopkg.in/redis.v3"
    "time"
)

type Options struct {
    RedisURL        RedisURL        `long:"redis-url" value-name:"redis://[:PASSWORD@]HOST[:PORT][/PREFIX]" env:"REDIS_URL"`
}

func (self Options) Empty() bool {
    return self.RedisURL.Empty()
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

    if redisClient, err := options.RedisURL.RedisClient(); err != nil {
        return err
    } else {
        self.redisClient = redisClient
    }

    self.prefix = options.RedisURL.Prefix()

    return nil
}

func (self *Redis) String() string {
    return self.options.RedisURL.String()
}

func (self *Redis) path(parts...string) string {
    return path.Join(append([]string{self.prefix}, parts...)...)
}

func (self *Redis) registerType(subType string) error {
    if err := self.redisClient.SAdd(self.path(), subType).Err(); err != nil {
        return err
    }

    return nil
}

// Get an existing sub
// may not necessarily be active yet!
func (self *Redis) GetSub(id ID) (*Sub, error) {
    if err := id.Check(); err != nil {
        return nil, err
    } else {
        return newSub(self, id)
    }
}

// Return a new Sub for the given name
func (self *Redis) NewSub(subType string, instance string) (*Sub, error) {
    if id, err := ParseID(subType, instance); err != nil {
        return nil, err
    } else if sub, err := newSub(self, id); err != nil {
        return nil, err
    } else {
        return sub, nil
    }
}

// List all types
func (self *Redis) ListTypes() ([]string, error) {
    return self.redisClient.SMembers(self.path()).Result()
}

// List all Subs, for given type
func (self *Redis) List(subType string) (subs []*Sub, err error) {
    start := time.Now().Add(-SUB_TTL)

    members, err := self.redisClient.ZRangeByScore(self.path(subType, ""), redis.ZRangeByScore{Min: fmt.Sprintf("%v", start.Unix()), Max: "+inf"}).Result()
    if err != nil {
        return nil, err
    }

    for _, subPath := range members {
        instance := path.Base(subPath)

        if id, err := ParseID(subType, instance); err != nil {
            return nil, err
        } else if sub, err := newSub(self, id); err != nil {
            return nil, err
        } else {
            subs = append(subs, sub)
        }
    }

    return
}
