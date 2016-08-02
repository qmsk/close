package main

import (
	"github.com/qmsk/close/shell"
)

var Opts shell.Options

func main() {
	Opts.Parse()

	shell.Main(Opts)
}
