package control

import (
    "bytes"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "sort"
    "strings"
)

const DOCKER_STOP_TIMEOUT = 10 // seconds

type DockerID struct {
    WorkerType  string          `json:"worker_type"`
    WorkerID    uint            `json:"worker_id"`
}

// Docker name
func (self DockerID) String() string {
    return fmt.Sprintf("close_%s_%d", self.WorkerType, self.WorkerID)
}

func (self DockerID) labels() map[string]string {
    return map[string]string{
        "close.worker":     self.WorkerType,
        "close.worker-id":  fmt.Sprintf("%d", self.WorkerID),
    }
}

func (self *DockerID) parseID(name string, labels map[string]string) error {
    // docker swarm "/node/name"
    // docker "/name"
    namePath := strings.Split(name, "/")
    name = namePath[len(namePath) - 1]

    if worker := labels["close.worker"]; worker == "" {
        return fmt.Errorf("missing close.worker=")
    } else {
        self.WorkerType = worker
    }

    if workerID := labels["close.worker-id"]; workerID == "" {
        return fmt.Errorf("missing close.worker-id=")
    } else if _, err := fmt.Sscan(workerID, &self.WorkerID); err != nil {
        return fmt.Errorf("invalid close.worker-id=%v: %v", workerID, err)
    }

    if name != self.String() {
        return fmt.Errorf("name mismatch %v: %v", name, self)
    }

    return nil
}

// configuration for container
// read-only; requires remove/create to change
type DockerConfig struct {
    Image       string          `json:"image"`
    Command     string          `json:"command"`
    Args        []string        `json:"args"`
    Env         []string        `json:"env"`
}

func configFromDocker(dockerContainer *docker.Container) DockerConfig {
    return DockerConfig{
        Image:          dockerContainer.Config.Image,
        Command:        dockerContainer.Config.Cmd[0],
        Args:           dockerContainer.Config.Cmd[1:],
        Env:            dockerContainer.Config.Env,
    }
}

func (self *DockerConfig) normalize() {
    sort.Strings(self.Env)
}

// Compare config against running config for compatibility
// The running config will include additional stuff from the image..
func (self DockerConfig) Equals(other DockerConfig) bool {
    if other.Image != self.Image {
        return false
    }
    if other.Command != self.Command {
        return false
    }

    // args must match exactly
    if len(self.Args) != len(other.Args) {
        return false
    }
    for i := 0; i < len(self.Args) && i < len(other.Args); i++ {
        if self.Args[i] != other.Args[i] {
            return false
        }
    }

    // env needs to be a subset
checkEnv:
    for i, j := 0, 0; i < len(self.Env); i++ {
        for ; j < len(other.Env); j++ {
            if self.Env[i] == other.Env[j] {
                continue checkEnv // next self.Env[i++]
            }
        }

        // inner for loop went to end of other.Env[j]
        return false
    }

    return true
}

// Running docker container
type DockerContainer struct {
    DockerID

    // Config
    Config      DockerConfig    `json:"config"`

    // Status
    ID          string          `json:"id"`
    Node        string          `json:"node"`
    Name        string          `json:"name"`
    Status      string          `json:"status"`
    Running     bool            `json:"running"`
}

func (self *DockerConfig) AddFlag(name string, value interface{}) {
    arg := fmt.Sprintf("-%s=%v", name, value)

    self.Args = append(self.Args, arg)
}

func (self *DockerConfig) AddArg(arg string) {
    self.Args = append(self.Args, arg)
}

func (self *DockerConfig) AddEnv(name string, value interface{}) {
    env := fmt.Sprintf("%s=%v", name, value)

    self.Env = append(self.Env, env)
}

func (self *DockerContainer) updateStatus(dockerContainer *docker.Container) {
    // docker swarm "/node/name"
    // docker "/name"
    namePath := strings.Split(dockerContainer.Name, "/")
    name := namePath[len(namePath) - 1]

    self.ID = dockerContainer.ID
    self.Name = name

    if dockerContainer.Node != nil {
        self.Node = dockerContainer.Node.Name
    }

    self.Status = dockerContainer.State.String()
    self.Running = dockerContainer.State.Running
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
        Config: configFromDocker(dockerContainer),
    }

    container.Config.normalize()

    // ID
    if err := container.parseID(dockerContainer.Name, dockerContainer.Config.Labels); err != nil {
        return container, err
    }

    container.updateStatus(dockerContainer)

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

func (self *Manager) DockerUp(id DockerID, config DockerConfig) (DockerContainer, error) {
    config.normalize()

    // check
    container, err := self.DockerGet(id.String())

    if _, ok := err.(*docker.NoSuchContainer); ok {
        // create
        container = DockerContainer{DockerID:id, Config: config}

    } else if err != nil {
        return container, err

    } else if config.Equals(container.Config) {
        log.Printf("Manager.DockerUp %v: exists\n", container)

    } else {
        log.Printf("Manager.DockerUp %v: old-config %v\n", container, container.Config)
        log.Printf("Manager.DockerUp %v: new-config %v\n", container, config)

        // cleanup to replace with our config
        log.Printf("Manager.DockerUp %v: destroy %v...\n", container, container.ID)

        if err := self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID, Force: true}); err != nil {
            return container, fmt.Errorf("dockerClient.RemoveContainer %v: %v", container.ID, err)
        }

        // create
        container = DockerContainer{DockerID:id, Config: config}
    }

    if container.ID == "" {
        // does not exist; create
        createOptions := docker.CreateContainerOptions{
            Name:   container.String(),
            Config: &docker.Config{
                Hostname:   container.String(),
                Env:        container.Config.Env,
                Cmd:        append([]string{container.Config.Command}, container.Config.Args...),
                Image:      container.Config.Image,
                Labels:     container.labels(),
            },
            HostConfig: &docker.HostConfig{

            },
        }

        log.Printf("Manager.DockerUp %v: create...\n", container)

        if dockerContainer, err := self.dockerClient.CreateContainer(createOptions); err != nil {
            return container, err
        } else {
            // status
            container.updateStatus(dockerContainer)
        }
    }

    // running
    if container.Running {
        log.Printf("Manager.DockerUp %v: running\n", container)
    } else if err := self.dockerClient.StartContainer(container.ID, nil); err != nil {
        return container, fmt.Errorf("dockerClient.StartContainer %v: %v", container.ID, err)
    } else {
        log.Printf("Manager.DockerUp %v: started\n", container)

        container.Running = true
    }

    return container, nil
}
