package main

import (
	"github.com/qmsk/close/shell/docker"
)

func init() {
	Opts.Register("docker", &docker.DockerConfig{})

	Opts.RegisterSub("docker", "info", &docker.InfoConfig)
	Opts.RegisterSub("docker", "list", &docker.ListConfig)
}
