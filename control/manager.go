package control

import (
    "bytes"
    "close/config"
    "close/docker"
    "fmt"
    "io"
    "log"
    "os"
    "close/stats"
    "strings"
    "github.com/BurntSushi/toml"
)

type Options struct {
    Stats           stats.ReaderOptions `group:"Stats Reader"`
    Config          config.Options      `group:"Config"`
    Docker          docker.Options      `group:"Docker"`

    Logger          *log.Logger
}

// full-system configuration
type Config struct {
    Clients         map[string]*ClientConfig
    Workers         map[string]*WorkerConfig
}

type Manager struct {
    options         Options
    log             *log.Logger

    configRedis     *config.Redis
    statsReader     *stats.Reader
    docker          *docker.Manager

    // state
    // XXX: these are unsafe against concurrent web requests
    config          Config
    clients         map[string]*Client
    workers         map[string]*Worker
}

func New(options Options) (*Manager, error) {
    if options.Logger == nil {
        options.Logger = log.New(os.Stderr, "Manager: ", 0)
    }

    self := &Manager{
        options:    options,
        log:        options.Logger,

        clients:        make(map[string]*Client),
        workers:        make(map[string]*Worker),
    }


    if err := self.init(options); err != nil {
        return nil, err
    }

    return self, nil
}

func (self *Manager) init(options Options) error {
    if options.Config.Empty() {
        return fmt.Errorf("Missing --config options")
    } else if configRedis, err := config.NewRedis(options.Config); err != nil {
        return fmt.Errorf("config.NewRedis %v: %v", options.Config, err)
    } else {
        self.configRedis = configRedis
    }

    if options.Stats.Empty() {
        return fmt.Errorf("Missing --stats options")
    } else if statsReader, err := stats.NewReader(options.Stats); err != nil {
        return fmt.Errorf("stats.NewReader %v: %v", options.Stats, err)
    } else {
        self.statsReader = statsReader
    }

    if dockerManager, err := docker.NewManager(options.Docker); err != nil {
        return fmt.Errorf("docker.NewManager %v: %v", options.Docker, err)
    } else {
        self.docker = dockerManager
    }

    return nil
}

// Activate the given config
func (self *Manager) loadConfig(meta toml.MetaData, config Config) (err error) {
    var undecodedKeys []string

    for _, key := range meta.Undecoded() {
        undecodedKeys = append(undecodedKeys, key.String())
    }

    if undecodedKeys != nil {
        return fmt.Errorf("Undecoded keys: %v", strings.Join(undecodedKeys, " "))
    }

    // load
    for clientName, clientConfig := range config.Clients {
        clientConfig.name = clientName

        self.log.Printf("loadConfig: client %#v", clientConfig)
    }

    for workerName, workerConfig := range config.Workers {
        workerConfig.name = workerName

        self.log.Printf("loadConfig: worker %#v", workerConfig)
    }

    // TODO: stop old config?
    self.config = config

    return nil
}

func (self *Manager) LoadConfigReader(reader io.Reader) error {
    var config Config

    if meta, err := toml.DecodeReader(reader, &config); err != nil {
        return err
    } else {
        return self.loadConfig(meta, config)
    }
}

func (self *Manager) LoadConfigFile(filePath string) error {
    var config Config

    if meta, err := toml.DecodeFile(filePath, &config); err != nil {
        return err
    } else {
        return self.loadConfig(meta, config)
    }
}

func (self *Manager) LoadConfigString(data string) error {
    var config Config

    if meta, err := toml.Decode(data, &config); err != nil {
        return err
    } else {
        return self.loadConfig(meta, config)
    }
}

// Get running configuration
func (self *Manager) DumpConfig() (string, error) {
    var buf bytes.Buffer

    if err := toml.NewEncoder(&buf).Encode(self.config); err != nil {
        return "", err
    }

    return buf.String(), nil
}

// Discover any existing running docker containers before initial Start()
// Must be run after loadConfig() to recognize any containers..
// Allows Start() to re-use existing containers, and cleanup undesired containers
func (self *Manager) Discover() (err error) {
    if dockerContainers, err := self.docker.List(); err != nil {
        return fmt.Errorf("DockerList: %v", err)
    } else {
        for _, dockerContainer := range dockerContainers {
            switch dockerContainer.Class {
            case "client":
                if client, err := self.discoverClient(dockerContainer); err != nil {
                    self.log.Printf("discoverClient %v: %v", dockerContainer, err)
                } else {
                    self.log.Printf("Discover %v: client %v", dockerContainer, client)

                    self.clients[client.String()] = client
                }
            case "worker":
                if worker, err := self.discoverWorker(dockerContainer); err != nil {
                    self.log.Printf("discoverWorker %v: %v", dockerContainer, err)
                } else {
                    self.log.Printf("Discover %v: worker %v", dockerContainer, worker)

                    self.workers[worker.String()] = worker
                }
            default:
                self.log.Printf("Discover %v: ignore unknown class: %v", dockerContainer, dockerContainer.Class)
            }
        }

        return nil
    }
}

// Start new configuration
func (self *Manager) Start() error {
    self.log.Printf("Start config...\n");

    // reconfigure clients
    self.markClients()
    for _, clientConfig := range self.config.Clients {
        if err := self.ClientUp(clientConfig); err != nil {
            return err
        }
    }

    // cleanup any unconfigured clients
    if err := self.ClientDown(nil); err != nil {
        return err
    }

    // reconfigure workers
    self.markWorkers()
    for _, workerConfig := range self.config.Workers {
        if err := self.WorkerUp(workerConfig); err != nil {
            return err
        }
    }

    // cleanup any unconfigured workers
    if err := self.WorkerDown(nil); err != nil {
        return err
    }

    self.log.Printf("Started\n");

    return nil
}

// Stop current configuration
func (self *Manager) Stop() (err error) {
    self.markWorkers()
    if workersErr := self.WorkerDown(nil); workersErr != nil {
        err = workersErr
    }

    self.markClients()
    if clientsErr:= self.ClientDown(nil); clientsErr != nil {
        err = clientsErr
    }

    return err
}

// Kill any running containers and reset state
func (self *Manager) Panic() (error) {
    self.log.Printf("Panic!\n");

    err := self.docker.Panic()

    self.clients = make(map[string]*Client)
    self.workers = make(map[string]*Worker)

    return err
}
