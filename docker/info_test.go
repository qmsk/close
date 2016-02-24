package docker

import (
    "github.com/fsouza/go-dockerclient"
    "os"
    "testing"
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

}
