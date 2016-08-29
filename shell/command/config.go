package command

import (
	"github.com/qmsk/close/shell/config"
	"fmt"
	"reflect"
)

type GenericConfig interface {
	Method()    string
	Path()      string
	ResType()   reflect.Type
	FieldName() string
}

type GenericConfigImpl struct {
	method    string
	path      string

	resType   reflect.Type
	fieldName string
}

func (config GenericConfigImpl) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand := &GenericCommandImpl{
		url: options.Url(),
		user: options.User(),
		config: config,
	}

	return genericCommand, nil
}

func NewGenericConfigImpl(method string, path string, resType reflect.Type, fieldName string) *GenericConfigImpl {
	config := &GenericConfigImpl {}
	config.init(method, path, resType, fieldName)
	return config
}

func (config *GenericConfigImpl) init(method string, path string, resType reflect.Type, fieldName string) {
	config.method = method
	config.path = path
	config.resType = resType
	config.fieldName = fieldName
}

func (config GenericConfigImpl) Method() string { return config.method }
func (config GenericConfigImpl) Path() string { return config.path }
func (config GenericConfigImpl) ResType() reflect.Type { return config.resType }
func (config GenericConfigImpl) FieldName() string { return config.fieldName }

type GenericCompositionalConfigImpl struct {
	subCommands map[string]config.CommandConfig
}

func (cfg *GenericCompositionalConfigImpl) Register(subcmd string, cmdConfig config.CommandConfig) {
	if cfg.subCommands == nil {
		cfg.subCommands = make(map[string]config.CommandConfig)
	}
	cfg.subCommands[subcmd] = cmdConfig
}

func (cfg GenericCompositionalConfigImpl) SubCommands() map[string]config.CommandConfig {
	return cfg.subCommands
}

func (cfg GenericCompositionalConfigImpl) SubCommand(options config.CommonOptions) (config.Command, error) {
	if opts, hasSubCmd := options.(config.CompositionalCommonOptions); !hasSubCmd {
		return nil, fmt.Errorf("trying to get a subcommand but provided options have no subcommand specified")
	} else {
		return cfg.subCommands[opts.SubCmd()].Command(options)
	}
}

func (cfg GenericCompositionalConfigImpl) Command(options config.CommonOptions) (config.Command, error) {
	if opts, hasSubCmd := options.(config.CompositionalCommonOptions); !hasSubCmd {
		return nil, fmt.Errorf("trying to create a compositional command but provided options have no subcommand specified")
	} else {
		cmd := &GenericCompositionalCommandImpl{
			url:    opts.Url(),
			user:   opts.User(),
			subCmd: opts.SubCmd(),
			config: cfg,
		}
		return cmd, nil
	}
}
