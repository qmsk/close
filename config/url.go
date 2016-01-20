package config

import (
    "fmt"
    "net"
    "path"
    "gopkg.in/redis.v3"
    "net/url"
)

const REDIS_PORT = "6379"

type RedisURL url.URL

func (self *RedisURL) UnmarshalFlag(value string) error {
    if parseURL, err := url.Parse(value); err != nil {
        return err
    } else {
        switch parseURL.Scheme {
        case "", "redis":
            *self = *(*RedisURL)(parseURL)
        default:
            return fmt.Errorf("Unsupported URL: %v", parseURL)
        }

        return nil
    }
}

func (self *RedisURL) MarshalFlag() (string, error) {
    return (*url.URL)(self).String(), nil
}

func (self RedisURL) Empty() bool {
    return self.Scheme == ""
}

// TODO: ?db=
func (self RedisURL) RedisOptions() (redisOptions redis.Options) {
    redisOptions.Network = "tcp"

    if _, _, err := net.SplitHostPort(self.Host); err != nil {
        redisOptions.Addr = self.Host
    } else {
        redisOptions.Addr = net.JoinHostPort(self.Host, REDIS_PORT)
    }

    if self.User != nil {
        redisOptions.Password, _ = self.User.Password()
    }

    return redisOptions
}

func (self RedisURL) RedisClient() (*redis.Client, error) {
    redisOptions := self.RedisOptions()

    redisClient := redis.NewClient(&redisOptions)

    if _, err := redisClient.Ping().Result(); err != nil {
        return nil, err
    }

    return redisClient, nil
}

func (self RedisURL) Prefix() string {
    return path.Clean(self.Path)
}
