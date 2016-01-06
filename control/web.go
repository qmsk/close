package control

import (
    "close/config"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
    "time"
)

func (self *Manager) GetConfigList(w rest.ResponseWriter, req *rest.Request) {
    subFilter := config.SubOptions{Module: req.PathParam("module")}

    if list, err := self.ConfigList(subFilter); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetConfig(w rest.ResponseWriter, req *rest.Request) {
    subOptions := config.SubOptions{Module: req.PathParam("module"), ID: req.PathParam("id")}

    if config, err := self.ConfigGet(subOptions); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(config)
    }
}

func (self *Manager) PostConfig(w rest.ResponseWriter, req *rest.Request) {
    subOptions := config.SubOptions{Module: req.PathParam("module"), ID: req.PathParam("id")}
    config := make(map[string]interface{})

    if err := req.DecodeJsonPayload(&config); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if err := self.ConfigPush(subOptions, config); err != nil {
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

func (self *Manager) RestApp() (rest.App, error) {
    return rest.MakeRouter(
        rest.Get("/config/", self.GetConfigList),
        rest.Get("/config/:module", self.GetConfigList),
        rest.Get("/config/:module/:id", self.GetConfig),
        rest.Post("/config/:module/:id", self.PostConfig),

        rest.Get("/stats", self.GetStatsTypes),
        rest.Get("/stats/", self.GetStatsList),
        rest.Get("/stats/:type", self.GetStatsList),
        rest.Get("/stats/:type/", self.GetStats),
        rest.Get("/stats/:type/:field", self.GetStats),
    )
}
