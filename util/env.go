package util

import (
    "fmt"
    "sort"
)

type Env []string

func MakeEnv(values []string) Env {
    // sorted copy
    self := Env(make([]string, len(values)))

    copy(self, values)
    self.sort()

    return self
}

func (self *Env) sort() {
    sort.Strings(*self)
}

func (self *Env) Add(name string, value interface{}) {
    env := fmt.Sprintf("%s=%v", name, value)

    *self = append(*self, env)
    self.sort()
}

// Returns true if env is a subset of the other env.
// i.e. other contains all the strings in env.
func (self Env) Subset(other Env) bool {
    if len(self) > len(other) {
        return false
    }
    for i, j := 0, 0; i < len(self) && j < len(other); j++ {
        if self[i] > other[j] {
            continue
        } else if self[i] < other[j] {
            return false
        } else {
            i++
        }
    }

    return true
}
