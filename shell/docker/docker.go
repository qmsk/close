package docker

import (
	"log"
	"github.com/qmsk/close/shell"
)

// TODO make similar Register functionality as for commands
type DockerConfig struct {
	subCommands  map[string]shell.CommandConfig
}

type DockerCmd struct {
	url    string
	user   shell.User
	subCmd string

	config DockerConfig
}

// Docker is a CompositionalCommand, it has subcommands
func (cfg *DockerConfig) Register(subcmd string, config shell.CommandConfig) {
	if cfg.subCommands == nil {
		cfg.subCommands = make(map[string]shell.CommandConfig)
	}
	cfg.subCommands[subcmd] = config
}

func (cfg DockerConfig) SubCommands() map[string]shell.CommandConfig {
	return cfg.subCommands
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

