package control

import (
    "close/config"
    "close/docker"
    "fmt"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
    "time"
)

type WebApp struct {
    manager       ManagerAPI
    statsReader   *stats.Reader
    docker        *docker.Manager
    config        *Config
}

type JsonApp interface {
    RestApp() (rest.App, error)
}

type APIGet struct {
    Config              Config          `json:"config"`
    ConfigText          string          `json:"config_text"`

    Clients             []ClientStatus  `json:"clients"`
    Workers             []WorkerStatus  `json:"workers"`
}

func (self *WebApp) Get(w rest.ResponseWriter, req *rest.Request) {
    out := APIGet{}

    if configText, err := self.manager.DumpConfig(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Config = *self.config
        out.ConfigText = configText
    }

    if listClients, err := self.manager.ListClients(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Clients = listClients
    }

    if listWorkers, err := self.manager.ListWorkers(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Workers = listWorkers
    }

    w.WriteJson(out)
}

func (self *WebApp) Post(w rest.ResponseWriter, req *rest.Request) {
    if err := self.manager.LoadConfigReader(req.Body); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if errs := self.manager.Start(); errs != nil {
        rest.Error(w, fmt.Sprintf("%v", errs), 500)
        return
    } else {
        // TODO: redirect to GET?
        w.WriteHeader(200)
    }
}

