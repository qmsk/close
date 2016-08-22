package docker

import (
	"reflect"
	"github.com/qmsk/close/shell"
)

type logsConfigId struct {
	Id  string   `positional-arg-name:"id"`
}

type logsConfig struct {
	*shell.GenericConfigImpl
	LogsCfgID    logsConfigId   `positional-args:"id-struct"`
}

var LogsConfig = &logsConfig {
	shell.NewGenericConfigImpl(
		"", reflect.TypeOf((*string)(nil)).Elem(), ""),
	logsConfigId{},
}

func (config logsConfig) Command(options shell.CommonOptions) (shell.Command, error) {
	genericCommand, err := config.GenericConfigImpl.Command(options)
	// XXX This looks hacky, but how else to insert logsConfig into the GenericCommand?
	genericCommand.(*shell.GenericCommandImpl).SetConfig(config)
	return genericCommand, err
}

func (config logsConfig) Path() string {
	// Could implement something with path parameters but is it worth it?
	// withPathParam := config.GenericConfigImpl.Path()
	return "/api/docker/" + config.LogsCfgID.Id + "/logs"
}
