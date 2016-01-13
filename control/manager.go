package control

import (
    "bytes"
    "close/config"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "io"
    "log"
    "os"
    "close/stats"
    "github.com/BurntSushi/toml"
)

type Options struct {
    StatsReader     stats.ReaderConfig
    Config          config.Options
    DockerEndpoint  string

    Logger          *log.Logger
}

// full-system configuration
type Config struct {
    Client          *ClientConfig
    Worker          *WorkerConfig
}

type Manager struct {
    options         Options
    log             *log.Logger

    configRedis     *config.Redis
    statsReader     *stats.Reader
    dockerClient    *docker.Client
    dockerName      string

    // state
    // XXX: these are unsafe against concurrent web requests
    config          *Config
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
    if options.Config.Redis.Addr == "" {
        return fmt.Errorf("missing --config-redis-addr")
    } else if configRedis, err := config.NewRedis(options.Config); err != nil {
        return fmt.Errorf("config.NewRedis %v: %v", options.Config, err)
    } else {
        self.configRedis = configRedis
    }

    if statsReader, err := stats.NewReader(options.StatsReader); err != nil {
        return fmt.Errorf("stats.NewReader %v: %v", options.StatsReader, err)
    } else {
        self.statsReader = statsReader
    }

    if options.DockerEndpoint == "" {
        if dockerClient, err := docker.NewClientFromEnv(); err != nil {
            return fmt.Errorf("docker.NewClientFromEnv: %v", err)
        } else {
            self.dockerClient = dockerClient
        }
    } else {
        if dockerClient, err := docker.NewClient(options.DockerEndpoint); err != nil {
            return fmt.Errorf("docker.NewClient: %v", err)
        } else {
            self.dockerClient = dockerClient
        }
    }

    if dockerInfo, err := self.dockerClient.Info(); err != nil {
        return fmt.Errorf("dockerClient.Info: %v", err)
    } else {
        self.dockerName = dockerInfo.Get("name")
    }

    return nil
}

// Activate the given config
func (self *Manager) LoadConfig(config Config) error {
    // TODO: stop old config?

    self.config = &config

    return nil
}

func (self *Manager) LoadConfigReader(reader io.Reader) error {
    var config Config

    if _, err := toml.DecodeReader(reader, &config); err != nil {
        return err
    }

    return self.LoadConfig(config)
}

func (self *Manager) LoadConfigFile(filePath string) error {
    var config Config

    if _, err := toml.DecodeFile(filePath, &config); err != nil {
        return err
    }

    return self.LoadConfig(config)
}

func (self *Manager) LoadConfigString(data string) error {
    var config Config

    if _, err := toml.Decode(data, &config); err != nil {
        return err
    }

    return self.LoadConfig(config)
}

// Discover running docker containers
// This lets use down unnecessary containers when starting
func (self *Manager) Discover() (err error) {
    if dockerContainers, err := self.DockerList(); err != nil {
        return err
    } else {
        for _, dockerContainer := range dockerContainers {
            switch dockerContainer.Class {
            case "client":
                if err = self.discoverClient(dockerContainer); err != nil {
                    self.log.Printf("discoverClient %v: %v", dockerContainer, err)
                }
            case "worker":
                if err = self.discoverWorker(dockerContainer); err != nil {
                    self.log.Printf("discoverWorker %v: %v", dockerContainer, err)
                }
            default:
                self.log.Printf("Discover %v: unknown class", dockerContainer)
            }
        }

        return err
    }
}

// Get running configuration
func (self *Manager) DumpConfig() (string, error) {
    var buf bytes.Buffer

    if self.config == nil {

    } else if err := toml.NewEncoder(&buf).Encode(self.config); err != nil {
        return "", err
    }

    return buf.String(), nil
}

// Start new configuration
func (self *Manager) Start() error {
    if self.config == nil {
        return nil
    }

    self.log.Printf("Start config...\n");

    if self.config.Client == nil {

    } else if err := self.StartClients(*self.config.Client); err != nil {
        return err
    }

    if self.config.Worker == nil {

    } else if err := self.StartWorkers(*self.config.Worker); err != nil {
        return err
    }

    self.log.Printf("Started\n");

    return nil
}

// Stop current configuration
func (self *Manager) Stop() (err error) {
    if workersErr := self.StopWorkers(); workersErr != nil {
        err = workersErr
    }
    if clientsErr:= self.StopClients(); clientsErr != nil {
        err = clientsErr
    }

    return err
}

// Kill any running containers and reset state
func (self *Manager) Panic() (error) {
    self.log.Printf("Panic!\n");

    err := self.DockerPanic()

    self.config = nil
    self.clients = make(map[string]*Client)
    self.workers = make(map[string]*Worker)

    return err
}
