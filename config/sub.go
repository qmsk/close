package config

import (
    "bytes"
    "fmt"
    "encoding/json"
    "log"
    "os"
    "gopkg.in/redis.v3"
    "time"
)

const SUB_TTL = 10 * time.Second

type SubOptions struct {
    Options
    Instance    string  `long:"config-instance" env:"CLOSE_INSTANCE"`
}

type Sub struct {
    id      ID
    redis   *Redis
    path    string
    log     *log.Logger

    subscribeChan   chan ConfigPush // in from redis
    configChan      chan ConfigPush // out to caller
}

func newSub(redis *Redis, id ID) (*Sub, error) {
    sub := &Sub{
        id:     id,
        redis:  redis,
        path:   redis.path(id.Type, id.Instance),
        log:    log.New(os.Stderr, fmt.Sprintf("config.Sub %v: ", id), 0),
    }

    return sub, nil
}

func (self *Sub) String() string {
    return self.id.String()
}
func (self *Sub) ID() ID {
    return self.id
}

// Check if this sub still exists, and return remaining TTL
func (self *Sub) Check() (time.Duration, error) {
    if duration, err := self.redis.redisClient.PTTL(self.path).Result(); err != nil {
        return 0, err
    } else if duration < 0 {
        // XXX: go-redis doesn't handle these
        n := int(duration.Seconds() * 1000)

        return 0, fmt.Errorf("PTTL: %d", n)
    } else {
        return duration, nil
    }
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
    jsonBytes, err := self.redis.redisClient.Get(self.path).Bytes()
    if err != nil {
        return err
    }

    // Decode using json.Number
    buf := bytes.NewReader(jsonBytes)
    decoder := json.NewDecoder(buf)
    decoder.UseNumber()

    if err := decoder.Decode(config); err != nil {
        return err
    }

    return nil
}

// update config in redis
// requires register() to be running
func (self *Sub) set(config Config, expire time.Duration) error {
    if jsonBuf, err := json.Marshal(config); err != nil {
        return err
    } else if err := self.redis.redisClient.Set(self.path, jsonBuf, expire).Err(); err != nil {
        return err
    } else {
        return nil
    }
}

func (self *Sub) refresh(zpath string, expire time.Time) error {
    if res := self.redis.redisClient.ExpireAt(self.path, expire); res.Err() != nil {
        return fmt.Errorf("ExpireAt %v %v: %v\n", self.path, expire, res.Err())
    } else if res := self.redis.redisClient.ZAdd(zpath, redis.Z{Score: float64(expire.Unix()), Member: self.path}); res.Err() != nil {
        return fmt.Errorf("ZAdd %v: %v\n", zpath, res.Err())
    } else {
        return nil
    }
}

func (self *Sub) clear(zpath string) {
    if err := self.redis.redisClient.ZRem(zpath, self.path).Err(); err != nil {
        self.log.Printf("clear ZRem %v: %v\n", zpath, err)
    }

    if err := self.redis.redisClient.Del(self.path).Err(); err != nil {
        self.log.Printf("clear Del %v: %v\n", self.path, err)
    }
}
// register config in redis, maintaining both the ZSet and the object expiry keepalive under TTL
func (self *Sub) register(config Config) {
    defer close(self.configChan)
    defer self.log.Printf("stopped")

    // register top-level type
    if err := self.redis.registerType(self.id.Type); err != nil {
        self.log.Printf("failed to register type %v: %v\n", self.id.Type, err)
        return
    }

    // register config
    zpath := self.redis.path(self.id.Type, "")
    expire := time.Now().Add(SUB_TTL)
    refreshTimer := time.Tick(SUB_TTL / 2)

    if err := self.set(config, -time.Since(expire)); err != nil {
        self.log.Printf("failed at intiial set: %v\n", err)
        return
    }
    if err := self.refresh(zpath, expire); err != nil {
        self.log.Printf("failed at initial refresh: %v\n", err)
        return
    }

    defer self.clear(zpath)

    // maintain registration, and handle pushes
    for {
        select {
        case configPush, alive := <-self.subscribeChan:
            if !alive {
                return
            }

            // call into worker mainloop via configChan
            configReturn, err := configPush.apply(self.configChan)

            if err != nil {
                self.log.Printf("push -> err: %v\n", err)
            } else if err := self.set(configReturn.Config, -time.Since(expire)); err != nil {
                self.log.Printf("push -> set: %v\n", err)
                configReturn.Error = err
            } else {
                self.log.Printf("push -> ok\n")
            }

            // TODO: Return PUBLISH

        case t := <-refreshTimer:
            // update expiry
            expire = t.Add(SUB_TTL)

            if err := self.refresh(zpath, expire); err != nil {
                self.log.Printf("refresh: %v\n", err)
                return
            }
        }
    }
}

// Subscribe to ConfigPush's from redis
func (self *Sub) subscribe(subscribeChan chan ConfigPush, pubsub *redis.PubSub) {
    defer close(subscribeChan)

    for {
        configPush := ConfigPush{}

        if msg, err := pubsub.ReceiveMessage(); err != nil {
            self.log.Printf("read: %v\n", err)
            break
        } else if err := json.Unmarshal([]byte(msg.Payload), &configPush); err != nil {
            self.log.Printf("read JSON: %v\n", err)
            continue
        }

        subscribeChan <- configPush
    }
}

// Register ourselves in redis, storing the given Config.
//
// Receive ConfigPush's from redis, and store the updated Config into redis.
func (self *Sub) Start(config Config) (chan ConfigPush, error) {
    // subscribe for updates
    pubsub, err := self.redis.redisClient.Subscribe(self.path)
    if err != nil {
        return nil, err
    }

    self.subscribeChan = make(chan ConfigPush)

    go self.subscribe(self.subscribeChan, pubsub)

    // register and handle updates
    self.configChan = make(chan ConfigPush)

    go self.register(config)


    return self.configChan, nil
}

// update (partial) params for sub
// return an error if there is no active sub
func (self *Sub) Push(config Config) error {
    configPush := ConfigPush{}

    if jsonBuf, err := json.Marshal(config); err != nil {
        return err
    } else {
        configPush.Config = (*json.RawMessage)(&jsonBuf)
    }

    if jsonBuf, err := json.Marshal(configPush); err != nil {
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
