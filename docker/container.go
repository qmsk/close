package docker

import (
    "github.com/fsouza/go-dockerclient"
    "strings"
)

// Running docker container
type Container struct {
    ID

    // Config
    Config      Config    `json:"config"`

    // Status
    ContainerID     string          `json:"id"`
    Node            string          `json:"node"`
    Name            string          `json:"name"`
    State           docker.State    `json:"state"`
    Status          string          `json:"status"` // from State
}

func (self *Container) IsUp() bool {
    return self.State.Running
}

func (self *Container) updateStatus(dockerContainer *docker.Container) {
    // docker swarm "/node/name"
    // docker "/name"
    namePath := strings.Split(dockerContainer.Name, "/")
    name := namePath[len(namePath) - 1]

    self.ContainerID = dockerContainer.ID
    self.Name = name

    if dockerContainer.Node != nil {
        self.Node = dockerContainer.Node.Name
    }

    self.State = dockerContainer.State
    self.Status = dockerContainer.State.String()
}

