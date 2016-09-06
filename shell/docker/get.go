package docker

import (
	"github.com/qmsk/close/shell/command"
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

var GetConfig = command.NewGenericConfigImpl(
	"GET", "/api/docker/", reflect.TypeOf((*docker.Container)(nil)).Elem(), "",
	reflect.TypeOf((*getConfig)(nil)).Elem())

