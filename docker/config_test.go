package docker

import (
    "testing"
)

func TestConfigEquals(t *testing.T) {
    config0 := Config{Image: "test"}
    config1 := Config{Image: "test", Command: "test", Args: []string{"-test"}, Env: []string{"TEST1=test", "TEST2=test"}}
    config2 := config1

    // self-equality
    if !config1.Equals(config1) {
        t.Errorf("equal: %#v %#v", config1, config1)
    }

    // default command
    if !config0.Equals(config1) {
        t.Errorf("non-equal empty Command: %#v %#v", config0, config1)
    }
    if !config0.Equals(config2) {
        t.Errorf("non-equal empty Command: %#v %#v", config0, config2)
    }

    // image/command
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
        t.Errorf("non-equal empty Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST1=test2"}
    if config1.Equals(config2) {
        t.Errorf("non-equal short Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST1=test", "TEST2=test2"}
    if config1.Equals(config2) {
        t.Errorf("non-equal modified Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST1=test", "TEST2=test", "TEST3=test"}
    if !config1.Equals(config2) {
        t.Errorf("equal superset Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST1=test", "TEST1b=test", "TEST2=test"}
    if !config1.Equals(config2) {
        t.Errorf("equal superset Env: %#v %#v", config1, config2)
    }

    config2 = config1
    config2.Env = []string{"TEST1=test", "TEST2=test2", "TEST3=test"}
    if config1.Equals(config2) {
        t.Errorf("non-equal modified superset Env: %#v %#v", config1, config2)
    }

}
