package shell

import (
    "github.com/BurntSushi/toml"
)

type User struct {
    Id       string   `short:"l" long:"login" description:"login username" default:"admin"`
    Password string   `short:"p" long:"password" description:"login password"`
}

type Config struct {
	URL  string       `short:"u" long:"url" description:"controller URL"`
	User User
}

func NewConfig(filePath string) (*Config, error) {
	cfg := &Config{}
    if _, err := toml.DecodeFile(filePath, cfg); err != nil {
        return nil, err
    }
	return cfg, nil
}
