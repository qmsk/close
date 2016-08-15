package docker

import (
	"github.com/qmsk/close/docker"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"os"
	"github.com/qmsk/close/shell"
	"github.com/qmsk/close/util"
)

type ListConfig struct {
}

type ListCmd struct {
	DockerCmd
	config ListConfig
}

func (config ListConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	listCmd := &ListCmd{

		DockerCmd {
			url:    options.Url(),
			user:   options.User(),
		},

		config,
	}
	return listCmd, nil
}

func (cmd ListCmd) Url() string {
	return cmd.url
}

func (cmd ListCmd) User() shell.User {
	return cmd.user
}

func (cmd ListCmd) Path() string {
	return "/api/docker/"
}

func (cmd ListCmd) Execute() error {
	return shell.MakeHttpRequest(cmd)
}

func (cmd ListCmd) ParseJSON(body io.ReadCloser) error {
	var containers []docker.ContainerStatus
	
	if err := json.NewDecoder(body).Decode(&containers); err != nil {
		return fmt.Errorf("Error decoding the list of docker containers: %v", err)
	} else {
		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("", containers)
		outputter.Printf(output)
		return nil
	}
}
