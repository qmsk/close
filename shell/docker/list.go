package docker

import (
	"github.com/qmsk/close/docker"
	"reflect"
	"github.com/qmsk/close/shell"
)

var ListConfig = shell.GenericConfigImpl {
	Path:     "/api/docker/",
	ResType:  reflect.SliceOf(reflect.TypeOf((*docker.ContainerStatus)(nil))),
}
