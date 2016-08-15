package docker

import (
	"github.com/qmsk/close/control"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"os"
	"github.com/qmsk/close/shell"
	"github.com/qmsk/close/util"
)

type WorkersConfig struct {
}

type WorkersCmd struct {
	DockerCmd
	config WorkersConfig
}

func (config WorkersConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	workersCmd := &WorkersCmd{

		DockerCmd {
			url:    options.Url(),
			user:   options.User(),
		},

		config,
	}
	return workersCmd, nil
}

func (cmd WorkersCmd) Url() string {
	return cmd.url
}

func (cmd WorkersCmd) User() shell.User {
	return cmd.user
}

func (cmd WorkersCmd) Path() string {
	return "/api/"
}

func (cmd WorkersCmd) Execute() error {
	return shell.MakeHttpRequest(cmd)
}

func (cmd WorkersCmd) ParseJSON(body io.ReadCloser) error {
	var res control.APIGet
	if err := json.NewDecoder(body).Decode(&res); err != nil {
		return fmt.Errorf("Error decoding controller state: %v", err)
	} else {
		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("Workers", res.Workers)
		outputter.Printf(output)
		return nil
	}
}
