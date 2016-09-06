package command

import (
	"github.com/qmsk/close/shell/config"
	"fmt"
	"reflect"
	"github.com/qmsk/close/util"
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

	// Additional parameters of the path can be configured through this field
	extra     interface{}
}

func (config GenericConfigImpl) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand := &GenericCommandImpl{
		url: options.Url(),
		user: options.User(),
		config: config,
	}

	return genericCommand, nil
}

func NewGenericConfigImpl(method string, path string, resType reflect.Type, fieldName string, configType reflect.Type) config.CommandConfig {
	// Reflect black magic here, allowing to extend
	// GenericConfigImpl with request parameters in the path
	if configType == nil {
		configType = reflect.TypeOf((*GenericConfigImpl)(nil)).Elem()
	}
	// Create new value of given config type (will be a pointer)
	cfgPtrToV := reflect.New(configType)
	// dereference the pointer
	cfgV := cfgPtrToV.Elem()
	// go back to the original variable
	cfg := cfgPtrToV.Interface()

	// Try getting the embedded field, if the given config type
	// is the extension of the GenericConfigImpl
	genCfgV := cfgV.FieldByName("GenericConfigImpl")

	genCfg := cfg
	if genCfgV.IsValid() {
		// If it is the extension, create the empty embedded GenericConfigImpl instance
		genCfgV.Set(reflect.ValueOf(&GenericConfigImpl{}))
		genCfg = genCfgV.Interface()
	}

	// Casts assume the correct configType was given
	genCfg.(*GenericConfigImpl).init(method, path, resType, fieldName, cfg)
	return cfg.(config.CommandConfig)
}

func (config *GenericConfigImpl) init(method string, path string, resType reflect.Type, fieldName string, extra interface{}) {
	config.method = method
	config.path = path
	config.resType = resType
	config.fieldName = fieldName
	config.extra = extra
}

func (config GenericConfigImpl) Method() string { return config.method }
func (config GenericConfigImpl) ResType() reflect.Type { return config.resType }
func (config GenericConfigImpl) FieldName() string { return config.fieldName }

func (config GenericConfigImpl) Path() string {
	if reflect.TypeOf(config.extra) == reflect.TypeOf(config) {
		return config.path
	} else {
		if path, err := util.ExpandPath(config.path, config.extra); err != nil {
			return ""
		} else {
			return path
		}
	}
}

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
