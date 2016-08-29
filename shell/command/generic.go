package command

import (
	"github.com/qmsk/close/shell/config"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"github.com/qmsk/close/util"
)

type GenericCommand interface {
	config.CommonOptions
	GenericConfig
	JSONResponseParser
}

// The separation of concepts is: URL and user are coming from
// the shell invoking user, config is coming from the command
// implementation, altogether they make a command
type GenericCommandImpl struct {
	url         string
	user        config.User
	config      GenericConfig
}

func (cmd *GenericCommandImpl) SetConfig(config GenericConfig) { cmd.config = config }
func (cmd GenericCommandImpl) Url() string { return cmd.url }
func (cmd GenericCommandImpl) User() config.User { return cmd.user }

func (cmd GenericCommandImpl) Method() string { return cmd.config.Method() }
func (cmd GenericCommandImpl) Path() string { return cmd.config.Path() }
func (cmd GenericCommandImpl) ResType() reflect.Type { return cmd.config.ResType() }
func (cmd GenericCommandImpl) FieldName() string { return cmd.config.FieldName() }

func (cmd GenericCommandImpl) Execute() error {
	return MakeHttpRequest(cmd)
}

func (cmd GenericCommandImpl) ParseJSON(body io.ReadCloser) error {

	if cmd.config.ResType() == nil {
		io.Copy(os.Stdout, body)
		return nil
	}

	v := reflect.New(cmd.config.ResType())
	decodeRes := v.Interface()
	printRes := decodeRes

	if err := json.NewDecoder(body).Decode(decodeRes); err != nil {
		return fmt.Errorf("Error decoding controller state: %v", err)
	} else {
		if cmd.config.FieldName() != "" {
			printRes = v.Elem().FieldByName(cmd.config.FieldName()).Interface()
		}

		outputter := log.New(os.Stdout, "", 0)
		outputter.Printf("")

		output := util.PrettySprintf("", printRes)
		outputter.Printf(output)
		return nil
	}
}

type GenericCompositionalCommandImpl struct {
	url    string
	user   config.User
	subCmd string

	config GenericCompositionalConfigImpl
}

func (cmd GenericCompositionalCommandImpl) Url() string { return cmd.url }
func (cmd GenericCompositionalCommandImpl) User() config.User { return cmd.user }
func (cmd GenericCompositionalCommandImpl) SubCmd() string { return cmd.subCmd }

func (cmd GenericCompositionalCommandImpl) Execute() error {
	if subCmd, err := cmd.config.SubCommand(cmd); err != nil {
		return fmt.Errorf("CompositionalCommand.Execute: SubCommand: %v", err)
	} else {
		return subCmd.Execute()
	}

	return nil
}
