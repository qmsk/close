package control

import (
	"github.com/qmsk/close/shell/command"
	managerconfig "github.com/qmsk/close/config"
	"github.com/qmsk/close/control"
	"reflect"
)

var WorkersConfig = &command.GenericCompositionalConfigImpl {}

var WorkersListConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/",
	reflect.TypeOf((*control.APIGet)(nil)).Elem(),
	"Workers",
	nil,
)


type listConfigType struct {
	Type      string   `positional-arg-name:"type"`
}

type listConfig struct {
	*command.GenericConfigImpl
	ListCfgType   listConfigType    `positional-args:"type-struct"`
}

var ConfigListConfig = command.NewGenericConfigImpl(
	"GET",
	"/api/config/:type",
	reflect.SliceOf(reflect.TypeOf((*control.ConfigItem)(nil))),
	"",
	reflect.TypeOf((*listConfig)(nil)).Elem())

type getConfigTypeInstance struct {
	Type            string   `positional-arg-name:"type"`
	Instance        string   `positional-arg-name:"instance"`
}

type getConfig struct {
	*command.GenericConfigImpl
	GetCfgTypeInstance  getConfigTypeInstance    `positional-args:"type-instance-struct"`
}

var GetConfig = command.NewGenericConfigImpl(
	"GET",
	"/api/config/:type/:instance",
	reflect.TypeOf((*managerconfig.Config)(nil)).Elem(),
	"",
	reflect.TypeOf((*getConfig)(nil)).Elem())
