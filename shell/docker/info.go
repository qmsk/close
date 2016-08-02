package docker

import (
	"github.com/qmsk/close/docker"
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
			subCmd: options.SubCmd(),
		},

		config,
	}
	return infoCmd, nil
}

func (cmd InfoCmd) Execute() (err error) {
	log.Printf("command docker info, Execute: url %v, %#v", cmd.url, cmd.config)

	if resp, err := shell.DoRequest(cmd.url, cmd.user , "/api/docker"); err != nil {
		log.Printf("shell.DoRequest %v: %v", cmd.url, err)
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

