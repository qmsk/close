package docker

import (
	"github.com/qmsk/close/docker"
	"reflect"
	"github.com/qmsk/close/shell"
)

type getConfigId struct {
	Id  string   `positional-arg-name:"id"`
}

type getConfig struct {
	*shell.GenericConfigImpl
	GetCfgID    getConfigId   `positional-args:"id-struct"`
}

var GetConfig = &getConfig {
	shell.NewGenericConfigImpl(
		"/api/docker/", reflect.TypeOf((*docker.Container)(nil)).Elem(), ""),
	getConfigId{},
}

func (config getConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	// XXX This looks hacky, but how else to insert getConfig into the GenericCommand?
	genericCommand.(*shell.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config getConfig) Path() string {
	return config.GenericConfigImpl.Path() + config.GetCfgID.Id
}
