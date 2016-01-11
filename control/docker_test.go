package control

import (
    "testing"
)

func TestDockerConfigEquals(t *testing.T) {
    config1 := DockerConfig{Image: "test", Command: "test", Args: []string{"-test"}, Env: []string{"TEST=test"}}
    config2 := config1

    if !config1.Equals(config1) {
        t.Errorf("equal: %#v %#v", config1, config1)
    }

    config2 = config1
    config2.Image = "test:2"
    if config1.Equals(config2) {
        t.Errorf("non-equal Image: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Command = "test:2"
    if config1.Equals(config2) {
        t.Errorf("non-equal Command: %#v %#v", config1, config2)
    }

    // args
    config2 = config1
    config2.Args = []string{}
    if config1.Equals(config2) {
        t.Errorf("non-equal Args: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Args = []string{"-test2"}
    if config1.Equals(config2) {
        t.Errorf("non-equal Args[0]: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Args = []string{"-test", "-test2"}
    if config1.Equals(config2) {
        t.Errorf("non-equal Args: %#v %#v", config1, config2)
    }

    // env
    config2 = config1
    config2.Env = []string{}
    if config1.Equals(config2) {
        t.Errorf("non-equal Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST=test2"}
    if config1.Equals(config2) {
        t.Errorf("non-equal Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST=test", "TEST2=test2"}
    if !config1.Equals(config2) {
        t.Errorf("superset Env: %#v %#v", config1, config2)
    }
}
