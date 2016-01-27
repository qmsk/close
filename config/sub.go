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

    setChan     chan []byte
    stopChan    chan bool
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
func (self *Sub) set(config Config) error {
    if jsonBuf, err := json.Marshal(config); err != nil {
        return err
    } else {
        self.setChan <- jsonBuf

        return nil
    }
}

// register config in redis, maintaining both the ZSet and the object expiry keepalive under TTL
func (self *Sub) register() {
    // registration set's path
    typePath := self.redis.path(self.id.Type, "")

    // register top-level type
    if err := self.redis.registerType(self.id.Type); err != nil {
        self.log.Printf("failed to register type %v: %v\n", self.id.Type, err)
        // XXX: just continue?!
    }

    expire := time.Now().Add(SUB_TTL)
    refreshTimer := time.Tick(SUB_TTL / 2)

    for {
        select {
        case jsonBuf, open := <-self.setChan:
            if !open {
                return
            }

            setTime := time.Now()

            if res := self.redis.redisClient.Set(self.path, jsonBuf, expire.Sub(setTime)); res.Err() != nil {
                self.log.Printf("Set: %v\n", self.path, res.Err())
            }

        case t := <-refreshTimer:
            // update expiry
            expire = t.Add(SUB_TTL)

            if res := self.redis.redisClient.ExpireAt(self.path, expire); res.Err() != nil {
                self.log.Printf("refresh ExpireAt %v: %v\n", expire, res.Err())

            } else if res := self.redis.redisClient.ZAdd(typePath, redis.Z{Score: float64(expire.Unix()), Member: self.path}); res.Err() != nil {
                self.log.Printf("refresh ZAdd %v: %v\n", typePath, res.Err())
            }
        }
    }

    // unregister
    self.redis.redisClient.ZRem(typePath, self.path)
}

// Stop refreshing the config in redis, and remove it
func (self *Sub) Stop() error {
    close(self.setChan)

    if res := self.redis.redisClient.Del(self.path); res.Err() != nil {
        return res.Err()
    }

    return nil
}

func (self *Sub) subscribe(pubsub *redis.PubSub, configChan chan ConfigPush, config Config) {
    defer close(configChan)

    for {
        if msg, err := pubsub.ReceiveMessage(); err != nil {
            self.log.Printf("read: %v\n", err)
            break
        } else if err := json.Unmarshal([]byte(msg.Payload), config); err != nil {
            self.log.Printf("read JSON: %v\n", err)
            continue
        }

        configPush := ConfigPush{
            ID:     self.id,
            Config: config,

            ackChan:    make(chan Config),
            errChan:    make(chan error),
        }

        // apply and wait
        configChan <- configPush

        select {
        case config := <-configPush.ackChan:
            configPush.Config = config

            if err := self.set(config); err != nil {
                self.log.Printf("subscribe -> set: %v\n", err)
                configPush.Error = err
            }

        case err := <-configPush.errChan:
            configPush.Error = err

            self.log.Printf("subscribe -> err: %v\n", err)
        }

        close(configPush.ackChan)
        close(configPush.errChan)

        // TODO: PUBLISH
    }
}

// Register ourselves in redis, storing or updating the given Config
// Read updates from redis, storing them into the given Config
// Each updated Config is delivered on the given chan
// Once the config has been sent, it is updated into redis
func (self *Sub) Start(config Config) (chan ConfigPush, error) {
    // register into redis
    self.setChan = make(chan []byte)

    go self.register() // XXX: don't deadlock set() on errors...

    // sync config
    if err := self.set(config); err != nil {
        return nil, err
    }

    // subscribe for updates
    pubsub, err := self.redis.redisClient.Subscribe(self.path)
    if err != nil {
        return nil, err
    }

    configChan := make(chan ConfigPush)

    go self.subscribe(pubsub, configChan, config)

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
