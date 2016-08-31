package control

import (
	"github.com/qmsk/close/shell/command"
	managerconfig "github.com/qmsk/close/config"
	"github.com/qmsk/close/shell/config"
	"github.com/qmsk/close/control"
	"reflect"
)

var WorkersListConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
)


type listConfigType struct {
	Type      string   `positional-arg-name:"type"`
}

type listConfig struct {
	*command.GenericConfigImpl
	ListCfgType   listConfigType    `positional-args:"type-struct"`
}

var ConfigListConfig = &listConfig {
	command.NewGenericConfigImpl(
		"GET", "/api/config/", reflect.SliceOf(reflect.TypeOf((*control.ConfigItem)(nil))), ""),
	listConfigType{},
}

func (config listConfig) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	genericCommand.(*command.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config listConfig) Path() string {
	return config.GenericConfigImpl.Path() + config.ListCfgType.Type
}

type getConfigTypeInstance struct {
	Type            string   `positional-arg-name:"type"`
	Instance        string   `positional-arg-name:"instance"`
}

type getConfig struct {
	*command.GenericConfigImpl
	GetCfgTypeInstance  getConfigTypeInstance    `positional-args:"type-instance-struct"`
}

var GetConfig = &getConfig {
	command.NewGenericConfigImpl(
		"GET", "/api/config/", reflect.TypeOf((*managerconfig.Config)(nil)).Elem(), ""),
	getConfigTypeInstance{},
}

func (config getConfig) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	genericCommand.(*command.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config getConfig) Path() string {
	return config.GenericConfigImpl.Path() + config.GetCfgTypeInstance.Type + "/" + config.GetCfgTypeInstance.Instance
}
