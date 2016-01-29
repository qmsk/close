package docker

import (
    "bytes"
    "github.com/fsouza/go-dockerclient"
)

func (manager *Manager) Logs(id string) (string, error) {
    var buf bytes.Buffer

    logsOptions := docker.LogsOptions{
        Container:      id,
        OutputStream:   &buf,
        ErrorStream:    &buf,
        Stdout:         true,
        Stderr:         true,
    }

    if err := manager.dockerClient.Logs(logsOptions); err != nil {
        return "", err
    }

    return buf.String(), nil
}
