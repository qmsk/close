package main

import (
    "close/control"
    "flag"
    "net/http"
    "log"
    "close/logs"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
)

var (
    controlOptions   control.Options

    configPath      string

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

    flag.StringVar(&configPath, "config-path", "",
        "Path to .toml config")
}

func main() {
    flag.Parse()

    logs, err := logs.New()
    if err != nil {
        log.Fatal(err)
    }

    controlOptions.Logger = logs.Logger("Manager: ")

    manager, err := control.New(controlOptions)
    if err != nil {
        log.Fatal(err)
    }

    if configPath != "" {
        if config, err := manager.LoadConfig(configPath); err != nil {
            log.Fatalf("manager.LoadConfig %v: %v\n", configPath, err)
        } else if err := manager.Start(config); err != nil {
            log.Fatalf("manager.Start %v: %v\n", config, err)
        } else {
            log.Printf("Started from %v...\n", configPath)
        }
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
    http.Handle("/logs", logs)
    http.Handle("/", staticHandler)

    if err := http.ListenAndServe(httpListen, nil); err != nil {
        log.Fatalf("http.ListenAndServe %v: %v\n", httpListen, err)
    }
}
