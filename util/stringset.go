package util

import (
    "fmt"
    "sort"
)

type StringSet []string

func MakeStringSet(values ...string) StringSet {
    // sorted copy
    self := StringSet(make([]string, len(values)))

    copy(self, values)
    self.sort()

    return self
}

func (self *StringSet) sort() {
    sort.Strings(*self)
}

func (self *StringSet) Copy() StringSet {
    out := make([]string, len(*self))

    copy(out, *self)

    return out
}

// Replace contents
func (self *StringSet) Set(values []string) {
    *self = make([]string, len(values))

    copy(*self, values)
    self.sort()
}

func (self *StringSet) Add(value string) {
    *self = append(*self, value)
    self.sort()
}

func (self *StringSet) AddEnv(name string, value interface{}) {
    env := fmt.Sprintf("%s=%v", name, value)

    *self = append(*self, env)
    self.sort()
}

func (self StringSet) Equals(other StringSet) bool {
    if len(self) != len(other) {
        return false
    }
    for i, value := range self {
        if value != other[i] {
            return false
        }
    }
    return true
}

// Returns true if env is a subset of the other env.
// i.e. other contains all the strings in env.
func (self StringSet) Subset(other StringSet) bool {
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

// TODO: just plain extend, with duplicates; implemenmt .Union()?
func (self StringSet) Extend(other StringSet) (out StringSet) {
    out = append(out, self...)
    out = append(out, other...)

    return
}
