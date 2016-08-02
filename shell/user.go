package shell

import (
    "github.com/BurntSushi/toml"
)

type User struct {
    Id       string   `short:"l" long:"login" description:"login username" default:"admin"`
    Password string   `short:"p" long:"password" description:"login password" required:"true"`
}

func NewUser(filePath string) (*User, error) {
	user := &User{}
    if _, err := toml.DecodeFile(filePath, user); err != nil {
        return nil, err
    }
	return user, nil
}
