package main

import (
	"github.com/qmsk/close/shell/docker"
)

func init() {
	Opts.Register("docker", docker.NewConfig())
}
