package control

import (
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
    "time"
)

func (self *Manager) GetWorkers(w rest.ResponseWriter, req *rest.Request) {
    if list, err := self.ConfigList(); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(list)
    }
}

func (self *Manager) GetWorker(w rest.ResponseWriter, req *rest.Request) {
    if config, err := self.ConfigGet(req.PathParam("worker")); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(config)
    }
}

func (self *Manager) PostWorker(w rest.ResponseWriter, req *rest.Request) {
    config := make(map[string]interface{})

    if err := req.DecodeJsonPayload(&config); err != nil {
        rest.Error(w, err.Error(), 400)
        return
    }

    if err := self.ConfigPush(req.PathParam("worker"), config); err != nil {
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
    // XXX: sanitize type, vulernable to InfluxQL injection...
    key := stats.SeriesKey{
        Type:       req.PathParam("type"),
        Hostname:   req.FormValue("hostname"),
        Instance:   req.FormValue("instance"),
    }
    field := req.PathParam("field")
    duration := 10 * time.Second

    if result, err := self.statsReader.GetSeries(key, field, duration); err != nil {
        rest.Error(w, err.Error(), 500)
    } else {
        w.WriteJson(result)
    }
}

func (self *Manager) RestApp() (rest.App, error) {
    return rest.MakeRouter(
        rest.Get("/workers/", self.GetWorkers),
        rest.Get("/workers/:worker", self.GetWorker),
        rest.Post("/workers/:worker", self.PostWorker),

        rest.Get("/stats", self.GetStatsTypes),
        rest.Get("/stats/", self.GetStatsList),
        rest.Get("/stats/:type", self.GetStatsList),
        rest.Get("/stats/:type/:field", self.GetStats),
    )
}
