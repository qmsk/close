package worker

import (
    "close/config"
    "github.com/jessevdk/go-flags"
    "log"
    "os"
    "close/stats"
)

type Options struct {
    Stats       stats.WriterOptions `group:"Stats Writer"`
    Config      config.SubOptions   `group:"Config Sub"`

    workers         map[string]WorkerConfig
    workerType      string
    workerConfig    WorkerConfig
}

func (options *Options) Register(name string, workerConfig WorkerConfig) {
    if options.workers == nil {
        options.workers = make(map[string]WorkerConfig)
    }

    options.workers[name] = workerConfig
}

func (options *Options) Parse() {
    parser := flags.NewParser(options, flags.Default)

    for workerName, workerConfig := range options.workers {
        if _, err := parser.AddCommand(workerName, "", "", workerConfig); err != nil {
            panic(err)
        }
    }

    if args, err := parser.Parse(); err != nil {
        os.Exit(1)
    } else if len(args) > 0 {
        log.Printf("flags Parser.Parser: extra arguments: %v\n", args)
        parser.WriteHelp(os.Stderr)
        os.Exit(1)
    }

    // worker command
    if command := parser.Active; command == nil {
        log.Fatalf("No command given\n")
    } else if workerConfig, found := options.workers[command.Name]; !found {
        log.Fatalf("Invalid command: %v\n", command)
    } else {
        log.Printf("Parse worker: %v\n", workerConfig)

        options.workerType = command.Name
        options.workerConfig = workerConfig
    }
}

func Main(options Options) {
    worker, err := options.workerConfig.Worker()
    if err != nil {
        log.Fatalf("%T: Apply: %v\n", options.workerConfig, err)
    } else {
        log.Printf("%T: Apply: %v\n", options.workerConfig, worker)
    }

    // config
    if options.Config.Empty() {
        log.Printf("Skip config")
    } else if configRedis, err := config.NewRedis(options.Config.Options); err != nil {
        log.Fatalf("config.NewRedis %v: %v\n", options.Config, err)
    } else if configSub, err := configRedis.NewSub(options.workerType, options.Config.Instance); err != nil {
        log.Fatalf("config.Redis %v: NewSub %v %v: %v\n", configRedis, options.workerType, options.Config.Instance, err)
    } else if err := worker.ConfigSub(configSub); err != nil {
        log.Fatalf("Worker %v: ConfigSub %v: %v\n", worker, configSub, err)
    } else {
        log.Printf("Worker %v: ConfigSub %v\n", worker, configSub)
    }

    // stats
    if options.Stats.Empty() {
        log.Printf("Skip stats")
    } else if statsWriter, err := stats.NewWriter(options.Stats); err != nil {
        log.Fatalf("stats.NewWriter %v: %v\n", options.Stats, err)
    } else if err := worker.StatsWriter(statsWriter); err != nil {
        log.Fatalf("Worker %v: StatsWriter %v: %v\n", worker, statsWriter, err)
    } else {
        log.Printf("Worker %v: StatsWriter %v\n", worker, statsWriter)
    }

    // run
    go stopping(worker)

    log.Printf("Run %T: %v\n", worker, worker)

    if err := worker.Run(); err != nil {
        log.Fatalf("%T %v: Run: %v\n", worker, worker, err)
    } else {
        log.Printf("%T %v done\n", worker, worker)
    }
}