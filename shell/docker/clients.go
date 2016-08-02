package docker

import (
	"log"
	"github.com/qmsk/close/shell"
)

type ClientsConfig struct {
}

type ClientsCmd struct {
	config ClientsConfig
}

func (config ClientsConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	clientsCmd := &ClientsCmd{
		config: config,
	}
	return clientsCmd, nil
}

func (cmd ClientsCmd) Execute() error {
	log.Printf("command docker worker, Execute")
	return nil
}
