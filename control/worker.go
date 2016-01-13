package control

import (
    "close/config"
    "fmt"
    "encoding/json"
)

type WorkerConfig struct {
    Name        string
    Count       uint
    Client      string // ClientConfig.Name

    Image       string
    Command     string
    Args        []string

    // worker 
    Type        string
    IDFlag      string

    RateConfig  string
    RateStats   string
}

func (self WorkerConfig) String() string {
    return self.Name
}

// track state of managed workers
type Worker struct {
    Config          *WorkerConfig
    Type            string
    ID              uint

    dockerContainer *DockerContainer
    configSub       *config.Sub
}

func (self Worker) String() string {
    return fmt.Sprintf("%s:%d", self.Type, self.ID)
}

func (self *Manager) discoverWorker(dockerContainer *DockerContainer) error {
    worker := &Worker{
        Type:   dockerContainer.Type,
        ID:     dockerContainer.Index,

        Config: nil, // TODO

        dockerContainer:    dockerContainer,
    }

    if configSub, err := self.configRedis.Sub(config.SubOptions{Type: worker.Type, ID: fmt.Sprintf("%d", worker.ID)}); err != nil {
        return fmt.Errorf("congigRedis.Sub: %v", err)
    } else {
        worker.configSub = configSub
    }

    self.workers[worker.String()] = worker

    return nil
}

// Setup workers from config
func (self *Manager) StartWorkers(workerConfig WorkerConfig) error {
    // mark
    for _, worker := range self.workers {
        worker.Config = nil
    }

    self.log.Printf("Start %d %s workers...\n", workerConfig.Count, workerConfig.Name);

    for index := uint(1); index <= workerConfig.Count; index++ {
        worker := &Worker{
            Type:   workerConfig.Type,
            ID:     index,
            Config: &workerConfig,
        }

        // docker
        dockerID := DockerID{Class:"worker", Type: workerConfig.Type, Index: index}

        dockerConfig := DockerConfig{
            Image:      workerConfig.Image,
            Command:    workerConfig.Command,
            Env:        []string{
                fmt.Sprintf("CLOSE_ID=%d", index),
            },
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
                return fmt.Errorf("Worker %v GetClient: %v", worker, err)
            } else {
                dockerConfig.SetNetworkContainer(client.dockerContainer)
            }
        }

        if container, err := self.DockerUp(dockerID, dockerConfig); err != nil {
            return fmt.Errorf("DockerUp %v: %v", dockerID, err)
        } else {
            worker.dockerContainer = container
        }

        if configSub, err := self.configRedis.Sub(config.SubOptions{Type: workerConfig.Type, ID: fmt.Sprintf("%d", worker.ID)}); err != nil {
            return fmt.Errorf("congigRedis.Sub: %v", err)
        } else {
            worker.configSub = configSub
        }

        self.workers[worker.String()] = worker
    }

    // sweep
    for key, worker := range self.workers {
        if worker.Config != nil {
            continue
        }

        if err := self.DockerDown(worker.dockerContainer); err != nil {
            self.log.Printf("DockerDown %v: %v", worker.dockerContainer, err)
        }

        delete(self.workers, key)
    }

    return nil
}

// Stop all running workers
func (self *Manager) StopWorkers() (retErr error) {
    // sweep
    for key, worker := range self.workers {
        if err := self.DockerDown(worker.dockerContainer); err != nil {
            self.log.Printf("DockerDown %v: %v", worker.dockerContainer, err)
            retErr = err
        }

        delete(self.workers, key)
    }

    return retErr
}

type WorkerStatus struct {
    Type            string  `json:"type"`
    ID              uint    `json:"id"`

    Docker          string  `json:"docker"`
    DockerStatus    string  `json:"docker_status"`

    ConfigTTL       float64 `json:"config_ttl"` // seconds

    Rate            json.Number `json:"rate"`   // config
}

func (self *Manager) ListWorkers() (workers []WorkerStatus, err error) {
    for _, worker := range self.workers {
        workerStatus := WorkerStatus{
            Type:       worker.Type,
            ID:         worker.ID,
        }

        if dockerContainer, err := self.DockerGet(worker.dockerContainer.String()); err != nil {
            return nil, err
        } else {
            workerStatus.Docker = dockerContainer.String()
            workerStatus.DockerStatus = dockerContainer.Status
        }

        if configTTL, err := worker.configSub.Check(); err != nil {
            return nil, err
        } else {
            workerStatus.ConfigTTL = configTTL.Seconds()
        }

        if worker.Config != nil {
            configMap, err := worker.configSub.Get()
            if err != nil {
                return nil, err
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
                return nil, fmt.Errorf("invalid %s RateConfig=%v value type %T: %#v", worker.Config.Type, worker.Config.RateConfig, rateValue, rateValue)
            }
        }

        workers = append(workers, workerStatus)
    }

    return workers, nil
}
