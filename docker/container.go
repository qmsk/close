package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "strings"
)

type ContainerStatus struct {
    ID

    ContainerID     string          `json:"id"`
    Node            string          `json:"node"`
    Name            string          `json:"name"`
    Status          string          `json:"status"` // from State
}

// docker swarm:    /node/name
// docker:          /name
func (self *ContainerStatus) fromDockerName(namePath string) error {
    nameParts := strings.Split(namePath, "/")

    switch len(nameParts) {
    case 2:
        self.Name = nameParts[1]
    case 3:
        self.Node = nameParts[1]
        self.Name = nameParts[2]
    default:
        return fmt.Errorf("Invalid container name: %v", nameParts)
    }

    return nil
}

func (self *ContainerStatus) fromDockerList(apiContainers docker.APIContainers) error {
    self.ContainerID = apiContainers.ID
    self.Status = apiContainers.Status

    if err := self.fromDockerName(apiContainers.Names[0]); err != nil {
        return err
    }

    return nil
}

func (self *ContainerStatus) fromDockerInspect(dockerContainer *docker.Container) error {
    self.ContainerID = dockerContainer.ID

    if err := self.fromDockerName(dockerContainer.Name); err != nil {
        return err
    }

    if dockerContainer.Node != nil {
        self.Node = dockerContainer.Node.Name
    }

    self.Status = dockerContainer.State.String()

    return nil
}

// Running docker container
type Container struct {
    ContainerStatus

    // Config
    Config      Config    `json:"config"`

    // State
    State       docker.State    `json:"state"`
}

func (self *Container) update(dockerContainer *docker.Container) error {
    if err := self.ContainerStatus.fromDockerInspect(dockerContainer); err != nil {
        return err
    }

    self.Config = configFromDocker(dockerContainer)
    self.State = dockerContainer.State

    return nil
}

func (self *Container) IsUp() bool {
    return self.State.Running
}