func (self *WebApp) PostStop(w rest.ResponseWriter, req *rest.Request) {
    if errs := self.manager.Stop(); errs != nil {
        rest.Error(w, fmt.Sprintf("%v", errs), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *WebApp) PostClean(w rest.ResponseWriter, req *rest.Request) {
    if errs := self.manager.Clean(); errs != nil {
        rest.Error(w, fmt.Sprintf("%v", errs), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *WebApp) Delete(w rest.ResponseWriter, req *rest.Request) {
    if errs := self.manager.Stop(); errs != nil {
        rest.Error(w, fmt.Sprintf("%v", errs), 500)
        return
    }
    if errs := self.manager.Clean(); errs != nil {
        rest.Error(w, fmt.Sprintf("%v", errs), 500)
        return
    }

    w.WriteHeader(200)
}

func (self *WebApp) GetWorker(w rest.ResponseWriter, req *rest.Request) {
    if workerStatus, err := self.manager.WorkerGet(req.PathParam("config"), req.PathParam("instance")); workerStatus == nil {
        rest.Error(w, "Not Found", 404)
    } else if err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(workerStatus)
    }
}

func (self *WebApp) DeleteWorkers(w rest.ResponseWriter, req *rest.Request) {
    if err := self.manager.WorkerDelete(req.PathParam("config"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *WebApp) DeleteClients(w rest.ResponseWriter, req *rest.Request) {
    if err := self.manager.ClientDelete(req.PathParam("config"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *WebApp) GetDockerList(w rest.ResponseWriter, req *rest.Request) {
    filter := docker.ID{}

    if list, err := self.docker.List(filter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *WebApp) GetDocker(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.docker.Get(req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else if list == nil {
        rest.Error(w, "Not Found", 404)
    } else {
        w.WriteJson(list)
    }
}

func (self *WebApp) GetDockerLogs(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.docker.Logs(req.PathParam("id")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *WebApp) GetConfigList(w rest.ResponseWriter, req *rest.Request) {
    subFilter := config.ID{Type: req.PathParam("type")}

    if list, err := self.manager.ConfigList(subFilter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *WebApp) GetConfig(w rest.ResponseWriter, req *rest.Request) {
    if configID, err := config.ParseID(req.PathParam("type"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if config, err := self.manager.ConfigGet(configID); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(config)
    }
}

func (self *WebApp) PostConfig(w rest.ResponseWriter, req *rest.Request) {
    configMap := make(config.ConfigMap)

    if err := req.DecodeJsonPayload(&configMap); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if configID, err := config.ParseID(req.PathParam("type"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if err := self.manager.ConfigPush(configID, configMap); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        // TODO: redirect to GET?
        w.WriteHeader(200)
    }
}

/*
 * Query a list of available stats types (InfluxDB measurements and their fields).
 *
 * This information is static, it only changes if the code changes to introduce new types/fields (or old measurememts are dropped).
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
func (self *WebApp) GetStatsTypes(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.statsReader.ListTypes(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

/*
 * Query a list of stats series, optionally for a given type (InfluxDB series (tag-sets), for a given measurement).
 *
 * This information is dynamic, it changes if new workers are started.
 *
 * TODO: cleanup/hide old series that are no longer active, i.e. expire after some time?
[
  {
    "type": "udp_recv",
    "hostname": "catcp-terom-dev",
    "instance": "127.0.0.1:1337"
  },
]
 */
func (self *WebApp) GetStatsList(w rest.ResponseWriter, req *rest.Request) {
    // XXX: sanitize type, vulernable to InfluxQL injection...
    filter := stats.SeriesKey{
        Type:       req.PathParam("type"),      /* Optional */
        Hostname:   req.FormValue("hostname"),
        Instance:   req.FormValue("instance"),
    }

    if list, err := self.statsReader.ListSeries(filter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

/*
 * Query stats series for data points, for either a given field or all fields.
 *
 * Each field is returned separately.
 *
 * This information is temporal, it changes continuously for active series.
[
  {
    "type": "udp_send",
    "hostname": "close-client-openvpn-1",
    "instance": "15042208547977655843",
    "field": "rate",
    "points": [
      {
        "time": "2016-01-26T10:01:12.108997292Z",
        "value": 10.00012210149086
      }
    ]
  }
]
 */
func (self *WebApp) GetStats(w rest.ResponseWriter, req *rest.Request) {
    var fields []string
    var duration time.Duration

    if req.PathParam("field") != "" {
        // TODO: figure out some syntax for multiple fields?
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
    if statsSeries, err := self.statsReader.GetSeries(seriesKey, fields, duration); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(statsSeries)
    }
}

func (self *WebApp) PostPanic(w rest.ResponseWriter, req *rest.Request) {
    if err := self.manager.Panic(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    }

    w.Header().Add("Location", "/")
    w.WriteHeader(302)
}

func (self *Manager) RestApp() (rest.App, error) {
    app := &WebApp {
        manager:      self,
        statsReader:  self.statsReader,
        docker:       self.docker,
        config:       &self.config,
    }
    return rest.MakeRouter(
        rest.Get("/",           app.Get),
        rest.Post("/",          app.Post),         // Load + Start
        rest.Post("/stop",      app.PostStop),
        rest.Post("/clean",     app.PostClean),
        rest.Delete("/",        app.Delete),       // Stop + Clean

        // Clients
        rest.Delete("/clients/",                    app.DeleteClients),
        rest.Delete("/clients/:config/",            app.DeleteClients),
        rest.Delete("/clients/:config/:instance",   app.DeleteClients),


        rest.Get("/workers/:config/:instance",      app.GetWorker),
        rest.Delete("/workers/",                    app.DeleteWorkers),
        rest.Delete("/workers/:config/",            app.DeleteWorkers),
        rest.Delete("/workers/:config/:instance",   app.DeleteWorkers),


        // list active containers
        rest.Get("/docker/", app.GetDockerList),
        rest.Get("/docker/:id", app.GetDocker),
        rest.Get("/docker/:id/logs", app.GetDockerLogs),

        // list active config items, with TTL
        rest.Get("/config/", app.GetConfigList),
        rest.Get("/config/:type", app.GetConfigList),

        // get full config
        rest.Get("/config/:type/:instance", app.GetConfig),

        // publish config change to worker
        rest.Post("/config/:type/:instance", app.PostConfig),

        // static information about available stats types/fields
        rest.Get("/stats", app.GetStatsTypes),

        // dynamic information about avilable stats series (hostname/instance)
        rest.Get("/stats/", app.GetStatsList),

        // ..filtered by type
        rest.Get("/stats/:type", app.GetStatsList),

        // data type's fields
        // may include multiple series, filtered by ?hostname=&instance=
        rest.Get("/stats/:type/", app.GetStats),

        // data for type's specific field
        // may include multiple series, filtered by ?hostname=&instance=
        rest.Get("/stats/:type/:field", app.GetStats),

        rest.Post("/panic", app.PostPanic),
    )
}
