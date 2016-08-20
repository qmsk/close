package docker

import (
	"github.com/qmsk/close/docker"
	"reflect"
	"github.com/qmsk/close/shell"
)

var ListConfig = shell.NewGenericConfigImpl(
	"/api/docker/",
	reflect.SliceOf(reflect.TypeOf((*docker.ContainerStatus)(nil))),
	"",
)
