package main

import (
	"github.com/qmsk/close/shell/control"
)

func init() {
	Opts.Register("clean", control.CleanConfig)
	Opts.Register("clients", control.ClientsConfig)
	Opts.Register("config", control.DumpConfigTextConfig)
	Opts.Register("stop", control.StopConfig)

	Opts.Register("workers", control.WorkersConfig)
	Opts.RegisterSub("workers", "ls", control.WorkersListConfig)
	Opts.RegisterSub("workers", "config", control.ConfigListConfig)
	Opts.RegisterSub("workers", "get", control.GetConfig)
}
