package docker

import (
	"github.com/qmsk/close/shell/command"
	"reflect"
)

type logsConfigId struct {
	Id  string   `positional-arg-name:"id"`
}

type logsConfig struct {
	*command.GenericConfigImpl
	LogsCfgID    logsConfigId   `positional-args:"id-struct"`
}

var LogsConfig = command.NewGenericConfigImpl(
		"GET", "", reflect.TypeOf((*string)(nil)).Elem(), "",
		reflect.TypeOf((*logsConfig)(nil)).Elem())

func (config logsConfig) Path() string {
	// Could implement something with path parameters but is it worth it?
	// withPathParam := config.GenericConfigImpl.Path()
	return "/api/docker/" + config.LogsCfgID.Id + "/logs"
}
