package control

import (
    "bytes"
    "close/config"
    "github.com/fsouza/go-dockerclient"
    "fmt"
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

func (self *Manager) LoadConfig(filePath string) (config Config, err error) {
    if _, err := toml.DecodeFile(filePath, &config); err != nil {
        return config, err
    }

    return config, nil
}

func (self *Manager) LoadConfigString(data string) (workerConfig WorkerConfig, err error) {
    if _, err := toml.Decode(data, &workerConfig); err != nil {
        return workerConfig, err
    }

    return workerConfig, nil
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
func (self *Manager) Start(config Config) error {
    self.config = &config

    self.log.Printf("Start config...\n");

    if config.Client == nil {

    } else if err := self.StartClients(*config.Client); err != nil {
        return err
    }

    if config.Worker == nil {

    } else if err := self.StartWorkers(*config.Worker); err != nil {
        return err
    }

    self.log.Printf("Started\n");

    return nil
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
