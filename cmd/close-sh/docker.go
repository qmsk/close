package main

import (
	"github.com/qmsk/close/shell/docker"
)

func init() {
	Opts.Register("docker", docker.DockerConfig)

	Opts.RegisterSub("docker", "info", docker.InfoConfig)
	Opts.RegisterSub("docker", "list", docker.ListConfig)
	Opts.RegisterSub("docker", "get", docker.GetConfig)
	Opts.RegisterSub("docker", "logs", docker.LogsConfig)
}
