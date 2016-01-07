package control

import (
    "bytes"
    "close/config"
    "fmt"
    "encoding/json"
    "log"
    "path"
    "strings"
    "github.com/BurntSushi/toml"
)

type Worker struct {
    Type            string
    ID              uint
    Config          *WorkerConfig

    dockerContainer *DockerContainer
    configSub       *config.Sub
}

func (self Worker) String() string {
    return fmt.Sprintf("%s:%d", self.Type, self.ID)
}

type WorkerConfig struct {
    Name        string
    Type        string
    Count       uint

    Image       string
    Command     string
    Arg         string

    IDFlag      string

    RateConfig  string
    RateStats   string
}

func (self WorkerConfig) String() string {
    return self.Name
}

func (self *Manager) LoadConfig(filePath string) (workerConfig WorkerConfig, err error) {
    workerConfig.Name = strings.Split(path.Base(filePath), ".")[0]

    if _, err := toml.DecodeFile(filePath, &workerConfig); err != nil {
        return workerConfig, err
    }

    return workerConfig, nil
}

// Setup workers from config
func (self *Manager) StartWorkers(workerConfig WorkerConfig) error {
    self.workerConfig = &workerConfig

    for index := uint(1); index <= workerConfig.Count; index++ {
        dockerID := DockerID{WorkerType: workerConfig.Type, WorkerID: index}

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

        if workerConfig.Arg != "" {
            dockerConfig.AddArg(workerConfig.Arg)
        }

        // up
        worker := &Worker{
            Type:   workerConfig.Type,
            ID:     index,
            Config: self.workerConfig,
        }

        if container, err := self.DockerUp(dockerID, dockerConfig); err != nil {
            return fmt.Errorf("DockerUp %v: %v", dockerID, err)
        } else {
            log.Printf("DockerUP %v: %v\n", workerConfig, container)

            worker.dockerContainer = &container
        }

        if configSub, err := self.configRedis.Sub(config.SubOptions{Type: workerConfig.Type, ID: fmt.Sprintf("%d", worker.ID)}); err != nil {
            return fmt.Errorf("congigRedis.Sub: %v", err)
        } else {
            worker.configSub = configSub
        }

        self.workers[worker.String()] = worker
    }

    return nil
}

type WorkerStatus struct {
    Type            string  `json:"type"`
    ID              uint    `json:"id"`

    Docker          string  `json:"docker"`
    DockerStatus    string  `json:"docker_status"`

    ConfigTTL       float64 `json:"config_ttl"` // seconds

    Rate            json.Number `json:"rate"`   // config
}

func (self *Manager) WorkerConfig() (string, error) {
    var buf bytes.Buffer

    if err := toml.NewEncoder(&buf).Encode(self.workerConfig); err != nil {
        return "", err
    }

    return buf.String(), nil
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

        if configMap, err := worker.configSub.Get(); err != nil {
            return nil, err
        } else {
            switch rateValue := configMap[worker.Config.RateConfig].(type) {
            case json.Number:
                workerStatus.Rate = rateValue
            // XXX: why isn't this always just json.Number?
            case float64:
                workerStatus.Rate = json.Number(fmt.Sprintf("%v", rateValue))
            default:
                return nil, fmt.Errorf("invalid %s RateConfig=%v value type %T: %#v", worker.Config.Type, worker.Config.RateConfig, rateValue, rateValue)
            }
        }

        workers = append(workers, workerStatus)
    }

    return workers, nil
}
