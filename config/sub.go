package config

import (
    "fmt"
    "encoding/json"
    "log"
    "os"
    "gopkg.in/redis.v3"
    "regexp"
    "time"
)

const SUB_TTL = 10 * time.Second

var subRegexp = regexp.MustCompile(`^(\w+):(\w+)$`)

type SubOptions struct {
    Module  string  `json:"module"`
    ID      string  `json:"id"`
}

func (self SubOptions) String() string {
    return fmt.Sprintf("%s:%s", self.Module, self.ID)
}

// Parse the SubOptions.String() format into a SubOptions
func ParseSub(str string) (self SubOptions, err error) {
    if match := subRegexp.FindStringSubmatch(str); match == nil {
        return self, fmt.Errorf("Invalid sub name: %s", str)
    } else {
        self.Module = match[1]
        self.ID = match[2]

        return
    }
}

type Sub struct {
    redis   *Redis
    options SubOptions
    path    string
    log     *log.Logger

    expire      time.Time
    stopChan    chan bool
}

func (self *Sub) init(options SubOptions) error {
    self.options = options
    self.path = self.redis.path(options.Module, options.ID)
    self.log = log.New(os.Stderr, fmt.Sprintf("config.Sub %v: ", self.path), 0)

    return nil
}

func (self *Sub) String() string {
    return self.options.String()
}
func (self *Sub) Options() SubOptions {
    return self.options
}

// Check if this sub still exists, and return remaining TTL
func (self *Sub) Check() (time.Duration, error) {
    return self.redis.redisClient.PTTL(self.path).Result()
}

// get as generic map[string]interface{}
// includes additional _fields for SubOptions
func (self *Sub) Get() (ConfigMap, error) {
    config := ConfigMap{}

    if err := self.get(&config); err != nil {
        return nil, err
    } else {
        return config, nil
    }
}

// update config from redis
func (self *Sub) get(config Config) error {
    if jsonBuf, err := self.redis.redisClient.Get(self.path).Bytes(); err != nil {
        return nil
    } else if err := json.Unmarshal(jsonBuf, config); err != nil {
        return err
    } else {
        return nil
    }
}

// update config in redis
func (self *Sub) set(config Config) error {
    if jsonBuf, err := json.Marshal(config); err != nil {
        return err
    } else if res := self.redis.redisClient.Set(self.path, jsonBuf, 0); res.Err() != nil {
        return res.Err()
    } else {
        self.log.Printf("set: %v\n", config)
        return nil
    }
}

// get an existing redis config, or set it
func (self *Sub) sync(config Config) error {
    if exists, err := self.redis.redisClient.Exists(self.path).Result(); err != nil {
        return err
    } else if exists {
        return self.get(config)
    } else {
        return self.set(config)
    }
}

// register config in redis, maintaining both the ZSet and the object expiry keepalive under TTL
func (self *Sub) register() {
    // registration set's path, vs self.path for our object
    path := self.redis.path(self.options.Module, "")

    expire := time.Now().Add(SUB_TTL)
    refreshTimer := time.Tick(SUB_TTL / 2)

    for {
        if res := self.redis.redisClient.ExpireAt(self.path, expire); res.Err() != nil {
            self.log.Printf("refresh ExpireAt %v: %v\n", expire, res.Err())

        } else if res := self.redis.redisClient.ZAdd(path, redis.Z{Score: float64(expire.Unix()), Member: self.path}); res.Err() != nil {
            self.log.Printf("refresh ZAdd %v: %v\n", path, res.Err())
        }

        select {
        case t := <-refreshTimer:
            // update expiry for next iteration
            expire = t.Add(SUB_TTL)

        case <-self.stopChan:
            // unregister
            self.redis.redisClient.ZRem(path, self.path)
            break
        }
    }
}

// Stop refreshing the config in redis, and remove it
func (self *Sub) Stop() error {
    self.stopChan <- true

    if res := self.redis.redisClient.Del(self.path); res.Err() != nil {
        return res.Err()
    }

    return nil
}

func (self *Sub) read(pubsub *redis.PubSub, configChan chan Config, config Config) {
    defer close(configChan)

    for {
        if msg, err := pubsub.ReceiveMessage(); err != nil {
            self.log.Printf("read: %v\n", err)
            break
        } else if err := json.Unmarshal([]byte(msg.Payload), config); err != nil {
            self.log.Printf("read JSON: %v\n", err)
            continue
        } else {
            configChan <- config
        }

        // update stored config
        if err := self.set(config); err != nil {
            self.log.Printf("read -> set: %v\n", err)
            continue
        }
    }
}

// Register ourselves in redis, storing or updating the given Config
// Read updates from redis, storing them into the given Config
// Each updated Config is delivered on the given chan
// Once the config has been sent, it is updated into redis
func (self *Sub) Start(config Config) (chan Config, error) {
    // sync object
    if err := self.sync(config); err != nil {
        return nil, err
    }

    // register top-level module
    if err := self.redis.registerModule(self.options.Module); err != nil {
        return nil, err
    }

    // register the object
    self.stopChan = make(chan bool)

    go self.register()

    // subscribe for updates
    pubsub, err := self.redis.redisClient.Subscribe(self.path)
    if err != nil {
        return nil, err
    }

    configChan := make(chan Config)

    go self.read(pubsub, configChan, config)

    // running
    return configChan, nil
}

// update (partial) params for sub
// return an error if there is no active sub
func (self *Sub) Push(config Config) error {
    if jsonBuf, err := json.Marshal(config); err != nil {
        return err
    } else if count, err := self.redis.redisClient.Publish(self.path, string(jsonBuf)).Result(); err != nil {
        return err
    } else if count == 0 {
        // redis did not have anything SUBSCRIBE'd to this path
        return fmt.Errorf("Publish to empty Sub: %v", self.path)
    } else {
        self.log.Printf("Push: %v\n", config)
        return nil
    }
}
