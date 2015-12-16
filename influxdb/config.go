package influxdb

const (
    DEFAULT_SERVER = "http://localhost:8086"
    DEFAULT_DB = "close"
)

type Config struct {
    Server   string
    Database string
}

func (c *Config) WithDefaults() *Config {
    d := *c
    if d.Server == "" {
        d.Server = DEFAULT_SERVER
    }
    if d.Database == "" {
        d.Database = DEFAULT_DB
    }
    return &d
}
