package docker

import (
	"github.com/qmsk/close/docker"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"github.com/qmsk/close/shell"
)

type InfoConfig struct{
}

type InfoCmd struct {
	DockerCmd
	config    InfoConfig
}

func (config InfoConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	infoCmd := &InfoCmd{

		DockerCmd {
			url:    options.Url(),
			user:   options.User(),
		},

		config,
	}
	return infoCmd, nil
}

func (cmd InfoCmd) Url() string {
	return cmd.url
}

func (cmd InfoCmd) User() shell.User {
	return cmd.user
}

func (cmd InfoCmd) SubCmd() string {
	return ""
}

func (cmd InfoCmd) Path() string {
	return "/api/docker"
}

func (cmd InfoCmd) Execute() error {
	return shell.MakeHttpRequest(cmd)
}

func (cmd InfoCmd) ParseJSON(body io.ReadCloser) error {
	var info docker.Info

	if err := json.NewDecoder(body).Decode(&info); err != nil {
		return fmt.Errorf("Error decoding Docker info: %v", err)
	} else {
		log.Printf("%+v", info)
		return nil
	}
}
