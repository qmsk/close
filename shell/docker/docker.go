package docker

import (
	"fmt"
	"github.com/qmsk/close/shell"
)

type DockerConfig struct {
	subCommands map[string]shell.CommandConfig
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
	if opts, hasSubCmd := options.(shell.CompositionalCommonOptions); !hasSubCmd {
		return nil, fmt.Errorf("docker is a compositional command but provided options have no subcommand specified")
	} else {
		dockerCmd := &DockerCmd{
			url:    opts.Url(),
			user:   opts.User(),
			subCmd: opts.SubCmd(),
			config: config,
		}
		return dockerCmd, nil
	}
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
		return fmt.Errorf("DockerCmd.Execute: SubCommand: %v", err)
	} else {
		return subCmd.Execute()
	}

	return nil
}
