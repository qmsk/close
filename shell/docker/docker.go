package docker

import (
	"log"
	"github.com/qmsk/close/shell"
)

// TODO make similar Register functionality as for commands
type DockerConfig struct {
	//*shell.Options

	ClientsCfg ClientsConfig `command:"clients"`
	WorkersCfg WorkersConfig `command:"workers"`
	InfoCfg    InfoConfig    `command:"info"`

	subCommands map[string]shell.CommandConfig
}

type DockerCmd struct {
	url    string
	user   shell.User
	subCmd string

	config DockerConfig
}

func (cfg *DockerConfig) Init() {
	if cfg.subCommands == nil {
		cfg.subCommands = make(map[string]shell.CommandConfig)
	}
	cfg.subCommands["clients"] = &cfg.ClientsCfg
	cfg.subCommands["workers"] = &cfg.WorkersCfg
	cfg.subCommands["info"] = &cfg.InfoCfg
}

func NewConfig() *DockerConfig {
	cfg := &DockerConfig{}
	cfg.Init()
	return cfg
}

func (config DockerConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	dockerCmd := &DockerCmd{
		url:    options.Url(),
		user:   options.User(),
		subCmd: options.SubCmd(),
		config: config,
	}
	return dockerCmd, nil
}

func (config DockerConfig) SubCommand(cmd DockerCmd) (shell.Command, error) {
	return config.subCommands[cmd.SubCmd()].Command(cmd)
}

// DockerCmd implements CommonOptions to pass them down to subcommands
func (cmd DockerCmd) Url() string {
	return cmd.url
}

func (cmd DockerCmd) User() shell.User {
	return cmd.user
}

func (cmd DockerCmd) SubCmd() string {
	return cmd.subCmd
}

func (cmd DockerCmd) Execute() error {
	// log.Printf("command Docker, subcommand %v execute, url %v: %#v", cmd.subCmd, cmd.config.URL, cmd.config)

	if subCmd, err := cmd.config.SubCommand(cmd); err != nil {
		log.Printf("command Docker, execute: get subcommand failed: %v", err)
		return err
	} else {
		return subCmd.Execute()
	}

	return nil
}

