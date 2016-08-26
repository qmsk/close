package docker

import (
	"github.com/qmsk/close/shell/command"
)

// Docker is a CompositionalCommand, it has subcommands
var DockerConfig = &command.GenericCompositionalConfigImpl {}
