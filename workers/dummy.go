// +build workertest

package workers

import (
    "bytes"
    "fmt"
    "log"
    "os"
    "time"

    "crypto/sha256"
    "encoding/hex"
    "encoding/binary"
    "math/rand"

    "close/worker"
    "close/stats"
    "close/config"
)

type ID [8]byte

func (self ID) String() string {
    return hex.EncodeToString(self[:])
}

// Parse an ID from hexadecimal
func parseID(s string) (id ID, err error) {
    if buf, err := hex.DecodeString(s); err != nil {
        return id, err
    } else {
        copy(id[:], buf[0:8])
        return id, nil
    }
}

// Generate an ID
func genID() (ID, error) {
    hash := sha256.New()
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    binary.Write(hash, binary.BigEndian, r.Float64())

    var hashSum []byte

    hashSum = hash.Sum(hashSum)

    // truncated hash-sum into 64-bit id 
    var id ID

    if err := binary.Read(bytes.NewReader(hashSum), binary.BigEndian, &id); err != nil {
        return id, err
    }

    return id, nil
}

type DummyConfig struct {
    CmdLineParam  string  `long:"cmd-param" value-name:"0-10" description:"Some integer command line parameter"`

    ID            string  `json:"id" long:"id"`
    ConfigParam   uint    `json:"cfg-param" long:"cfg-param"`
}

func (self DummyConfig) Worker() (worker.Worker, error) {
    return NewDummy(self)
}

type DummyWorker struct {
    config       DummyConfig
    log          *log.Logger

    id           ID
    cmdParam     uint
    configParam  uint

    configChan   chan config.ConfigPush
}

func NewDummy(config DummyConfig) (*DummyWorker, error) {
    dummy := &DummyWorker{
        log:    log.New(os.Stderr, "dummyWorker: ", 0),
    }

    if err := dummy.apply(config); err != nil {
        return nil, err
    }

    return dummy, nil
}

// Worker interface implementation
func (d *DummyWorker) StatsWriter(statsWriter *stats.Writer) error {
    return nil
}

func (d *DummyWorker) ConfigSub(configSub *config.Sub) error {
    // initial config
    if configChan, err := configSub.Start(d.config); err != nil {
        return err
    } else {
        d.configChan = configChan
    }

    return nil
}

func (d *DummyWorker) Run() error {
    for {
        select {
        case configPush, open := <-d.configChan:
            if !open {
                // killed
                return nil
            } else {
                configPush.ApplyFunc(d.configPush)
            }
        }
    }
}

func (d *DummyWorker) Stop() {
    d.log.Printf("stopping...\n")

    close(d.configChan)
}

// Private
func (d *DummyWorker) apply(config DummyConfig) error {
    if config.ID == "" {
        if id, err := genID(); err != nil {
            return fmt.Errorf("genID: %v", err)
        } else {
            config.ID = id.String()
            d.id = id
        }
    } else {
        if id, err := parseID(config.ID); err != nil {
            return fmt.Errorf("parseID %v: %v", config.ID, err)
        } else {
            d.id = id
        }
    }

    d.config = config

    return nil
}

func (d *DummyWorker) configPush(configPush config.ConfigPush) (config.Config, error) {
    config := d.config // copy

    if err := configPush.Unmarshal(&config); err != nil {
        return nil, err
    }

    d.log.Printf("configPush: %#v\n", config)

    if config.ID != d.config.ID {
        return nil, fmt.Errorf("Cannot change ID")
    }

    if config.ConfigParam != d.config.ConfigParam {
        d.log.Printf("config ConfigParam=%d\n", config.ConfigParam)

        d.config.ConfigParam = config.ConfigParam
    }
    
    return d.config, nil
}
