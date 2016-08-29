package docker

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/shell/config"
	"reflect"
)

type logsConfigId struct {
	Id  string   `positional-arg-name:"id"`
}

type logsConfig struct {
	*command.GenericConfigImpl
	LogsCfgID    logsConfigId   `positional-args:"id-struct"`
}

var LogsConfig = &logsConfig {
	command.NewGenericConfigImpl(
		"GET", "", reflect.TypeOf((*string)(nil)).Elem(), ""),
	logsConfigId{},
}

func (config logsConfig) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	// XXX This looks hacky, but how else to insert logsConfig into the GenericCommand?
	genericCommand.(*command.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config logsConfig) Path() string {
	// Could implement something with path parameters but is it worth it?
	// withPathParam := config.GenericConfigImpl.Path()
	return "/api/docker/" + config.LogsCfgID.Id + "/logs"
}
