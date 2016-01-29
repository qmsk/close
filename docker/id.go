package docker

import (
    "fmt"
    "strings"
)

type ID struct {
    Class       string          `json:"class"`
    Type        string          `json:"type"`
    Instance    string          `json:"instance"`
}

// Docker name
func (self ID) String() string {
    return fmt.Sprintf("close-%s-%s-%s", self.Class, self.Type, self.Instance)
}

func (self ID) labels() map[string]string {
    return map[string]string{
        "close":            self.Class,
        "close.type":       self.Type,
        "close.instance":   self.Instance,
    }
}

func (self *ID) parseID(name string, labels map[string]string) error {
    // docker swarm "/node/name"
    // docker "/name"
    namePath := strings.Split(name, "/")
    name = namePath[len(namePath) - 1]

    if class := labels["close"]; class== "" {
        return fmt.Errorf("missing close.class=")
    } else {
        self.Class = class
    }

    if typ := labels["close.type"]; typ == "" {
        return fmt.Errorf("missing close.typ=")
    } else {
        self.Type = typ
    }

    if instance := labels["close.instance"]; instance == "" {
        return fmt.Errorf("missing close.instance=")
    } else {
        self.Instance = instance
    }

    if name != self.String() {
        return fmt.Errorf("name mismatch %v: %v", name, self)
    }

    return nil
}


