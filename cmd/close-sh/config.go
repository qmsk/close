package main

import (
	"github.com/qmsk/close/shell"
)

func init() {
	Opts.Register("config", &shell.DumpConfigCmdConfig{})
}
