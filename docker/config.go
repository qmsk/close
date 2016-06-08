package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "encoding/json"
    "github.com/qmsk/close/util"
)

// configuration for container
// read-only; requires remove/create to change
type Config struct {
    Image       string          `json:"image"`
    Command     string          `json:"command"`
    Args        []string        `json:"args"`
    Env         util.StringSet          `json:"env"`

    Privileged      bool                `json:"privileged"`
    Mounts          []docker.Mount      `json:"mounts"`
    NetworkMode     string              `json:"net_container"`

    Constraints     util.StringSet
}

func (self *Config) Argv() []string {
    if self.Command == "" {
        return nil
    } else {
        return append([]string{self.Command}, self.Args...)
    }
}

func (self *Config) AddFlag(name string, value interface{}) {
    arg := fmt.Sprintf("-%s=%v", name, value)

    self.Args = append(self.Args, arg)
}

func (self *Config) AddArg(args ...string) {
    self.Args = append(self.Args, args...)
}

func (self *Config) AddMount(name string, bind string, readonly bool) {
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

func (self *Config) SetNetworkContainer(id ID) {
    self.NetworkMode = fmt.Sprintf("container:%s", id.String())
}

func configFromDocker(dockerContainer *docker.Container) Config {
    var constraints []string

    if constraintsLabel, exists := dockerContainer.Config.Labels["com.docker.swarm.constraints"]; !exists {

    } else if err := json.Unmarshal([]byte(constraintsLabel), &constraints); err != nil {
        // XXX
    } else {
        // ok
    }

    return Config{
        Image:          dockerContainer.Config.Image,
        Command:        dockerContainer.Config.Cmd[0],
        Args:           dockerContainer.Config.Cmd[1:],
        Env:            util.MakeStringSet(dockerContainer.Config.Env...),

        Privileged:     dockerContainer.HostConfig.Privileged,
        Mounts:         dockerContainer.Mounts,
        NetworkMode:    dockerContainer.HostConfig.NetworkMode,

        Constraints:    util.MakeStringSet(constraints...),
    }
}

// Compare config against running config for compatibility
// The running config will include additional stuff from the image..
func (self Config) Equals(other Config) bool {
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
    if !self.Env.Subset(other.Env) {
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

    if !self.Constraints.Equals(other.Constraints) {
        return false
    }

    return true
}


func (config Config) createOptions(id ID) docker.CreateContainerOptions {
    env := config.Env.Copy()

    for _, constraint := range config.Constraints {
        env.Add(fmt.Sprintf("constraint:%s", constraint))
    }

    createOptions := docker.CreateContainerOptions{
        Name:   id.String(),
        Config: &docker.Config{
            Env:        env,
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
        createOptions.Config.Hostname = id.String()
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

    return createOptions
}
