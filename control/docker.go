package control

import (
    "bytes"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "sort"
    "strings"
)

const DOCKER_STOP_TIMEOUT = 10 // seconds

type DockerID struct {
    Class       string          `json:"class"`
    Type        string          `json:"type"`
    Index       uint            `json:"index"`
}

// Docker name
func (self DockerID) String() string {
    return fmt.Sprintf("close-%s_%s_%d", self.Class, self.Type, self.Index)
}

func (self DockerID) labels() map[string]string {
    return map[string]string{
        "close":            self.Class,
        "close.type":       self.Type,
        "close.index":      fmt.Sprintf("%d", self.Index),
    }
}

func (self *DockerID) parseID(name string, labels map[string]string) error {
    // docker swarm "/node/name"
    // docker "/name"
    namePath := strings.Split(name, "/")
    name = namePath[len(namePath) - 1]

    if class := labels["close"]; class== "" {
        return fmt.Errorf("missing close.class=")
    } else {
        self.Class = class
    }

    if typ := labels["close.type"]; typ == "" {
        return fmt.Errorf("missing close.typ=")
    } else {
        self.Type = typ
    }

    if index := labels["close.index"]; index == "" {
        return fmt.Errorf("missing close.index=")
    } else if _, err := fmt.Sscan(index, &self.Index); err != nil {
        return fmt.Errorf("invalid close.index=%v: %v", index, err)
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

    Privileged      bool                `json:"privileged"`
    Mounts          []docker.Mount      `json:"mounts"`
    NetworkMode     string              `json:"net_container"`
}

func (self *DockerConfig) Argv() []string {
    if self.Command == "" {
        return nil
    } else {
        return append([]string{self.Command}, self.Args...)
    }
}

func (self *DockerConfig) AddFlag(name string, value interface{}) {
    arg := fmt.Sprintf("-%s=%v", name, value)

    self.Args = append(self.Args, arg)
}

func (self *DockerConfig) AddArg(args ...string) {
    self.Args = append(self.Args, args...)
}

func (self *DockerConfig) AddEnv(name string, value interface{}) {
    env := fmt.Sprintf("%s=%v", name, value)

    self.Env = append(self.Env, env)
}

func (self *DockerConfig) AddMount(name string, bind string, readonly bool) {
    mount := docker.Mount{
        Source:         bind,
        Destination:    name,
    }

    if readonly {
        mount.Mode = "ro"
        mount.RW = false
    }

    self.Mounts = append(self.Mounts, mount)
}

func (self *DockerConfig) SetNetworkContainer(container *DockerContainer) {
    self.NetworkMode = fmt.Sprintf("container:%s", container.String())
}

func configFromDocker(dockerContainer *docker.Container) DockerConfig {
    return DockerConfig{
        Image:          dockerContainer.Config.Image,
        Command:        dockerContainer.Config.Cmd[0],
        Args:           dockerContainer.Config.Cmd[1:],
        Env:            dockerContainer.Config.Env,
        Privileged:     dockerContainer.HostConfig.Privileged,
        Mounts:         dockerContainer.Mounts,
        NetworkMode:    dockerContainer.HostConfig.NetworkMode,
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

    // override command, or take from image?
    if self.Command != "" {
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

    if self.Privileged != other.Privileged {
        return false
    }

    // args must match exactly
    if len(self.Mounts) != len(other.Mounts) {
        return false
    }
    for i := 0; i < len(self.Mounts) && i < len(other.Mounts); i++ {
        if self.Mounts[i] != other.Mounts[i] {
            return false
        }
    }

    if self.NetworkMode == "" {

    } else if other.NetworkMode != self.NetworkMode {
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
    State       docker.State    `json:"state"`
    Status      string          `json:"status"` // from State
}

func (self *DockerContainer) IsUp() bool {
    return self.State.Running
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

    self.State = dockerContainer.State
    self.Status = dockerContainer.State.String()
}

func (self *Manager) DockerList() (containers []*DockerContainer, err error) {
    opts := docker.ListContainersOptions{
        All:        true,
        Filters:    map[string][]string{
            "label":    []string{"close"},
        },
    }

    if listContainers, err := self.dockerClient.ListContainers(opts); err != nil {
        return nil, err
    } else {
        for _, listContainer := range listContainers {
            if container, err := self.DockerGet(listContainer.ID); err != nil {
                return nil, err
            } else {
                containers = append(containers, container)
            }
        }
    }

    return containers, nil
}

func (self *Manager) DockerGet(id string) (*DockerContainer, error) {
    dockerContainer, err := self.dockerClient.InspectContainer(id)
    if _, ok := err.(*docker.NoSuchContainer); ok {
        return nil, nil

    } else if err != nil {
        return nil, fmt.Errorf("dockerClient.InspectContainer %v: %v", id, err)
    }

    container := DockerContainer{
        Config: configFromDocker(dockerContainer),
    }

    container.Config.normalize()

    // ID
    if err := container.parseID(dockerContainer.Name, dockerContainer.Config.Labels); err != nil {
        return nil, err
    }

    container.updateStatus(dockerContainer)

    return &container, nil
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

func (self *Manager) DockerUp(id DockerID, config DockerConfig) (*DockerContainer, error) {
    config.normalize()

    // check
    container, err := self.DockerGet(id.String())

    if err != nil {
        return container, err
    } else if container == nil {
        // create

    } else if config.Equals(container.Config) {
        self.log.Printf("DockerUp %v: exists\n", container)

    } else {
        self.log.Printf("DockerUp %v: old-config %#v\n", container, container.Config)
        self.log.Printf("DockerUp %v: new-config %#v\n", container, config)

        // cleanup to replace with our config
        self.log.Printf("DockerUp %v: destroy %v...\n", container, container.ID)

        if err := self.dockerClient.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID, Force: true}); err != nil {
            return container, fmt.Errorf("dockerClient.RemoveContainer %v: %v", container.ID, err)
        }

        // create
        container = nil
    }

    if container == nil {
        container = &DockerContainer{DockerID:id, Config: config}

        // does not exist; create
        createOptions := docker.CreateContainerOptions{
            Name:   container.String(),
            Config: &docker.Config{
                Env:        config.Env,
                Cmd:        config.Argv(),
                Image:      config.Image,
                // Mounts:     config.Mounts,
                Labels:     id.labels(),
            },
            HostConfig: &docker.HostConfig{
                Privileged:     config.Privileged,
                NetworkMode:    config.NetworkMode,
            },
        }

        if config.NetworkMode == "" {
            // match hostname from container name, unless running with NetworkMode=container:*
            createOptions.Config.Hostname = container.String()
        }

        // XXX: .Config.Mounts = ... doesn't work? fake it!
        createOptions.Config.Volumes = make(map[string]struct{})
        for _, mount := range config.Mounts {
            createOptions.Config.Volumes[mount.Destination] = struct{}{}

            if mount.Source != "" {
                bind := fmt.Sprintf("%s:%s:%s", mount.Source, mount.Destination, mount.Mode)

                createOptions.HostConfig.Binds = append(createOptions.HostConfig.Binds, bind)
            }
        }

        self.log.Printf("DockerUp %v: create...\n", container)

        if dockerContainer, err := self.dockerClient.CreateContainer(createOptions); err != nil {
            return nil, err
        } else {
            // status
            container.updateStatus(dockerContainer)
        }
    }

    // running
    if container.IsUp() {
        self.log.Printf("DockerUp %v: running\n", container)
    } else if err := self.dockerClient.StartContainer(container.ID, nil); err != nil {
        return nil, fmt.Errorf("dockerClient.StartContainer %v: %v", container.ID, err)
    } else {
        self.log.Printf("DockerUp %v: started\n", container)

        container.State.Running = true // XXX
    }

    return container, nil
}

func (self *Manager) DockerDown(container *DockerContainer) error {
    self.log.Printf("DockerDown %v: stopping..\n", container)

    if err := self.dockerClient.StopContainer(container.ID, DOCKER_STOP_TIMEOUT); err == nil {

    } else if err, isNotRunning := err.(*docker.ContainerNotRunning); isNotRunning {
        // skip
    } else {
        return err
    }

    container.State.Running = false // XXX

    return nil
}

// Stop all and all running containers
func (self *Manager) DockerPanic() (retErr error) {
    opts := docker.ListContainersOptions{
        Filters:    map[string][]string{
            "label":    []string{"close"},
        },
    }

    if listContainers, err := self.dockerClient.ListContainers(opts); err != nil {
        return err
    } else {
        for _, listContainer := range listContainers {
            if err := self.dockerClient.KillContainer(docker.KillContainerOptions{ID: listContainer.ID, Signal: 9}); err != nil {
                self.log.Printf("DockerPanic: dockerClient.KillContainer %v: %v\n", listContainer.ID, err)
                retErr = err
            } else {
                self.log.Printf("DockerPanic %v: killed\n", listContainer.Names[0])
            }
        }
    }

    return retErr
}
