package control

import (
    "close/config"
    "fmt"
    "encoding/json"
)

type WorkerConfig struct {
    name        string
    Count       uint
    Client      string // ClientConfig.name

    Image       string
    Privileged  bool
    Command     string
    Args        []string

    // worker 
    Type            string
    InstanceFlag    string

    StatsType       string      // default: .Type
    StatsInstanceFromConfig string

    RateConfig      string
    RateStats       string
}

func (self WorkerConfig) String() string {
    return self.name
}

// track state of managed workers
type Worker struct {
    Config          *WorkerConfig
    Instance        string

    dockerContainer *DockerContainer
    configSub       *config.Sub
}

func (self Worker) configID() config.ID {
    return config.ID{Type:self.Config.Type, Instance:self.String()}
}

func (self Worker) String() string {
    return fmt.Sprintf("%v:%s", self.Config, self.Instance)
}

func (self *Manager) discoverWorker(dockerContainer *DockerContainer) (*Worker, error) {
    workerConfig := self.config.Workers[dockerContainer.Type]

    if workerConfig == nil {
        return nil, fmt.Errorf("Unknown worker config type: %v", dockerContainer.Type)
    }

    worker := &Worker{
        Config:     workerConfig,
        Instance:   dockerContainer.Instance,

        dockerContainer:    dockerContainer,
    }

    if configSub, err := self.configRedis.GetSub(worker.configID()); err != nil {
        return nil, fmt.Errorf("configRedis.GetSub: %v", err)
    } else {
        worker.configSub = configSub
    }

    return worker, nil
}

// Lookup current state of worker
func (self *Manager) workerUp(workerConfig *WorkerConfig, instance string) (*Worker, error) {
    worker := &Worker{
        Config:     workerConfig,
        Instance:   instance,
    }

    // docker
    dockerID := DockerID{Class:"worker", Type: workerConfig.String(), Instance: instance}

    dockerConfig := DockerConfig{
        Image:      workerConfig.Image,
        Command:    workerConfig.Command,
        Privileged: workerConfig.Privileged,
    }

    // TODO: rename to CLOSE_WORKER=?
    dockerConfig.Env.Add("CLOSE_INSTANCE", worker.String())

    dockerConfig.AddFlag("influxdb-addr", self.options.StatsReader.InfluxDB.Addr)
    dockerConfig.AddFlag("influxdb-database", self.options.StatsReader.Database)
    dockerConfig.AddFlag("config-redis-addr", self.options.Config.Redis.Addr)
    dockerConfig.AddFlag("config-redis-db", self.options.Config.Redis.DB)
    dockerConfig.AddFlag("config-prefix", self.options.Config.Prefix)

    if workerConfig.InstanceFlag != "" {
        dockerConfig.AddFlag(workerConfig.InstanceFlag, worker.String())
    }

    dockerConfig.AddArg(workerConfig.Args...)

    if workerConfig.Client != "" {
        if client, err := self.GetClient(workerConfig.Client, instance); err != nil {
            return worker, fmt.Errorf("GetClient: %v", err)
        } else {
            dockerConfig.SetNetworkContainer(client.dockerContainer)
        }
    }

    if container, err := self.DockerUp(dockerID, dockerConfig); err != nil {
        return worker, fmt.Errorf("DockerUp %v: %v", dockerID, err)
    } else {
        worker.dockerContainer = container
    }

    if configSub, err := self.configRedis.GetSub(worker.configID()); err != nil {
        return nil, fmt.Errorf("configRedis.GetSub: %v", err)
    } else {
        worker.configSub = configSub
    }

    return worker, nil
}

func (self *Manager) markWorkers() {
    for _, worker := range self.workers {
        worker.Config = nil
    }
}

// Setup workers from config
func (self *Manager) WorkerUp(workerConfig *WorkerConfig) error {
    self.log.Printf("WorkerUp %v: Start %d workers...\n", workerConfig, workerConfig.Count)

    for index := uint(1); index <= workerConfig.Count; index++ {
        instance := fmt.Sprintf("%d", index)

        if worker, err := self.workerUp(workerConfig, instance); err != nil {
            return fmt.Errorf("WorkerUp %v: workerUp %v: %v", workerConfig, instance, err)
        } else {
            self.workers[worker.String()] = worker
        }
    }

    return nil
}

// Stop running workers for given config
// Call with config=nil to cleanup all unconfigured workers
func (self *Manager) WorkerDown(config *WorkerConfig) error {
    // sweep
    for key, worker := range self.workers {
        if worker.Config == config {
            if err := self.DockerDown(worker.dockerContainer); err != nil {
                return fmt.Errorf("WorkerDown %v: DockerDown %v: %v", config, worker.dockerContainer, err)
            }

            delete(self.workers, key)
        }
    }

    return nil
}

