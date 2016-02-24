package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "os"
    "testing"
    "time"
)

func loadDockerInfo(file string) (env docker.Env) {
    if file, err := os.Open(file); err != nil {
        panic(err)
    } else if err := env.Decode(file); err != nil {
        panic(err)
    } else {
        return
    }
}

func TestSwarmInfo(t *testing.T) {
    var info Info

    env := loadDockerInfo("test/swarm-info.json")

    if err := info.decode(&env); err != nil {
        t.Fatalf("info.decode %#v: %v\n", env, err)
    }

    if info.Name != "catcp-terom-dev" {
        t.Errorf("info.Name: %#v\n", info.Name)
    }
    if info.ServerVersion != "swarm/1.1.1" {
        t.Errorf("info.ServerVersion: %#v\n", info.ServerVersion)
    }
    if info.OperatingSystem != "linux" {
        t.Errorf("info.OperatingSystem: %#v\n", info.OperatingSystem)
    }

    if info.Swarm == nil {
        t.Errorf("info.Swarm: nil\n")
    } else {
        if info.Swarm.Role != "primary" {
            t.Errorf("info.Swarm.Role: %#v\n", info.Swarm.Role)
        }
        if info.Swarm.Strategy != "random" {
            t.Errorf("info.Swarm.Strategy: %#v\n", info.Swarm.Strategy)
        }
        if info.Swarm.NodeCount != 9 {
            t.Errorf("info.Swarm.NodeCount: %#v\n", info.Swarm.NodeCount)
        }
    }

    if len(info.Nodes) < 2 {
        t.Errorf("info.Nodes: missing #1\n")
    } else {
        node := *info.Nodes[1]
        node1 := NodeInfo{
            Name:           "catcp3-terom-dev",
            Addr:           "catcp3-terom-dev.terom-dev.test.catcp:2376",
            SwarmStatus:    "Healthy",
            Containers:     21,
            CPU:            2,
            CPUReserved:    0,
            Memory:         MemoryInfo{1.026, "GiB"},
            MemoryReserved: MemoryInfo{0.0, "B"},
            Labels:         "executiondriver=native-0.2, kernelversion=3.16.0-4-amd64, operatingsystem=Debian GNU/Linux 8 (jessie), storagedriver=aufs",
            SwarmError:     nil,
            SwarmUpdated:   time.Date(2016, 2, 24, 17, 27, 29, 0, time.UTC),
        }

        if fmt.Sprintf("%#v", node) != fmt.Sprintf("%#v", node1) {
            t.Errorf("info.Nodes[1]:\n- %#v\n+ %#v\n", node, node1)
        }
    }

}
