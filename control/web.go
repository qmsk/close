package control

import (
    "close/config"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
    "time"
    "github.com/BurntSushi/toml"
)

type APIWorkers struct {
    Config              string          `json:"config"`
    WorkerConfig        *WorkerConfig   `json:"worker_config"`
    Workers             []WorkerStatus  `json:"workers"`
}

func (self *Manager) GetWorkers(w rest.ResponseWriter, req *rest.Request) {
    out := APIWorkers{}

    if workerConfig, err := self.WorkerConfig(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Config = workerConfig
        out.WorkerConfig = self.workerConfig
    }

    if listWorkers, err := self.ListWorkers(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Workers = listWorkers
    }

    w.WriteJson(out)
}

func (self *Manager) PostWorkers(w rest.ResponseWriter, req *rest.Request) {
    var workerConfig WorkerConfig

    if _, err := toml.DecodeReader(req.Body, &workerConfig); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if err := self.StartWorkers(workerConfig); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        // TODO: redirect to GET?
        w.WriteHeader(200)
    }
}

func (self *Manager) DeleteWorkers(w rest.ResponseWriter, req *rest.Request) {
    if err := self.StopWorkers(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *Manager) GetDockerList(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.DockerList(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetDocker(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.DockerGet(req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else if list == nil {
        rest.Error(w, "Not Found", 404)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetDockerLogs(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.DockerLogs(req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetConfigList(w rest.ResponseWriter, req *rest.Request) {
    subFilter := config.SubOptions{Type: req.PathParam("type")}

    if list, err := self.ConfigList(subFilter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetConfig(w rest.ResponseWriter, req *rest.Request) {
    if subOptions, err := config.ParseSub(req.PathParam("type"), req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if config, err := self.ConfigGet(subOptions); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(config)
    }
}

func (self *Manager) PostConfig(w rest.ResponseWriter, req *rest.Request) {
    configMap := make(config.ConfigMap)

    if err := req.DecodeJsonPayload(&configMap); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if subOptions, err := config.ParseSub(req.PathParam("type"), req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if err := self.ConfigPush(subOptions, configMap); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        // TODO: redirect to GET?
        w.WriteHeader(200)
    }
}

/*
 * Query metadata about available stats types.
 *
    [
      {
        "type": "icmp_latency",
        "fields": [
          "rtt"
        ]
      }
    ]
 */
func (self *Manager) GetStatsTypes(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.statsReader.ListTypes(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

/*
 * Query available stast series, for a given type.
 *
[
  {
    "type": "udp_recv",
    "hostname": "catcp-terom-dev",
    "instance": "127.0.0.1:1337"
  },
]
 */
func (self *Manager) GetStatsList(w rest.ResponseWriter, req *rest.Request) {
    // XXX: sanitize type, vulernable to InfluxQL injection...
    filter := stats.SeriesKey{
        Type:       req.PathParam("type"),
        Hostname:   req.FormValue("hostname"),
        Instance:   req.FormValue("instance"),
    }

    if list, err := self.statsReader.ListSeries(filter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetStats(w rest.ResponseWriter, req *rest.Request) {
    var fields []string
    var duration time.Duration

    if req.PathParam("field") != "" {
        // TODO: figureout some syntax
        fields = []string{req.PathParam("field")}
    }

    // XXX: sanitize type, vulernable to InfluxQL injection...
    seriesKey := stats.SeriesKey{
        Type:       req.PathParam("type"),
        Hostname:   req.FormValue("hostname"),
        Instance:   req.FormValue("instance"),
    }

    if req.FormValue("duration") == "" {
        duration = 10 * time.Second
    } else if parseDuration, err := time.ParseDuration(req.FormValue("duration")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else {
        duration = parseDuration
    }

    // apply
    if result, err := self.statsReader.GetSeries(seriesKey, fields, duration); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(result)
    }
}

func (self *Manager) Panic(w rest.ResponseWriter, req *rest.Request) {
    if err := self.PanicWorkers(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    }

    w.Header().Add("Location", "/")
    w.WriteHeader(302)
}

func (self *Manager) RestApp() (rest.App, error) {
    return rest.MakeRouter(
        rest.Get("/workers", self.GetWorkers),
        rest.Post("/workers", self.PostWorkers),
        rest.Delete("/workers", self.DeleteWorkers),

        // list active containers
        rest.Get("/docker/", self.GetDockerList),
        rest.Get("/docker/:id", self.GetDocker),
        rest.Get("/docker/:id/logs", self.GetDockerLogs),

        // list active config items, with TTL
        rest.Get("/config/", self.GetConfigList),
        rest.Get("/config/:type", self.GetConfigList),

        // get full config
        rest.Get("/config/:type/:id", self.GetConfig),

        // publish config change to worker
        rest.Post("/config/:type/:id", self.PostConfig),

        // static information about available stats types/fields
        rest.Get("/stats", self.GetStatsTypes),

        // dynamic information about avilable stats series (hostname/instance)
        rest.Get("/stats/", self.GetStatsList),

        // ..filtered by type
        rest.Get("/stats/:type", self.GetStatsList),

        // data type's fields
        // may include multiple series, filtered by ?hostname=&instance=
        rest.Get("/stats/:type/", self.GetStats),

        // data for type's specific field
        // may include multiple series, filtered by ?hostname=&instance=
        rest.Get("/stats/:type/:field", self.GetStats),

        rest.Post("/panic", self.Panic),
    )
}
