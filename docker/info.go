package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
)

type SwarmInfo struct {
    Role        string
    Strategy    string
    NodeCount   int

    Dump        string
}

type Info struct {
    Name            string
    ServerVersion   string
    OperatingSystem string

    Swarm           *SwarmInfo
}

func (info *Info) decode(env *docker.Env) error {
    // XXX: go-dockerclient is crazy... it takes the JSON response, unmarshals it as a map, formats that as a []string{"KEY=VALUE"}, and then parses that again into a map for every .Get() call
    infoMap := env.Map()

    info.Name = infoMap["Name"]
    info.ServerVersion = infoMap["ServerVersion"]
    info.OperatingSystem = infoMap["OperatingSystem"]

    if env.Exists("SystemStatus") {
        var systemStatus swarmSystemStatus

        if err := env.GetJSON("SystemStatus", &systemStatus); err != nil {
            log.Printf("Info: unmarshal SystemStatus: %v\n", err)
        } else if err := info.decodeSwarmStatus(systemStatus); err != nil {
            log.Printf("Info: decode SystemStatus: %v\n", err)
        }
    }

    return nil
}

type swarmSystemStatus [][2]string

// Decode the human-readable swarm 1.1.x info output. Because it's the only way.
func (info *Info) decodeSwarmStatus(systemStatus swarmSystemStatus) error {
    info.Swarm = &SwarmInfo{
        Dump:   fmt.Sprintf("%#v", systemStatus),
    }

    for _, line := range systemStatus {
        switch line[0] {
        case "Role":
            info.Swarm.Role = line[1]
        case "Strategy":
            info.Swarm.Strategy = line[1]
        case "Nodes":
            if _, err := fmt.Sscanf(line[1], "%d", &info.Swarm.NodeCount); err != nil {
                return err
            }
        }
    }

    return nil
}

func (manager *Manager) Info() (info Info, err error) {
    if env, err := manager.dockerClient.Info(); err != nil {
        return info, fmt.Errorf("dockerClient.Info: %v", err)
    } else {
        return info, info.decode(env)
    }
}
