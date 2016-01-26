package control

import (
    "close/config"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
    "time"
)

type APIGet struct {
    Config              Config          `json:"config"`
    ConfigText          string          `json:"config_text"`

    Clients             []ClientStatus  `json:"clients"`
    Workers             []WorkerStatus  `json:"workers"`
}

func (self *Manager) Get(w rest.ResponseWriter, req *rest.Request) {
    out := APIGet{}

    if configText, err := self.DumpConfig(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Config = self.config
        out.ConfigText = configText
    }

    if listClients, err := self.ListClients(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Clients = listClients
    }

    if listWorkers, err := self.ListWorkers(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        out.Workers = listWorkers
    }

    w.WriteJson(out)
}

func (self *Manager) Post(w rest.ResponseWriter, req *rest.Request) {
    if err := self.LoadConfigReader(req.Body); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if err := self.Start(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        // TODO: redirect to GET?
        w.WriteHeader(200)
    }
}

func (self *Manager) Delete(w rest.ResponseWriter, req *rest.Request) {
    if err := self.Stop(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteHeader(200)
    }
}

func (self *Manager) GetWorker(w rest.ResponseWriter, req *rest.Request) {
    if workerStatus, err := self.WorkerGet(req.PathParam("config"), req.PathParam("instance")); workerStatus == nil {
        rest.Error(w, "Not Foud", 404)
    } else if err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(workerStatus)
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
    subFilter := config.ID{Type: req.PathParam("type")}

    if list, err := self.ConfigList(subFilter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetConfig(w rest.ResponseWriter, req *rest.Request) {
    if configID, err := config.ParseID(req.PathParam("type"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if config, err := self.ConfigGet(configID); err != nil {
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

    if configID, err := config.ParseID(req.PathParam("type"), req.PathParam("instance")); err != nil {
        rest.Error(w, err.Error(), 400)
    } else if err := self.ConfigPush(configID, configMap); err != nil {
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
func (self *Manager) GetStatsTypes(w rest.ResponseWriter, req *rest.Request) {
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
func (self *Manager) GetStatsList(w rest.ResponseWriter, req *rest.Request) {
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
type APIStats struct {
    stats.SeriesKey
    Field       string                  `json:"field"`

    Tab         *stats.SeriesTab        `json:"tab,omitempty"`
    Points      []stats.SeriesPoint     `json:"points"`
}

func (self *Manager) GetStats(w rest.ResponseWriter, req *rest.Request) {
    var fields []string
    var duration time.Duration
    var tabMap map[string]stats.SeriesTab

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

    // get summary info for one field?
    if len(fields) != 1 {
        tabMap = nil
    } else if getStats, err := self.statsReader.GetStats(seriesKey, fields[0], duration); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    } else {
        tabMap = make(map[string]stats.SeriesTab)

        for _, stats := range getStats {
            tabMap[stats.String()] = stats.SeriesTab
        }
    }

    // apply
    var list []APIStats

    if getSeries, err := self.statsReader.GetSeries(seriesKey, fields, duration); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    } else {
        for _, seriesData := range getSeries {
            apiStats := APIStats{
                SeriesKey:  seriesData.SeriesKey,
                Field:      seriesData.Field,

                Points:     seriesData.Points,
            }

            // merge in StatsTab
            if tab, exists := tabMap[seriesData.String()]; exists {
                apiStats.Tab = &tab
            }

            list = append(list, apiStats)
        }
        w.WriteJson(list)
    }
}

func (self *Manager) PostPanic(w rest.ResponseWriter, req *rest.Request) {
    if err := self.Panic(); err != nil {
        rest.Error(w, err.Error(), 500)
        return
    }

    w.Header().Add("Location", "/")
    w.WriteHeader(302)
}

func (self *Manager) RestApp() (rest.App, error) {
    return rest.MakeRouter(
        rest.Get("/",           self.Get),
        rest.Post("/",          self.Post),         // Load + Start
        rest.Delete("/",        self.Delete),       // Stop

        rest.Get("/workers/:config/:instance",  self.GetWorker),

        // list active containers
        rest.Get("/docker/", self.GetDockerList),
        rest.Get("/docker/:id", self.GetDocker),
        rest.Get("/docker/:id/logs", self.GetDockerLogs),

        // list active config items, with TTL
        rest.Get("/config/", self.GetConfigList),
        rest.Get("/config/:type", self.GetConfigList),

        // get full config
        rest.Get("/config/:type/:instance", self.GetConfig),

        // publish config change to worker
        rest.Post("/config/:type/:instance", self.PostConfig),

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

        rest.Post("/panic", self.PostPanic),
    )
}
