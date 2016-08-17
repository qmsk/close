package shell

import (
	"github.com/qmsk/close/control"
	"reflect"
)

var DumpConfigTextConfig = GenericConfigImpl {
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"ConfigText",
}

var ClientsConfig = GenericConfigImpl {
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Clients",
}

var WorkersConfig = GenericConfigImpl {
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
}
