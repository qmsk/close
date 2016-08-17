package docker

import (
	"github.com/qmsk/close/docker"
	"reflect"
	"github.com/qmsk/close/shell"
)

var InfoConfig = shell.GenericConfigImpl {
	Path:     "/api/docker",
	ResType:  reflect.TypeOf((*docker.Info)(nil)).Elem(),
}
