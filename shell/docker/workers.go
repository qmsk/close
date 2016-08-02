package docker

import (
	"log"
	"github.com/qmsk/close/shell"
)

type WorkersConfig struct {
}

type WorkersCmd struct {
	config WorkersConfig
}

func (config WorkersConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	workersCmd := &WorkersCmd{
		config: config,
	}
	return workersCmd, nil
}

func (cmd WorkersCmd) Execute() error {
	log.Printf("command docker workers, Execute: ")
	return nil
}

