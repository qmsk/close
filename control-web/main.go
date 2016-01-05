package main

import (
    "close/control"
    "flag"
    "net/http"
    "log"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
)

var (
    controlOptions   control.Options

    httpDevel       bool
    httpListen      string
    staticPath      string
)

func init() {
    flag.StringVar(&controlOptions.StatsReader.InfluxDB.Addr, "influxdb-addr", "http://influxdb:8086",
        "influxdb http://... address")
    flag.StringVar(&controlOptions.StatsReader.Database, "influxdb-database", stats.INFLUXDB_DATABASE,
        "influxdb database name")

    flag.StringVar(&controlOptions.Config.Redis.Addr, "config-redis-addr", "",
        "host:port")
    flag.Int64Var(&controlOptions.Config.Redis.DB, "config-redis-db", 0,
        "Database to select")
    flag.StringVar(&controlOptions.Config.Prefix, "config-prefix", "close",
        "Redis key prefix")

    flag.BoolVar(&httpDevel, "http-devel", false,
        "Development mode for HTTP")
    flag.StringVar(&httpListen, "http-listen", ":8282",
        "host:port for HTTP API")
    flag.StringVar(&staticPath, "static-path", "",
        "Path to /static files")
}

func main() {
    flag.Parse()

    manager, err := control.New(controlOptions)
    if err != nil {
        log.Fatal(err)
    }

    // run
    api := rest.NewApi()

    if httpDevel {
        api.Use(rest.DefaultDevStack...)
    }

    if app, err := manager.RestApp(); err != nil {
        log.Fatalf("manager.RestApp: %v\n", err)
    } else {
        api.SetApp(app)
    }

    staticHandler := http.FileServer(http.Dir(staticPath))

    http.Handle("/api/", http.StripPrefix("/api", api.MakeHandler()))
    http.Handle("/", staticHandler)

    if err := http.ListenAndServe(httpListen, nil); err != nil {
        log.Fatalf("http.ListenAndServe %v: %v\n", httpListen, err)
    }
}
