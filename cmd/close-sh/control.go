package main

import (
	"github.com/qmsk/close/shell"
)

func init() {
	Opts.Register("config", shell.DumpConfigTextConfig)
	Opts.Register("clients", shell.ClientsConfig)
	Opts.Register("workers", shell.WorkersConfig)
}
