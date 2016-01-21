package main

import (
    "close/control"
    "github.com/jessevdk/go-flags"
    "net/http"
    "log"
    "close/logs"
    "os"
    "github.com/ant0ine/go-json-rest/rest"
)

type Options struct {
    control.Options

    ConfigPath      string      `long:"config-path" value-name:"PATH.toml"`
    Start           bool        `long:"start" description:"Start --config after loading and discovery"`

    AuthConfigPath  string      `long:"auth-config" value-name:"PATH.toml"`
    HttpDevel       bool        `long:"http-devel" description:"Use development mode"`
    HttpListen      string      `long:"http-listen" value-name:"HOST:PORT" default:":8282"`
    StaticPath      string      `long:"static-path" value-name:"PATH" description:"Path to / static files"`
}

func main() {
    var options Options

    if _, err := flags.Parse(&options); err != nil {
        os.Exit(1)
    }

    // logs
    logs, err := logs.New()
    if err != nil {
        log.Fatal(err)
    }

    // manager
    controlOptions := options.Options

    controlOptions.Logger = logs.Logger("Manager: ")

    manager, err := control.New(controlOptions)
    if err != nil {
        log.Fatal(err)
    }

    // config
    if options.ConfigPath == "" {

    } else if err := manager.LoadConfigFile(options.ConfigPath); err != nil {
        log.Fatalf("manager.LoadConfig %v: %v\n", options.ConfigPath, err)
    } else {
        log.Printf("Loaded configuration from %v...\n", options.ConfigPath)
    }

    if err := manager.Discover(); err != nil {
        log.Fatal(err)
    }

    // TODO: should happen concurrently?
    if !options.Start {

    } else if err := manager.Start(); err != nil {
        log.Fatalf("manager.Start: %v\n", err)
    } else {
        log.Printf("Started...\n")
    }

    // http API
    api := rest.NewApi()

    if options.AuthConfigPath == "" {
        log.Printf("Warning: starting without authentication\n")
    } else if auth, err := manager.NewAuth(options.AuthConfigPath); err != nil {
        log.Fatalf("manager.NewAuth %v: %v\n", options.AuthConfigPath, err)
    } else {
        api.Use(auth)
        log.Printf("Loaded users from %v...\n", options.AuthConfigPath)
    }

    if options.HttpDevel {
        api.Use(rest.DefaultDevStack...)
    }

    if app, err := manager.RestApp(); err != nil {
        log.Fatalf("manager.RestApp: %v\n", err)
    } else {
        api.SetApp(app)
    }

    staticHandler := http.FileServer(http.Dir(options.StaticPath))

    http.Handle("/api/", http.StripPrefix("/api", api.MakeHandler()))
    http.Handle("/logs", logs)
    http.Handle("/", staticHandler)

    if err := http.ListenAndServe(options.HttpListen, nil); err != nil {
        log.Fatalf("http.ListenAndServe %v: %v\n", options.HttpListen, err)
    }
}
