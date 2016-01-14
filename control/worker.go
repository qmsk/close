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
    Type        string
    IDFlag      string

    RateConfig  string
    RateStats   string
}

func (self WorkerConfig) String() string {
    return self.name
}

// track state of managed workers
type Worker struct {
    Config          *WorkerConfig
    ID              uint

    dockerContainer *DockerContainer
    configSub       *config.Sub
}

func (self Worker) String() string {
    return fmt.Sprintf("%v:%d", self.Config, self.ID)
}

func (self *Manager) discoverWorker(dockerContainer *DockerContainer) (*Worker, error) {
    workerConfig := self.config.Workers[dockerContainer.Type]

    if workerConfig == nil {
        return nil, fmt.Errorf("Unknown worker config type: %v", dockerContainer.Type)
    }

    worker := &Worker{
        Config: workerConfig,
        ID:     dockerContainer.Index,

        dockerContainer:    dockerContainer,
    }

    if subOptions, err := config.ParseSub(worker.Config.Type, fmt.Sprintf("%d", worker.ID)); err != nil {
        return nil, fmt.Errorf("config.ParseSub: %v", err)
    } else if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return nil, fmt.Errorf("congigRedis.Sub %v: %v", subOptions, err)
    } else {
        worker.configSub = configSub
    }

    return worker, nil
}

func (self *Manager) workerUp(workerConfig *WorkerConfig, index uint) (*Worker, error) {
    worker := &Worker{
        Config: workerConfig,
        ID:     index,
    }

    // docker
    dockerID := DockerID{Class:"worker", Type: workerConfig.String(), Index: index}

    dockerConfig := DockerConfig{
        Image:      workerConfig.Image,
        Command:    workerConfig.Command,
        Env:        []string{
            fmt.Sprintf("CLOSE_ID=%d", index),
        },
        Privileged: workerConfig.Privileged,
    }

    dockerConfig.AddFlag("influxdb-addr", self.options.StatsReader.InfluxDB.Addr)
    dockerConfig.AddFlag("influxdb-database", self.options.StatsReader.Database)
    dockerConfig.AddFlag("config-redis-addr", self.options.Config.Redis.Addr)
    dockerConfig.AddFlag("config-redis-db", self.options.Config.Redis.DB)
    dockerConfig.AddFlag("config-prefix", self.options.Config.Prefix)

    if workerConfig.IDFlag != "" {
        dockerConfig.AddFlag(workerConfig.IDFlag, index)
    }

    dockerConfig.AddArg(workerConfig.Args...)

    if workerConfig.Client != "" {
        if client, err := self.GetClient(workerConfig.Client, index); err != nil {
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

    // XXX: this is not unique if there are multiple config.Workers with the same Type!
    if subOptions, err := config.ParseSub(worker.Config.Type, fmt.Sprintf("%d", worker.ID)); err != nil {
        return nil, fmt.Errorf("config.ParseSub: %v", err)
    } else if configSub, err := self.configRedis.Sub(subOptions); err != nil {
        return worker, fmt.Errorf("congigRedis.Sub: %v", err)
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
        if worker, err := self.workerUp(workerConfig, index); err != nil {
            return fmt.Errorf("WorkerUp %v: workerUp %v: %v", workerConfig, index, err)
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
    Config          string      `json:"config"` // WorkerConfig.name
    ID              uint        `json:"id"`

    Docker          string  `json:"docker"`
    DockerStatus    string  `json:"docker_status"`

    State           WorkerState `json:"state"`

    ConfigError     string      `json:"config_error,omitempty"`
    ConfigTTL       float64     `json:"config_ttl"` // seconds

    Rate            json.Number `json:"rate"`   // config
}

func (self *Manager) ListWorkers() (workers []WorkerStatus, err error) {
    for _, worker := range self.workers {
        workerStatus := WorkerStatus{
            Config:     worker.Config.name,
            ID:         worker.ID,
        }

        if dockerContainer, err := self.DockerGet(worker.dockerContainer.String()); err != nil {
            return nil, fmt.Errorf("ListWorkers %v: DockerGet %v: %v", worker, worker.dockerContainer, err)
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
            workerStatus.ConfigTTL = configTTL.Seconds()
        }


        if configMap, err := worker.configSub.Get(); err != nil {
            self.log.Printf("ListWorkers %v: configSub.Get %v: %v\n", worker, worker.configSub, err)

            workerStatus.ConfigError = err.Error()
        } else {
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
        }

        workers = append(workers, workerStatus)
    }

    return workers, nil
}
