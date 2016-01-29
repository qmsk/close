package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "close/util"
)

// configuration for container
// read-only; requires remove/create to change
type Config struct {
    Image       string          `json:"image"`
    Command     string          `json:"command"`
    Args        []string        `json:"args"`
    Env         util.Env        `json:"env"`

    Privileged      bool                `json:"privileged"`
    Mounts          []docker.Mount      `json:"mounts"`
    NetworkMode     string              `json:"net_container"`
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
    return Config{
        Image:          dockerContainer.Config.Image,
        Command:        dockerContainer.Config.Cmd[0],
        Args:           dockerContainer.Config.Cmd[1:],
        Env:            util.MakeEnv(dockerContainer.Config.Env...),
        Privileged:     dockerContainer.HostConfig.Privileged,
        Mounts:         dockerContainer.Mounts,
        NetworkMode:    dockerContainer.HostConfig.NetworkMode,
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

    return true
}

