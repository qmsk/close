package shell

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/control"
	"reflect"
)

var DumpConfigTextConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"ConfigText",
)

var ClientsConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Clients",
)

var WorkersConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
)

var StopConfig = command.NewGenericConfigImpl (
	"POST",
	"/api/stop",
	nil,
	"",
)

var CleanConfig = command.NewGenericConfigImpl (
	"POST",
	"/api/clean",
	nil,
	"",
)
