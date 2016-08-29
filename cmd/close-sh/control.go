package main

import (
	"github.com/qmsk/close/shell"
)

func init() {
	Opts.Register("clean", shell.CleanConfig)
	Opts.Register("clients", shell.ClientsConfig)
	Opts.Register("config", shell.DumpConfigTextConfig)
	Opts.Register("stop", shell.StopConfig)
	Opts.Register("workers", shell.WorkersConfig)
}
