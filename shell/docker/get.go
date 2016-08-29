package docker

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/shell/config"
	"github.com/qmsk/close/docker"
	"reflect"
)

type getConfigId struct {
	Id  string   `positional-arg-name:"id"`
}

type getConfig struct {
	*command.GenericConfigImpl
	GetCfgID    getConfigId   `positional-args:"id-struct"`
}

var GetConfig = &getConfig {
	command.NewGenericConfigImpl(
		"GET", "/api/docker/", reflect.TypeOf((*docker.Container)(nil)).Elem(), ""),
	getConfigId{},
}

func (config getConfig) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	// XXX This looks hacky, but how else to insert getConfig into the GenericCommand?
	genericCommand.(*command.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config getConfig) Path() string {
	return config.GenericConfigImpl.Path() + config.GetCfgID.Id
}
