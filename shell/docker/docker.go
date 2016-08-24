package docker

import (
	"github.com/qmsk/close/shell/config"
	"fmt"
)

type DockerConfig struct {
	subCommands map[string]config.CommandConfig
}

type DockerCmd struct {
	url    string
	user   config.User
	subCmd string

	config DockerConfig
}

// Docker is a CompositionalCommand, it has subcommands
func (cfg *DockerConfig) Register(subcmd string, cmdConfig config.CommandConfig) {
	if cfg.subCommands == nil {
		cfg.subCommands = make(map[string]config.CommandConfig)
	}
	cfg.subCommands[subcmd] = cmdConfig
}

func (cfg DockerConfig) SubCommands() map[string]config.CommandConfig {
	return cfg.subCommands
}

func (cfg DockerConfig) Command(options config.CommonOptions) (config.Command, error) {
	if opts, hasSubCmd := options.(config.CompositionalCommonOptions); !hasSubCmd {
		return nil, fmt.Errorf("docker is a compositional command but provided options have no subcommand specified")
	} else {
		dockerCmd := &DockerCmd{
			url:    opts.Url(),
			user:   opts.User(),
			subCmd: opts.SubCmd(),
			config: cfg,
		}
		return dockerCmd, nil
	}
}

func (cfg DockerConfig) SubCommand(cmd DockerCmd) (config.Command, error) {
	return cfg.subCommands[cmd.SubCmd()].Command(cmd)
}

// DockerCmd implements CommonOptions to pass them down to subcommands
func (cmd DockerCmd) Url() string {
	return cmd.url
}

func (cmd DockerCmd) User() config.User {
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
