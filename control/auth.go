package control

import (
    "github.com/ant0ine/go-json-rest/rest"
    "github.com/BurntSushi/toml"
)

var (
    auth authConfig
)

type user struct {
    Id       string
    Password string
}

type authConfig struct {
    Realm string
    Users []user
}

func (m* Manager) NewAuth(filePath string) (*rest.AuthBasicMiddleware, error) {
    if _, err := toml.DecodeFile(filePath, &auth); err != nil {
        return nil, err
    }

    return &rest.AuthBasicMiddleware{
        Realm: auth.Realm,
        Authenticator: authenticate,
    }, nil
}


func authenticate(givenId string, givenPassword string) bool {
    for _, user := range auth.Users {
        if user.Id == givenId {
            return user.Password == givenPassword
        }
    }
    return false
}
