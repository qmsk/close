package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "regexp"
    "strings"
)

// XXX: hack to reverse-engineer runtime state from status string, because the list JSON doesn't include that information
// https://github.com/docker/docker/blob/master/container/state.go#L42
// "Up 9 minutes"
// "Exited (0) 2 days ago",
// "Exited (137) 4 weeks ago"
var statusRegexp = regexp.MustCompile(`^(Up (.+)( \(Paused\))?|(Exited|Restarting) \((\d+\)) (.+) ago|Removal In Progress|Dead|Created|)$`)

func parseStatus(status string) (state string, exit int, since string) {
    match := statusRegexp.FindStringSubmatch(status)

    if match == nil {
        return
    }

    if match[2] != "" {
        since = match[2]
    } else if match[6] != "" {
        since = match[6]
    }

    if match[5] == "" {

    } else {
        fmt.Sscanf(match[5], "%d", &exit)
    }

    if match[1] == "Created" {
        state = "created"
    } else if match[1] == "Dead" {
        state = "dead"
    } else if match[1] == "Removal In Progress" {
        state = "removal" //
    } else if match[4] == "Restarting" {
        state = "restarting"
    } else if match[4] == "Exited" {
        state = "exited"
    } else if match[3] != "" {
        state = "paused"
    } else if match[2] != "" {
        state = "running"
    }

    return
}

// XXX: bcompat for missing State.Status in go-dockerclient
func stateStatus(state docker.State) (status string) {
    if state.Running && state.Paused {
        return "paused"
    } else if state.Running && state.Restarting {
        return "restarting"
    } else if state.Running {
        return "running"
    } else if state.StartedAt.IsZero() {
        return "created"
    } else {
        return "exited"
    }
}

type ContainerStatus struct {
    ID

    ContainerID     string          `json:"id"`
    Node            string          `json:"node"`
    Name            string          `json:"name"`

    Status          string          `json:"status"`     // human-readable
    State           string          `json:"state"`      // machine-readable
    ExitCode        int             `json:"exit_code"`
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

    if err := self.fromDockerName(apiContainers.Names[0]); err != nil {
        return err
    }

    self.Status = apiContainers.Status
    self.State, self.ExitCode, _ = parseStatus(apiContainers.Status) // poor-man's inspect

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
    self.State = stateStatus(dockerContainer.State) // XXX: .Status
    self.ExitCode = dockerContainer.State.ExitCode

    return nil
}

func (self *ContainerStatus) IsUp() bool {
    switch self.State {
    case "paused", "restarting", "running":
        return true
    case "dead", "created", "exited":
        return false
    default:
        // XXX
        return false
    }
}

func (self *ContainerStatus) IsError() bool {
    if self.ExitCode != 0 {
        return true
    }

    return false
}

// Running docker container
type Container struct {
    ContainerStatus

    // Config
    Config      Config    `json:"config"`

    // State
    // TODO: remove?
    ContainerState  docker.State    `json:"container_state"`
}

func (self *Container) update(dockerContainer *docker.Container) error {
    if err := self.ContainerStatus.fromDockerInspect(dockerContainer); err != nil {
        return err
    }

    self.Config = configFromDocker(dockerContainer)
    self.ContainerState = dockerContainer.State

    return nil
}
