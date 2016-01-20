package worker

import (
    "close/config"
    "github.com/jessevdk/go-flags"
    "log"
    "os"
    "close/stats"
)

type WorkerOptions struct {
    Stats       stats.WriterOptions `group:"Stats Writer"`
    Config      config.SubOptions   `group:"Config Sub"`
}

func Main(worker Worker) {
    var options WorkerOptions

    parser := flags.NewParser(&options, flags.Default)

    workerConfig := worker.Config()

    if _, err := parser.AddGroup("Worker Options", "", workerConfig); err != nil {
        log.Fatalf("flags Parser.AddGroup %T: %v\n", workerConfig, err)
    } else {
        log.Printf("flags Parse.AddGroup %T\n", workerConfig)
    }

    if args, err := parser.Parse(); err != nil {
        os.Exit(1)
    } else if len(args) > 0 {
        log.Printf("flags Parser.Parser: extra arguments: %v\n", args)
        parser.WriteHelp(os.Stderr)
        os.Exit(1)
    }

    // config
    if options.Config.Empty() {
        log.Printf("Skip config")
    } else if configSub, err := config.NewSub(options.Config); err != nil {
        log.Fatalf("config.NewSub %v: %v\n", options.Config, err)
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
