package control

import (
    "github.com/ant0ine/go-json-rest/rest"
)

func (self *Manager) GetWorkers(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.ConfigList(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) RestApp() (rest.App, error) {
    return rest.MakeRouter(
        rest.Get("/workers", self.GetWorkers),
    )
}
