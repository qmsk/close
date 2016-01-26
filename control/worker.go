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

    LatencyStats    string
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

/*
 * Get current configuration.
 *
 * TODO: do this in parallel for all worker instances using a pipelined redis get..
 */
func (worker *Worker) ConfigGet() (config.ConfigMap, error) {
    return worker.configSub.Get()
}

/*
 * Get configured rate from configMap.
 *
 * Returns 0 if no configured rate.
 */
func (worker *Worker) ConfigGetRate(configMap config.ConfigMap) (uint, error) {
    if worker.Config == nil || worker.Config.RateConfig == "" {
        return 0, nil
    }

    switch rateValue := configMap[worker.Config.RateConfig].(type) {
    case json.Number:
        if intValue, err := rateValue.Int64(); err != nil {
            return 0, err
        } else if intValue < 0 {
            return 0, fmt.Errorf("Negative rate %v[%v]: %v", worker, worker.Config.RateConfig, rateValue)
        } else {
            return uint(intValue), nil
        }
    default:
        return 0, fmt.Errorf("Invalid %v.RateConfig=%v value type %T: %#v", worker.Config, worker.Config.RateConfig, rateValue, rateValue)
    }
}

/*
 * Get configured stats instance.
 *
 * Returns "" if no configured stats instance.
 */
func (worker *Worker) StatsInstance(configMap config.ConfigMap) (string, error) {
    if worker.Config == nil {
        return "", nil
    }
    if worker.Config.StatsInstanceFromConfig == "" {
        // default to config instance
        return worker.String(), nil
    }

    // lookup from config
    switch configValue := configMap[worker.Config.StatsInstanceFromConfig].(type) {
    case string:
        return configValue, nil
    case json.Number:
        return configValue.String(), nil
    default:
         return "", fmt.Errorf("Invalid %v.StatsInstanceFromConfig=%v for %v: type %T: %#v", worker.Config, worker.Config.StatsInstanceFromConfig, worker, configValue, configValue)
    }
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

    dockerConfig.Env.Add("CLOSE_INSTANCE", worker)
    dockerConfig.Env.Add("INFLUXDB_URL", self.options.Stats.InfluxURL)
    dockerConfig.Env.Add("REDIS_URL", self.options.Config.RedisURL)

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

    RateConfig      uint        `json:"rate_config"`    // config

    StatsInstance   string      `json:"stats_instance"`
}

func (self *Manager) workerGet(worker *Worker, detail bool) (WorkerStatus, error) {
    workerStatus := WorkerStatus{
        Instance:   worker.Instance,
    }

    if worker.Config != nil {
        workerStatus.Config = worker.Config.String()
    }

    if detail {
        workerStatus.WorkerConfig = worker.Config
    }

    if dockerContainer, err := self.DockerGet(worker.dockerContainer.String()); err != nil {
        return workerStatus, fmt.Errorf("ListWorkers %v: DockerGet %v: %v", worker, worker.dockerContainer, err)
    } else if dockerContainer == nil {
        workerStatus.Docker = ""
        workerStatus.DockerStatus = ""
        workerStatus.State = WorkerDown
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
    if configMap, err := worker.ConfigGet(); err != nil {
        self.log.Printf("ListWorkers %v: ConfigGet: %v\n", worker, err)

        workerStatus.ConfigError = err.Error()
    } else {
        if detail {
            workerStatus.ConfigMap = configMap
        }

        if rate, err := worker.ConfigGetRate(configMap); err != nil {
            workerStatus.ConfigError = err.Error()
        } else {
            workerStatus.RateConfig = rate
        }

        if statsInstance, err := worker.StatsInstance(configMap); err != nil {
            workerStatus.ConfigError = err.Error()
        } else {
            workerStatus.StatsInstance = statsInstance
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
