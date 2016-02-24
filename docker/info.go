package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "regexp"
    "strings"
    "time"
)

type SwarmInfo struct {
    Role        string
    Strategy    string
    NodeCount   int

    Dump        string
}

type MemoryInfo struct {
    Size        float64
    Unit        string
}

type NodeInfo struct {
    Name        string
    Addr        string

    SwarmStatus     string
    SwarmError      error
    SwarmUpdated    time.Time

    Containers  int
    CPU         int
    CPUReserved int
    Memory          MemoryInfo
    MemoryReserved  MemoryInfo
    Labels          map[string]string
}

var swarmLabelRegexp = regexp.MustCompile("([a-z0-9-.]+)=(.*?)(, |$)")

func (node *NodeInfo) parseSwarmLabels(labels string) error {
    node.Labels = make(map[string]string)

    for _, match := range swarmLabelRegexp.FindAllStringSubmatch(labels, -1) {
        node.Labels[match[1]] = match[2]
    }

    return nil
}

type Info struct {
    Name            string
    ServerVersion   string
    OperatingSystem string

    Swarm           *SwarmInfo
    Nodes           []*NodeInfo
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

const swarmAttrPrefix = "  └ "

// Decode the human-readable swarm 1.1.x info output. Because it's the only way.
// https://github.com/docker/swarm/blob/v1.1.0/cluster/swarm/cluster.go#L828
func (info *Info) decodeSwarmStatus(systemStatus swarmSystemStatus) error {
    var node *NodeInfo

    info.Swarm = &SwarmInfo{
        Dump:   fmt.Sprintf("%#v", systemStatus),
    }

    for _, line := range systemStatus {
        if !strings.HasPrefix(line[0], " ") {
            // top-level attr
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
        } else if strings.HasPrefix(line[0], swarmAttrPrefix) {
            // node attr
            attr := strings.TrimPrefix(line[0], swarmAttrPrefix)
            value := line[1]

            if node == nil {
                log.Printf("Info: ignore attr=%v for node=%v\n", attr, node)
            }

            switch attr {
            case "Status":
                node.SwarmStatus = value
            case "Containers":
                fmt.Sscanf(value, "%d", &node.Containers)
            case "Reserved CPUs":
                fmt.Sscanf(value, "%d / %d", &node.CPUReserved, &node.CPU)
            case "Reserved Memory":
                fmt.Sscanf(value, "%f %s / %f %s", &node.MemoryReserved.Size, &node.MemoryReserved.Unit, &node.Memory.Size, &node.Memory.Unit)
            case "Labels":
                if err := node.parseSwarmLabels(value); err != nil {
                    log.Printf("Info: skip attr=%v for node=%v: value=%#v err=%v\n", attr, node, value, err)
                }
            case "Error":
                if value == "(none)" {
                    node.SwarmError = nil
                } else {
                    node.SwarmError = fmt.Errorf("%s", value)
                }
            case "UpdatedAt":
                if t, err := time.Parse("2006-01-02T15:04:05Z", value); err != nil {
                    log.Printf("Info: skip attr=%v for node=%v: value=%#v err=%v\n", attr, node, value, err)
                } else {
                    node.SwarmUpdated = t
                }
            default:
                log.Printf("Info: skip attr=%v for node=%v\n", attr, node)
            }

        } else {
            // node
            node = &NodeInfo{
                Name:   strings.TrimSpace(line[0]),
                Addr:   line[1],
            }

            info.Nodes = append(info.Nodes, node)
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
