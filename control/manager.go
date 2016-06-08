package control

import (
    "github.com/qmsk/close/config"
    "github.com/qmsk/close/docker"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "github.com/qmsk/close/stats"
    "strings"
    "github.com/BurntSushi/toml"
)

type ManagerAPI interface {
    Start() []error
    Stop() []error
    Clean() []error
    DumpConfig() (string, error)
    ConfigList(config.ID) ([]ConfigItem, error)
    ConfigGet(config.ID) (config.Config, error)
    ConfigPush(config.ID, config.Config) error
    ListClients() ([]ClientStatus, error)
    ListWorkers() ([]WorkerStatus, error)
    WorkerGet(string, string) (*WorkerStatus, error)
    WorkerDelete(string, string) error
    ClientDelete(string, string) error
    LoadConfigReader(io.Reader) error
    Panic() (error)
}

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

    /* State */
    // XXX: these are unsafe against concurrent web requests
    configText      string
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

// Load a TOML-formatted config.
// Call Start() to activate it. Possibly with Discover() in between when first starting up from an unkonwn state.
func (self *Manager) loadConfig(text string) (err error) {
    var config Config

    meta, err := toml.Decode(text, &config)
    if err != nil {
        return err
    }

    // check meta
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
    self.configText = text
    self.config = config

    return nil
}

func (self *Manager) LoadConfigReader(reader io.Reader) error {
    if data, err := ioutil.ReadAll(reader); err != nil {
        return err
    } else {
        return self.loadConfig(string(data))
    }
}

func (self *Manager) LoadConfigFile(filePath string) error {
    if data, err := ioutil.ReadFile(filePath); err != nil {
        return err
    } else {
        return self.loadConfig(string(data))
    }
}

func (self *Manager) LoadConfigString(data string) error {
    return self.loadConfig(data)
}

// Get running configuration
func (self *Manager) DumpConfig() (string, error) {
    return self.configText, nil
}

// Discover any existing running docker containers before initial Start()
// Must be run after loadConfig() to recognize any containers..
// Allows Start() to re-use existing containers, and cleanup undesired containers
func (self *Manager) Discover() (err error) {
    if dockerContainers, err := self.docker.List(docker.ID{}); err != nil {
        return fmt.Errorf("DockerList: %v", err)
    } else {
        for _, containerStatus := range dockerContainers {
            switch containerStatus.Class {
            case "client":
                if client, err := self.discoverClient(containerStatus.ID); err != nil {
                    self.log.Printf("discoverClient %v: %v", containerStatus, err)
                } else {
                    self.log.Printf("Discover %v: client %v", containerStatus, client)

                    self.clients[client.String()] = client
                }
            case "worker":
                if worker, err := self.discoverWorker(containerStatus.ID); err != nil {
                    self.log.Printf("discoverWorker %v: %v", containerStatus, err)
                } else {
                    self.log.Printf("Discover %v: worker %v", containerStatus, worker)

                    self.workers[worker.String()] = worker
                }
            default:
                self.log.Printf("Discover %v: ignore unknown class: %v", containerStatus, containerStatus.Class)
            }
        }

        return nil
    }
}

// Start new configuration
func (self *Manager) Start() (errs []error) {
    self.log.Printf("Start config...\n");

    // reconfigure clients
    self.markClients()
    for _, clientConfig := range self.config.Clients {
        if upErrs := self.ClientUp(clientConfig); upErrs != nil {
            errs = append(errs, upErrs...)
        }
    }
    if clientErrs := self.sweepClients(); clientErrs != nil {
        errs = append(errs, clientErrs...)
    }

    // reconfigure workers
    self.markWorkers()
    for _, workerConfig := range self.config.Workers {
        if upErrs := self.WorkerUp(workerConfig); upErrs != nil {
            errs = append(errs, upErrs...)
        }
    }
    if workerErrs := self.sweepWorkers(); workerErrs != nil {
        errs = append(errs, workerErrs...)
    }

    self.log.Printf("Started: %d errors\n", len(errs));

    return errs
}

// Stop current configuration
func (self *Manager) Stop() (errs []error) {
    if workersErrs := self.WorkerDown(nil); workersErrs != nil {
        errs = append(errs, workersErrs...)
    }

    if clientsErrs := self.ClientDown(nil); clientsErrs != nil {
        errs = append(errs, clientsErrs...)
    }

    return errs
}

// Clean unused workers
func (self *Manager) Clean() (errs []error) {
    if workersErrs := self.WorkerClean(); workersErrs != nil {
        errs = append(errs, workersErrs...)
    }

    if clientsErrs := self.ClientClean(); clientsErrs != nil {
        errs = append(errs, clientsErrs...)
    }

    return errs
}

// Kill any running containers and reset state
func (self *Manager) Panic() (error) {
    self.log.Printf("Panic!\n");

    err := self.docker.Panic()

    return err
}
