package config

// Type-checked JSON-serializeable config object
type Config interface{}

// Generic JSON config object
type ConfigMap map[string]interface{}

type ConfigPush struct {
    ID          ID              `json:"id"`
    Config      Config          `json:"config,omitempty"`
    Error       error           `json:"error,omitempty"`

    ackChan     chan Config
    errChan     chan error
}

func (self ConfigPush) SendAck(config Config) {
    self.ackChan <- config
}
func (self ConfigPush) SendError(err error) {
    self.errChan <- err
}

