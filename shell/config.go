package shell

import (
	"github.com/qmsk/close/control"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"os"
	"github.com/qmsk/close/util"
)

type DumpConfigCmdConfig struct {
}

type DumpConfigCmd struct {
	url    string
	user   User

	config DumpConfigCmdConfig
}

func (config DumpConfigCmdConfig) Command(opts CommonOptions) (Command, error) {
	dumpConfigCmd := &DumpConfigCmd{
		url:    opts.Url(),
		user:   opts.User(),
		config: config,
	}
	return dumpConfigCmd, nil
}

func (cmd DumpConfigCmd) Url() string {
	return cmd.url
}

func (cmd DumpConfigCmd) User() User {
	return cmd.user
}

func (cmd DumpConfigCmd) Path() string {
	return "/api/"
}

func (cmd DumpConfigCmd) Execute() error {
	return MakeHttpRequest(cmd)
}

func (cmd DumpConfigCmd) ParseJSON(body io.ReadCloser) error {
	var res control.APIGet

	if err := json.NewDecoder(body).Decode(&res); err != nil {
		return fmt.Errorf("Error decoding controller config: %v", err)
	} else {
		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("", res.ConfigText)
		outputter.Printf(output)
		return nil
	}
}
