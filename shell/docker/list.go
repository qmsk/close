package docker

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/docker"
	"reflect"
)

var ListConfig = command.NewGenericConfigImpl(
	"GET",
	"/api/docker/",
	reflect.SliceOf(reflect.TypeOf((*docker.ContainerStatus)(nil))),
	"",
	nil,
)
