package config

import (
    "fmt"
    "regexp"
)

var idRegexp = regexp.MustCompile(`^([a-zA-Z0-9_]+)/([a-zA-Z0-9_:-]+)$`)

type ID struct {
    Type        string  `json:"type"`
    Instance    string  `json:"instance"`
}

func (self ID) String() string {
    return fmt.Sprintf("%s/%s", self.Type, self.Instance)
}

func (self ID) Valid() bool {
    return idRegexp.MatchString(self.String())
}

// Return error if not Valid()
func (self ID) Check() error {
    if !self.Valid() {
        return fmt.Errorf("Invalid ID: %#v", self)
    }

    return nil
}

func ParseID(subType string, instance string) (ID, error) {
    id := ID{subType, instance}

    if err := id.Check(); err != nil {
        return id, err
    }

    return id, nil
}


