package main

import (
//	"net/http"
	"log"
)

type DockerConfig struct {
	SubCmd     struct {} `command:"subcommand"`
}

type DockerCmd struct {
}

func init() {
	Opts.Register("docker", &DockerConfig{})
}

func (cmd DockerCmd) Execute() error {
	log.Printf("command Docker execute: ")
	return nil
}

func (config DockerConfig) Command() (Command, error) {
	dockerCmd := &DockerCmd{
	}
	return dockerCmd, nil
}


