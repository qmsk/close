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

type ClientsConfig struct {
}

type ClientsCmd struct {
	DockerCmd
	config ClientsConfig
}

func (config ClientsConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	clientsCmd := &ClientsCmd{
		DockerCmd {
			url:    options.Url(),
			user:   options.User(),
		},

		config,
	}
	return clientsCmd, nil
}

func (cmd ClientsCmd) Url() string {
	return cmd.url
}

func (cmd ClientsCmd) User() shell.User {
	return cmd.user
}

func (cmd ClientsCmd) Path() string {
	return "/api/"
}

func (cmd ClientsCmd) Execute() error {
	return shell.MakeHttpRequest(cmd)
}

func (cmd ClientsCmd) ParseJSON(body io.ReadCloser) error {
	var res control.APIGet
	if err := json.NewDecoder(body).Decode(&res); err != nil {
		return fmt.Errorf("Error decoding controller state: %v", err)
	} else {
		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("Clients", res.Clients)
		outputter.Printf(output)
		return nil
	}
}