type WorkerState string

var WorkerDown      WorkerState     = "down"    // not running, clean exit
var WorkerUnknown   WorkerState     = "unknown" // running, unknown
var WorkerWait      WorkerState     = "wait"    // running, pending
var WorkerUp        WorkerState     = "up"      // running, ready
var WorkerError     WorkerState     = "error"   // not running, unclean exit

type WorkerStatus struct {
    Config          string      `json:"config"`         // WorkerConfig.name
    Instance        string      `json:"instance"`

    WorkerConfig    *WorkerConfig   `json:"worker_config,omitempty"`    // detail

    Docker          string      `json:"docker"`
    DockerStatus    string      `json:"docker_status"`

    State           WorkerState `json:"state"`

    ConfigInstance  string              `json:"config_instance"`
    ConfigError     string              `json:"config_error,omitempty"`
    ConfigTTL       float64             `json:"config_ttl"` // seconds
    ConfigMap       config.ConfigMap    `json:"config_map,omitempty"`   // detail

    Rate            json.Number `json:"rate"`   // config

    StatsInstance   string      `json:"stats_instance"`
}

func (self *Manager) workerGet(worker *Worker, detail bool) (WorkerStatus, error) {
    workerStatus := WorkerStatus{
        Config:     worker.Config.name,
        Instance:   worker.Instance,
    }

    if detail {
        workerStatus.WorkerConfig = worker.Config
    }

    if dockerContainer, err := self.DockerGet(worker.dockerContainer.String()); err != nil {
        return workerStatus, fmt.Errorf("ListWorkers %v: DockerGet %v: %v", worker, worker.dockerContainer, err)
    } else {
        workerStatus.Docker = dockerContainer.String()
        workerStatus.DockerStatus = dockerContainer.Status

        if dockerContainer.State.Running {
            workerStatus.State = WorkerUp
        } else if dockerContainer.State.ExitCode == 0 {
            workerStatus.State = WorkerDown
        } else {
            workerStatus.State = WorkerError
        }
    }

    if configTTL, err := worker.configSub.Check(); err != nil {
        if workerStatus.State == WorkerUp {
            workerStatus.State = WorkerWait
        }
    } else {
        workerStatus.ConfigInstance = worker.String()
        workerStatus.ConfigTTL = configTTL.Seconds()
    }

    // current running config
    if configMap, err := worker.configSub.Get(); err != nil {
        self.log.Printf("ListWorkers %v: configSub.Get %v: %v\n", worker, worker.configSub, err)

        workerStatus.ConfigError = err.Error()
    } else {
        if detail {
            workerStatus.ConfigMap = configMap
        }

        switch rateValue := configMap[worker.Config.RateConfig].(type) {
        case json.Number:
            workerStatus.Rate = rateValue
        // XXX: why isn't this always just json.Number?
        case float64:
            workerStatus.Rate = json.Number(fmt.Sprintf("%v", rateValue))
        case nil:
            // XXX: not yet set...
        default:
            workerStatus.ConfigError = fmt.Sprintf("invalid %s RateConfig=%v value type %T: %#v", worker.Config.Type, worker.Config.RateConfig, rateValue, rateValue)
        }

        if worker.Config.StatsInstanceFromConfig == "" {
            workerStatus.StatsInstance = worker.String()
        } else if configValue, exists := configMap[worker.Config.StatsInstanceFromConfig]; !exists {
            workerStatus.ConfigError = fmt.Sprintf("Invalid %s StatsInstanceFromConfig %v: not found", worker.Config, worker.Config.StatsInstanceFromConfig)
        } else if statsInstance, ok := configValue.(string); ok {
            workerStatus.StatsInstance = statsInstance
        } else if statsInstance, ok := configValue.(json.Number); ok {
            // XXX: not as floating point!
            workerStatus.StatsInstance = statsInstance.String()
        } else {
            workerStatus.ConfigError = fmt.Sprintf("Invalid %s StatsInstanceFromConfig %v: type %T: %#v", worker.Config, worker.Config.StatsInstanceFromConfig, configValue, configValue)
        }
    }

    return workerStatus, nil
}

func (self *Manager) ListWorkers() (workers []WorkerStatus, err error) {
    for _, worker := range self.workers {
        if workerStatus, err := self.workerGet(worker, false); err != nil {
            return workers, err
        } else {
            workers = append(workers, workerStatus)
        }
    }

    return workers, nil
}

func (self *Manager) WorkerGet(configName string, instance string) (*WorkerStatus, error) {
    workerName := fmt.Sprintf("%s:%s", configName, instance)

    if worker, found := self.workers[workerName]; !found {
        return nil, nil
    } else if workerStatus, err := self.workerGet(worker, true); err != nil {
        return nil, err
    } else {
        return &workerStatus, nil
    }
}
