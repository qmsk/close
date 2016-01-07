package control

import (
    "bytes"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "strings"
)

const DOCKER_LABEL="net.qmsk.close.worker"

type DockerContainer struct {
    Name        string          `json:"name"`
    Image       string          `json:"image"`
    Command     string          `json:"command"`
    Args        []string        `json:"args"`
    Env         []string        `json:"env"`

    WorkerType  string          `json:"worker_type"`
    WorkerID    uint            `json:"worker_id"`

    // Status
    Node        string          `json:"node"`
    ID          string          `json:"id"`
    Status      string          `json:"status"`
    Running     bool            `json:"running"`
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
            "label":    []string{"close.worker"},
        },
    }

    if listContainers, err := self.dockerClient.ListContainers(opts); err != nil {
        return nil, err
    } else {
        for _, listContainer := range listContainers {
            if container, err := self.DockerGet(listContainer.ID); err != nil {
                log.Printf("Manager.DockerList %v: %v\n", listContainer.ID, err)
                continue
            } else {
                containers = append(containers, container)
            }
        }
    }

    return containers, nil
}

func (self *Manager) DockerGet(id string) (DockerContainer, error) {
    dockerContainer, err := self.dockerClient.InspectContainer(id)
    if err != nil {
        return DockerContainer{}, fmt.Errorf("dockerClient.InspectContainer %v: %v", id, err)
    }

    container := DockerContainer{
        ID:             dockerContainer.ID,
        Image:          dockerContainer.Config.Image,
        Command:        dockerContainer.Config.Cmd[0],
        Args:           dockerContainer.Config.Cmd[1:],
        Env:            dockerContainer.Config.Env,

        WorkerType:     dockerContainer.Config.Labels["close.worker"],

        Status:         dockerContainer.State.String(),
        Running:        dockerContainer.State.Running,
    }

    // parse name
    nameParts := strings.Split(dockerContainer.Name, "/")

    if len(nameParts) == 3 {
        // docker swarm "/node/name" style
        container.Node = nameParts[1]
        container.Name = nameParts[2]
    } else if len(nameParts) == 2 {
        // normal docker "/name"
        container.Node = self.dockerName
        container.Name = nameParts[1]
    } else {
        return container, fmt.Errorf("invalid name=%v\n", dockerContainer.Name)
    }

    if dockerContainer.Node != nil {
        container.Node = dockerContainer.Node.Name
    } else {
        container.Node = self.dockerName
    }

    if _, err := fmt.Sscan(dockerContainer.Config.Labels["close.worker-id"], &container.WorkerID); err != nil {
        return container, fmt.Errorf("invalid close.worker-index=%v: %v", dockerContainer.Config.Labels["close.worker-id"], err)
    }

    return container, nil
}

func (self *Manager) DockerLogs(id string) (string, error) {
    var buf bytes.Buffer

    logsOptions := docker.LogsOptions{
        Container:      id,
        OutputStream:   &buf,
        ErrorStream:    &buf,
        Stdout:         true,
        Stderr:         true,
    }

    if err := self.dockerClient.Logs(logsOptions); err != nil {
        return "", err
    }

    return buf.String(), nil
}

func (self *Manager) DockerUp(container DockerContainer) (DockerContainer, error) {
    // TODO: check existing?
    // TODO Node: Env constraint:node==
    opts := docker.CreateContainerOptions{
        Name:   container.Name,
        Config: &docker.Config{
            Hostname:   container.Name,
            Env:        container.Env,
            Cmd:        append([]string{container.Command}, container.Args...),
            Image:      container.Image,
            Labels:     map[string]string{
                "close.worker": container.WorkerType,
                "close.worker-id":  fmt.Sprintf("%d", container.WorkerID),
            },
        },
        HostConfig: &docker.HostConfig{

        },
    }

    // create or get
    dockerContainer, err := self.dockerClient.CreateContainer(opts)
    if err == nil {
        log.Printf("Manager.DockerUp %v: created\n", container.Name)
    } else if err == docker.ErrContainerAlreadyExists {
        log.Printf("Manager.DockerUp %v: exists\n", container.Name)
        dockerContainer, err = self.dockerClient.InspectContainer(container.Name)
    }

    if err != nil {
        return container, err
    }

    // info
    container.ID = dockerContainer.ID

    if dockerContainer.Node != nil {
        container.Node = dockerContainer.Node.Name
    } else {
        container.Node = self.dockerName
    }
    container.Running = dockerContainer.State.Running
    container.Status = dockerContainer.State.String()

    // running
    if container.Running {
        log.Printf("Manager.DockerUp %v: running\n", container.Name)
    } else if err := self.dockerClient.StartContainer(container.ID, nil); err != nil {
        return container, fmt.Errorf("dockerClient.StartContainer %v: %v", container.ID, err)
    } else {
        log.Printf("Manager.DockerUp %v: started\n", container.Name)

        container.Running = true
    }

    return container, nil
}
