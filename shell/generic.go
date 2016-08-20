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

type GenericConfig interface {
	Path()      string
	ResType()   reflect.Type
	FieldName() string
}

type GenericCommand interface {
	CommonOptions
	GenericConfig
	JSONResponseParser
}

type GenericConfigImpl struct {
	path      string

	resType   reflect.Type
	fieldName string
}

// The separation of concepts is: URL and user are coming from
// the shell invoking user, config is coming from the command
// implementation, altogether they make a command
type GenericCommandImpl struct {
	url         string
	user        User
	config      GenericConfig
}

func (config GenericConfigImpl) Command(options CommonOptions) (Command, error) {
	genericCommand := &GenericCommandImpl{
		url: options.Url(),
		user: options.User(),
		config: config,
	}

	return genericCommand, nil
}

func NewGenericConfigImpl(path string, resType reflect.Type, fieldName string) *GenericConfigImpl {
	config := &GenericConfigImpl {}
	config.init(path, resType, fieldName)
	return config
}

func (config *GenericConfigImpl) init(path string, resType reflect.Type, fieldName string) {
	config.path = path
	config.resType = resType
	config.fieldName = fieldName
}

func (config GenericConfigImpl) Path() string { return config.path }
func (config GenericConfigImpl) ResType() reflect.Type { return config.resType }
func (config GenericConfigImpl) FieldName() string { return config.fieldName }

func (cmd *GenericCommandImpl) SetConfig(config GenericConfig) { cmd.config = config }
func (cmd GenericCommandImpl) Url() string { return cmd.url }
func (cmd GenericCommandImpl) User() User { return cmd.user }

func (cmd GenericCommandImpl) Path() string { return cmd.config.Path() }
func (cmd GenericCommandImpl) ResType() reflect.Type { return cmd.config.ResType() }
func (cmd GenericCommandImpl) FieldName() string { return cmd.config.FieldName() }

func (cmd GenericCommandImpl) Execute() error {
	return MakeHttpRequest(cmd)
}

func (cmd GenericCommandImpl) ParseJSON(body io.ReadCloser) error {
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
