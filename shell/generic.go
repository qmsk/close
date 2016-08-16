package shell

import (
	"fmt"
	"io"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"github.com/qmsk/close/util"
)

type GenericConfigImpl struct {
//	Cmd  string
	Path    string
	ResType reflect.Type
}

type GenericCommandImpl struct {
	url    string
	user   User
	config GenericConfigImpl
}

func (config GenericConfigImpl) Command(options CommonOptions) (Command, error) {
	genericCommand := &GenericCommandImpl{
		url:    options.Url(),
		user:   options.User(),
		config: config,
	}

	return genericCommand, nil
}

func (cmd GenericCommandImpl) Url() string {
	return cmd.url
}

func (cmd GenericCommandImpl) User() User {
	return cmd.user
}

func (cmd GenericCommandImpl) Path() string {
	return cmd.config.Path
}

func (cmd GenericCommandImpl) Execute() error {
	return MakeHttpRequest(cmd)
}

func (cmd GenericCommandImpl) ParseJSON(body io.ReadCloser) error {
	v := reflect.New(cmd.config.ResType)
	res := v.Interface()

	if err := json.NewDecoder(body).Decode(res); err != nil {
		return fmt.Errorf("Error decoding controller state: %v", err)
	} else {
		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("", res)
		outputter.Printf(output)
		return nil
	}
}