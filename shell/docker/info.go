package docker

import (
	"github.com/qmsk/close/docker"
	"reflect"
	"github.com/qmsk/close/shell"
)

var InfoConfig = shell.GenericConfigImpl {
	"/api/docker",
	reflect.TypeOf((*docker.Info)(nil)).Elem(),
}
