package config

import (
    "github.com/BurntSushi/toml"
)

type Command interface {
	Execute() error
}

// Options, common for all commands
type CommonOptions interface {
	Url()     string
	User()    User
}

type CompositionalCommonOptions interface {
	CommonOptions
	SubCmd()  string
}

// Shell commands
type CommandConfig interface {
	Command(options CommonOptions) (Command, error)
}

type CompositionalCommandConfig interface {
	Register(subcmd string, config CommandConfig)
	SubCommands() map[string]CommandConfig
}

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
