package shell

import (
	"github.com/qmsk/close/control"
	"reflect"
	"github.com/qmsk/close/shell/command"
)

var DumpConfigTextConfig = command.NewGenericConfigImpl (
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"ConfigText",
)

var ClientsConfig = command.NewGenericConfigImpl (
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Clients",
)

var WorkersConfig = command.NewGenericConfigImpl (
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
)
