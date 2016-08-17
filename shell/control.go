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
