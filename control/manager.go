package control

import (
    "bytes"
    "close/config"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "net/http"
    "log"
    "close/stats"
    "github.com/BurntSushi/toml"
)

type Options struct {
    StatsReader     stats.ReaderConfig
    Config          config.Options
    DockerEndpoint  string
}

// full-system configuration
type Config struct {
    Client          *ClientConfig
    Worker          *WorkerConfig
}

type Manager struct {
    options         Options

    logs            *Logs
    configRedis     *config.Redis
    statsReader     *stats.Reader
    dockerClient    *docker.Client
    dockerName      string

    // state
    log             *log.Logger
    config          *Config
    clients         map[string]*Client
    workers         map[string]*Worker
}

func New(options Options) (*Manager, error) {
    self := &Manager{
        options:    options,

        clients:        make(map[string]*Client),
        workers:        make(map[string]*Worker),
    }

    if err := self.init(options); err != nil {
        return nil, err
    }

    return self, nil
}

func (self *Manager) init(options Options) error {
    if logs, err := NewLogs(); err != nil {
        return fmt.Errorf("NewLogs: %v", err)
    } else {
        self.logs = logs
        self.log = logs.Logger("Manager: ")
    }

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

func (self *Manager) LogsHandler() http.Handler {
    return self.logs
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
