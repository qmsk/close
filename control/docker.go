package control

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
)

const DOCKER_LABEL="net.qmsk.close.worker"

type DockerContainer struct {
    Node        string
    ID          string
    Name        string
    Image       string
    Command     string
    Args        []string
    Env         []string
}

func (self *DockerContainer) AddFlag(name string, value interface{}) {
    arg := fmt.Sprintf("-%s=%v", name, value)

    self.Args = append(self.Args, arg)
}

func (self *DockerContainer) AddArg(arg string) {
    self.Args = append(self.Args, arg)
}

func (self *DockerContainer) AddEnv(name string, value interface{}) {
    env := fmt.Sprintf("%s=%v", name, value)

    self.Env = append(self.Env, env)
}

func (self *Manager) DockerList() (containers []DockerContainer, err error) {
    opts := docker.ListContainersOptions{
        All:        true,
        Filters:    map[string][]string{
            "label":    []string{DOCKER_LABEL},
        },
    }

    if listContainers, err := self.dockerClient.ListContainers(opts); err != nil {
        return nil, err
    } else {
        for _, listContainer := range listContainers {
            container := DockerContainer{
                ID:         listContainer.ID,
                Name:       listContainer.Names[0],
                Image:      listContainer.Image,
                Command:    listContainer.Command,
            }

            containers = append(containers, container)
        }
    }

    return containers, nil
}

func (self *Manager) DockerUp(container DockerContainer) (DockerContainer, error) {
    opts := docker.CreateContainerOptions{
        Name:   container.Name,
        Config: &docker.Config{
            Hostname:   container.Name,
            Env:        container.Env,
            Cmd:        append([]string{container.Command}, container.Args...),
            Image:      container.Image,
        },
    }

    // TODO: check existing?
    // TODO Node: Env constraint:node==

    if dockerContainer , err := self.dockerClient.CreateContainer(opts); err == nil {
        container.ID = dockerContainer.ID
    } else if alreadyRunning, ok := err.(*docker.ContainerAlreadyRunning); ok {
        container.ID = alreadyRunning.ID
    } else {
        return container, err
    }

    return container, nil
}
