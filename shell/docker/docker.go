package docker

import (
    "github.com/qmsk/close/docker"
	"encoding/json"
	// "io/ioutil"
	// "net/http"
	"log"
    "github.com/qmsk/close/shell"
)

type ClientsConfig struct{
}

type WorkersConfig struct{
}

type InfoConfig struct{
}

type DockerConfig struct {
	//*shell.Options

	ClientsCfg ClientsConfig `command:"clients"`
	WorkersCfg WorkersConfig `command:"workers"`
	InfoCfg    InfoConfig    `command:"info"`

	subCommands map[string]shell.CommandConfig
}

type DockerCmd struct {
	config    DockerConfig
	subCmd    string
}

type ClientsCmd struct {
	config    ClientsConfig
}

type WorkersCmd struct {
	config    WorkersConfig
}

type InfoCmd struct {
	config    InfoConfig
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

func (config DockerConfig) Command(subCommand string) (shell.Command, error) {
	dockerCmd := &DockerCmd{
		config: config,
		subCmd: subCommand,
	}
	return dockerCmd, nil
}

func (config DockerConfig) SubCommand(subCommand string) (shell.Command, error) {
	return config.subCommands[subCommand].Command("")
}

func (cmd DockerCmd) Execute(options shell.Options) error {
	// log.Printf("command Docker, subcommand %v execute, url %v: %#v", cmd.subCmd, cmd.config.URL, cmd.config)

	if subCmd, err := cmd.config.SubCommand(cmd.subCmd); err != nil {
		log.Printf("command Docker, execute: get subcommand failed: %v", err)
		return err
	} else {
		return subCmd.Execute(options)
	}

	return nil
}

func (config ClientsConfig) Command(subCommand string) (shell.Command, error) {
	clientsCmd := &ClientsCmd{
		config: config,
	}
	return clientsCmd, nil
}

func (config WorkersConfig) Command(subCommand string) (shell.Command, error) {
	workersCmd := &WorkersCmd{
		config: config,
	}
	return workersCmd, nil
}

func (config InfoConfig) Command(subCommand string) (shell.Command, error) {
	infoCmd := &InfoCmd{
		config: config,
	}
	return infoCmd, nil
}

func (cmd InfoCmd) Execute(options shell.Options) (err error) {
	log.Printf("command docker clients, Execute: url %v, %#v", options.URL, cmd.config)

	if resp, err := shell.DoRequest(options, "/api/docker"); err != nil {
		log.Printf("shell.DoRequest %v: %v", options, err)
	} else {
		defer resp.Body.Close()
		log.Printf("Response %v, %v, content length %v\n", resp.Status, resp.Proto, resp.ContentLength)

		var info docker.Info

		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			log.Printf("Error decoding Docker info: %v", err)
		} else {
			log.Printf("%+v", info)
		}
	}

	return
}

func (cmd WorkersCmd) Execute(options shell.Options) error {
	log.Printf("command docker workers, Execute: url %v, %#v", options.URL, cmd.config)
	return nil
}

func (cmd ClientsCmd) Execute(options shell.Options) error {
	log.Printf("command docker worker, Execute")
	return nil
}
