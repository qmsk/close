package control

import (
    "close/config"
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "close/stats"
)

type Options struct {
    StatsReader     stats.ReaderConfig
    Config          config.Options
    DockerEndpoint  string
}

type Manager struct {
    options         Options

    configRedis     *config.Redis
    statsReader     *stats.Reader
    dockerClient    *docker.Client
    dockerName      string

    workerConfig    *WorkerConfig               `json:"worker_config"` // active
    workers         map[string]*Worker          `json:"workers"`
}

func New(options Options) (*Manager, error) {
    self := &Manager{
        options:    options,

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
