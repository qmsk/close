package control

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/control"
	"reflect"
)

var WorkersListConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
)

var ConfigListConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/config/",
	reflect.SliceOf(reflect.TypeOf((*control.ConfigItem)(nil))),
	"",
)
