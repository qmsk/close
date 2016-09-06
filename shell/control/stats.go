package control

import (
	"github.com/qmsk/close/shell/command"
	"github.com/qmsk/close/stats"
	"reflect"
)

var StatsConfig = &command.GenericCompositionalConfigImpl {}

var StatsTypesConfig = command.NewGenericConfigImpl (
	"GET",
	"/api/stats",
	reflect.SliceOf(reflect.TypeOf((*stats.SeriesMeta)(nil))),
	"",
	nil,
)
