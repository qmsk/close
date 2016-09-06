package docker

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/docker"
	"reflect"
)

var InfoConfig = command.NewGenericConfigImpl(
	"GET", "/api/docker", reflect.TypeOf((*docker.Info)(nil)).Elem(), "", nil )
