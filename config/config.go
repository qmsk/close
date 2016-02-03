package config

import (
    "fmt"
    "encoding/json"
)

// Type-checked JSON-serializeable config object
type Config interface{}

// Generic JSON config object
type ConfigMap map[string]interface{}

type ConfigPush struct {
    // this needs to be a pointer for .MarshalJSON()
    Config      *json.RawMessage    `json:"config,omitempty"`

    retChan     chan ConfigReturn
}

type ConfigReturn struct {
    ID          ID              `json:"id"`
    Error       error           `json:"error,omitempty"`
    Config      Config          `json:"config,omitempty"`
}

func (self ConfigPush) Unmarshal(config interface{}) error {
    if err := json.Unmarshal(*self.Config, config); err != nil {
        return fmt.Errorf("unmarshal %v: %v", self.Config, err)
    } else {
        return nil
    }
}

// Send this ConfigPush over the given chan, read the result and return it.
//
// If given a nil chan, immediately errors out
func (self ConfigPush) apply(configChan chan ConfigPush) (ConfigReturn, error) {
    if configChan == nil {
        ret := ConfigReturn{Error: fmt.Errorf("No config push support")}
        return ret, ret.Error
    }

    self.retChan = make(chan ConfigReturn)

    configChan <- self

    ret, valid := <-self.retChan

    if !valid {
        return ret, fmt.Errorf("No return")
    }

    return ret, ret.Error
}

func (self ConfigPush) ApplyFunc(applyFunc func (ConfigPush) (Config, error)) {
    defer close(self.retChan)

    if config, err := applyFunc(self); err != nil {
        self.retChan <- ConfigReturn{Error: err}
    } else {
        self.retChan <- ConfigReturn{Config: config}
    }
}

func (self ConfigPush) Reject(err error) {
    self.retChan <- ConfigReturn{Error: err}
    close(self.retChan)
}
