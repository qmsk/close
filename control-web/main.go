package main

import (
    "close/config"
    "close/control"
    "flag"
    "net/http"
    "log"
    "github.com/ant0ine/go-json-rest/rest"
    "close/stats"
)

var (
    httpDevel       bool
    httpListen      string
    statsConfig     stats.Config
    configOptions   config.Options
)

func init() {
    flag.StringVar(&statsConfig.InfluxDB.Addr, "influxdb-addr", "http://influxdb:8086",
        "influxdb http://... address")
    flag.StringVar(&statsConfig.InfluxDBDatabase, "influxdb-database", stats.INFLUXDB_DATABASE,
        "influxdb database name")

    flag.StringVar(&configOptions.Redis.Addr, "config-redis-addr", "",
        "host:port")
    flag.Int64Var(&configOptions.Redis.DB, "config-redis-db", 0,
        "Database to select")
    flag.StringVar(&configOptions.Prefix, "config-prefix", "close",
        "Redis key prefix")

    flag.BoolVar(&httpDevel, "http-devel", false,
        "Development mode for HTTP")
    flag.StringVar(&httpListen, "http-listen", ":8282",
        "host:port for HTTP API")
}

func main() {
    var manager *control.Manager

    flag.Parse()

    // config?
    if configOptions.Redis.Addr == "" {
        log.Fatalf("missing --config-redis-addr")
    } else if configRedis, err := config.NewRedis(configOptions); err != nil {
        log.Fatalf("config.NewRedis %v: %v\n", configOptions, err)
    } else {
        log.Printf("config.Redis %v\n", configRedis)

        manager = control.New(configRedis)
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

    if err := http.ListenAndServe(httpListen, api.MakeHandler()); err != nil {
        log.Fatalf("http.ListenAndServe %v: %v\n", httpListen, err)
    }
}
