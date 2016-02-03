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
        case "tcp":
            *self = *(*RedisURL)(parseURL)
        default:
            return fmt.Errorf("Unsupported URL: %v", parseURL)
        }

        return nil
    }
}

func (self RedisURL) String() string {
    return (*url.URL)(&self).String()
}

func (self *RedisURL) MarshalFlag() (string, error) {
    return self.String(), nil
}

func (self RedisURL) Empty() bool {
    return self.Host == ""
}

// TODO: ?db=
func (self RedisURL) RedisOptions() (redisOptions redis.Options) {
    redisOptions.Network = self.Scheme

    if _, port, err := net.SplitHostPort(self.Host); err == nil && port != "" {
        redisOptions.Addr = self.Host
    } else {
        redisOptions.Addr = net.JoinHostPort(self.Host, REDIS_PORT)
    }

    if self.User != nil {
        redisOptions.Password, _ = self.User.Password()
    }

    return redisOptions
}

func (self RedisURL) Prefix() string {
    return path.Clean(self.Path)
}
