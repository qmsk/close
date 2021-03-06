package control

import (
    "github.com/qmsk/close/config"
    "github.com/qmsk/close/docker"
    "fmt"
    "encoding/json"
    "github.com/qmsk/close/stats"
    "strings"
    "time"
    "net/url"
)

type WorkerConfig struct {
    name        string
    Count       uint
    Client      string // ClientConfig.name

    // Docker
    Constraints []string
    Image       string
    Privileged  bool
    Command     string
    Args        []string

    // worker 
    Type            string
    InstanceFlag    string

    /*
     * Stats URL: type [ "/" field ] [ "?" ("instance=" ("$" | "$" configName ) ) [ "&" ... ] ]
     */
    Stats           string

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
    up              bool

    dockerID        docker.ID
    configSub       *config.Sub
}

func (self Worker) configID() config.ID {
    return config.ID{Type:self.Config.Type, Instance:self.String()}
}

func (self Worker) String() string {
    return fmt.Sprintf("%v:%s", self.Config, self.Instance)
}

func (self *Manager) discoverWorker(dockerID docker.ID) (*Worker, error) {
    workerConfig := self.config.Workers[dockerID.Type]

    if workerConfig == nil {
        return nil, nil
    }

    worker := &Worker{
        Config:     workerConfig,
        Instance:   dockerID.Instance,

        dockerID:   dockerID,
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
        up:         true,
        dockerID:   docker.ID{Class:"worker", Type: workerConfig.String(), Instance: instance},
    }

    // docker
    dockerConfig := docker.Config{
        Image:      workerConfig.Image,
        Command:    workerConfig.Command,
        Privileged: workerConfig.Privileged,
    }

    dockerConfig.Env.AddEnv("CLOSE_INSTANCE", worker)
    dockerConfig.Env.AddEnv("INFLUXDB_URL", self.options.Stats.InfluxURL)
    dockerConfig.Env.AddEnv("REDIS_URL", self.options.Config.RedisURL)

    if workerConfig.InstanceFlag != "" {
        dockerConfig.AddFlag(workerConfig.InstanceFlag, worker.String())
    }

    dockerConfig.AddArg(workerConfig.Args...)

    dockerConfig.Constraints.Set(workerConfig.Constraints)

    if workerConfig.Client != "" {
        if client, err := self.GetClient(workerConfig.Client, instance); err != nil {
            return worker, fmt.Errorf("GetClient: %v", err)
        } else {
            dockerConfig.SetNetworkContainer(client.dockerID)
        }
    }

    if container, err := self.docker.Up(worker.dockerID, dockerConfig); err != nil {
        return worker, fmt.Errorf("docker.Up %v: %v", worker.dockerID, err)
    } else {
        self.log.Printf("docker.Up %v: %v", worker.dockerID, container)
    }

    if configSub, err := self.configRedis.GetSub(worker.configID()); err != nil {
        return worker, fmt.Errorf("configRedis.GetSub: %v", err)
    } else {
        worker.configSub = configSub
    }

    return worker, nil
}

// Mark all workers for sweep
func (self *Manager) markWorkers() {
    for _, worker := range self.workers {
        worker.up = false
    }
}

// Setup workers from config
func (self *Manager) WorkerUp(workerConfig *WorkerConfig) (errs []error) {
    self.log.Printf("WorkerUp %v: Start %d workers...\n", workerConfig, workerConfig.Count)

    for index := uint(1); index <= workerConfig.Count; index++ {
        instance := fmt.Sprintf("%d", index)

        worker, err := self.workerUp(workerConfig, instance)
        if err != nil {
            errs = append(errs, fmt.Errorf("WorkerUp %v: workerUp %v: %v", workerConfig, instance, err))
        }
        if worker != nil {
            self.workers[worker.String()] = worker
        }
    }

    return errs
}

// Stop any workers that are not configured, or are marked
func (self *Manager) sweepWorkers() (errs []error) {
    for _, worker := range self.workers {
        if worker.up {
            continue
        }

        if err := self.docker.Down(worker.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("WorkerDown %v: docker.Down %v: %v", worker, worker.dockerID, err))
        }
    }

    return errs
}

// Stop running workers for given config
//
// Call with config=nil to stop all workers.
func (self *Manager) WorkerDown(config *WorkerConfig) (errs []error) {
    for _, worker := range self.workers {
        if config != nil && worker.Config != config {
            continue
        }

        worker.up = false

        if err := self.docker.Down(worker.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("WorkerDown %v: docker.Down %v: %v", config, worker.dockerID, err))
        }
    }

    return errs
}

// Cleanup down'd workers
func (self *Manager) WorkerClean() (errs []error) {
    for key, worker := range self.workers {
        if worker.up {
            continue
        }

        if err := self.docker.Clean(worker.dockerID); err != nil {
            errs = append(errs, fmt.Errorf("WorkerClean %v: docker.Clean %v: %v", worker, worker.dockerID, err))
        }

        delete(self.workers, key)
    }

    return errs
}

/*
 * ListWorkers() needs to query stats for a lot of workers, but often only for a couple different types...
 * aggregate and cache those queries.
 */
type workerCache struct {
    statsReader     *stats.Reader
    eager           bool

    dockerCache     *docker.Cache

    configCache     map[*Worker]config.ConfigMap
    statsCache      map[string]stats.SeriesStats        // %s/%s?instance=%s
    statsIndex      map[string]bool                     // statsUrl
}

func makeWorkerCache(manager *Manager, eager bool) workerCache {
    cache := workerCache{
        statsReader:    manager.statsReader,
        eager:          eager,

        dockerCache:    manager.docker.NewCache(eager),

        configCache:    make(map[*Worker]config.ConfigMap),
        statsCache:     make(map[string]stats.SeriesStats),
        statsIndex:     make(map[string]bool),
    }

    return cache
}


/*
 * Get current configuration for worker.
 *
 * TODO: do this in parallel for all worker instances using a pipelined redis get..
 */
func (cache *workerCache) ConfigGet(worker *Worker) (config.ConfigMap, error) {
    if configMap, exists := cache.configCache[worker]; exists {
        return configMap, nil
    } else if worker.configSub == nil {
        return nil, fmt.Errorf("No ConfigSub for worker")
    } else if configMap, err := worker.configSub.Get(); err != nil {
        return nil, err
    } else {
        cache.configCache[worker] = configMap

        return configMap, nil
    }
}

/*
 * Get configured rate from configMap.
 *
 * Returns 0 if no configured rate.
 */
func (cache *workerCache) ConfigGetRate(worker *Worker) (uint, error) {
    if worker.Config.RateConfig == "" {
        return 0, nil
    } else if configMap, err := cache.ConfigGet(worker); err != nil {
        return 0, err
    } else {
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
}

/*
 * Get configured stats instance.
 *
 * Returns "" if no configured stats instance.
 */
func (cache *workerCache) parseStats(worker *Worker, statsUrl string) (series stats.SeriesKey, field string, err error) {
    parseUrl, err := url.Parse(statsUrl)
    if err != nil {
        return series, "", err
    }
    pathParts := strings.Split(parseUrl.Path, "/")

    switch len(pathParts) {
    case 0:
        series.Type = worker.Config.Type
    case 1:
        series.Type = pathParts[0]
    case 2:
        series.Type = pathParts[0]
        field = pathParts[1]
    default:
        return series, field, fmt.Errorf("Invalid stats path: %v", pathParts)
    }

    if urlHostname := parseUrl.Query().Get("hostname"); urlHostname == "" {
        series.Hostname = ""
    } else {
        series.Hostname = urlHostname
    }

    if urlInstance := parseUrl.Query().Get("instance"); urlInstance == "" {
        series.Instance = ""
    } else if urlInstance == "$" {
        // config instance
        series.Instance = worker.String()
    } else if strings.HasPrefix(urlInstance, "$") {
        configMap, err := cache.ConfigGet(worker)
        if err != nil {
            return series, field, err
        }

        // lookup from config
        switch configValue := configMap[strings.TrimPrefix(urlInstance, "$")].(type) {
        case string:
            series.Instance = configValue
        case json.Number:
            series.Instance = configValue.String()
        default:
            return series, field, fmt.Errorf("Invalid stats URL ?instance=%v: config type %T: %#v", urlInstance, worker, configValue, configValue)
        }
    } else {
        series.Instance = urlInstance
    }

    return
}

func (cache *workerCache) StatsMeta(worker *Worker) (stats.SeriesKey, error) {
    if worker.Config == nil || worker.Config.Stats == "" {
        return stats.SeriesKey{}, nil
    }

    if seriesKey, field, err := cache.parseStats(worker, worker.Config.Stats); err != nil {
        return seriesKey, fmt.Errorf("parseStats %v: %v", worker.Config.Stats, err)
    } else if field != "" {
        return seriesKey, fmt.Errorf("WorkerConfig %v.Stats=%v: should not have field", worker.Config, worker.Config.Stats)
    } else {
        return seriesKey, err
    }
}

func (cache *workerCache) StatsGet(worker *Worker, statsUrl string) (*stats.SeriesStats, error) {
    duration := 10 * time.Second

    if statsUrl == "" {
        return nil, nil
    }

    seriesKey, field, err := cache.parseStats(worker, statsUrl)
    if err != nil {
        return nil, fmt.Errorf("parseStats %v: %v", statsUrl, err)
    }

    // get from warm cache
    cacheIndex := fmt.Sprintf("%s/%s", seriesKey.Type, field)
    cacheKey := fmt.Sprintf("%s/%s?instance=%s", seriesKey.Type, field, seriesKey.Instance)

    if stat, exists := cache.statsCache[cacheKey]; exists {
        return &stat, nil
    } else if cache.statsIndex[cacheIndex] {
        // negative cache
        return nil, nil
    } else if cache.eager {
        // prefetch all instances into cache, and mark index
        seriesKey.Instance = ""
    } else {
        // normal single-fetch into cache
    }

    if stats, err := cache.statsReader.GetStats(seriesKey, field, duration); err != nil {
        return nil, fmt.Errorf("stats.Reader: GetStats seriesKey=%v field=%v duration=%v: %v", seriesKey, field, duration, err)
    } else {
        for _, stat := range stats {
            statKey := fmt.Sprintf("%s/%s?instance=%s", stat.Type, stat.Field, stat.Instance)

            cache.statsCache[statKey] = stat
        }

        if cache.eager {
            // mark index as cached
            cache.statsIndex[cacheIndex] = true
        }
    }

    // get from hot cache
    if stat, found := cache.statsCache[cacheKey]; !found {
        return nil, fmt.Errorf("StatsGet %v %v: Not found %v", worker, statsUrl, cacheKey)
    } else {
        return &stat, nil
    }
}

func (cache *workerCache) getStatus(worker *Worker, detail bool) (WorkerStatus, error) {
    workerStatus := WorkerStatus{
        Instance:   worker.Instance,
        Up:         worker.up,
    }

    // docker
    if !detail {

    } else if dockerContainer, err := cache.dockerCache.GetContainer(worker.dockerID); err != nil {
        return workerStatus, fmt.Errorf("ListWorkers %v: docker.Get %v: %v", worker, worker.dockerID, err)
    } else {
        workerStatus.DockerContainer = dockerContainer
    }

    if containerStatus, err := cache.dockerCache.GetStatus(worker.dockerID); err != nil {
        workerStatus.Docker = worker.dockerID.String()
        workerStatus.DockerStatus = fmt.Sprintf("%v", err)
        workerStatus.State = WorkerError
    } else if containerStatus == nil {
        workerStatus.Docker = ""
        workerStatus.DockerStatus = ""
        workerStatus.State = WorkerDown
    } else {
        workerStatus.Docker = containerStatus.String()
        workerStatus.DockerStatus = containerStatus.Status
        workerStatus.DockerNode = containerStatus.Node

        if containerStatus.IsUp() {
            workerStatus.State = WorkerUp
        } else if containerStatus.IsError() {
            workerStatus.State = WorkerError
        } else {
            workerStatus.State = WorkerDown
        }
    }

    // config
    if worker.Config != nil {
        workerStatus.Config = worker.Config.String()

        if detail {
            workerStatus.WorkerConfig = worker.Config
        }

        if worker.configSub == nil {

        } else if configTTL, err := worker.configSub.Check(); err != nil {
            if workerStatus.State == WorkerUp {
                workerStatus.State = WorkerWait
            }
        } else {
            workerStatus.ConfigInstance = worker.String()
            workerStatus.ConfigTTL = configTTL.Seconds()
        }

        // current running config
        if !worker.up {
            // ignore
        } else if configMap, err := cache.ConfigGet(worker); err != nil {

        } else {
            if detail {
                workerStatus.ConfigMap = configMap
            }

            if rate, err := cache.ConfigGetRate(worker); err != nil {
                workerStatus.ConfigError = err.Error()
            } else {
                workerStatus.RateConfig = rate
            }

            if seriesKey, err := cache.StatsMeta(worker); err != nil {
                workerStatus.StatsMeta = seriesKey
                workerStatus.ConfigError = err.Error()
            } else {
                workerStatus.StatsMeta = seriesKey
            }

            if rateStats, err := cache.StatsGet(worker, worker.Config.RateStats); err != nil {
                workerStatus.ConfigError = err.Error()
            } else {
                workerStatus.RateStats = rateStats
            }

            if latencyStats, err := cache.StatsGet(worker, worker.Config.LatencyStats); err != nil {
                workerStatus.ConfigError = err.Error()
            } else {
                workerStatus.LatencyStats = latencyStats
            }
        }
    }

    return workerStatus, nil
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

    Docker          string              `json:"docker"`
    DockerStatus    string              `json:"docker_status"`
    DockerNode      string              `json:"docker_node"`
    DockerContainer *docker.Container    `json:"docker_container,omitempty"` // detail

    Up              bool                `json:"up"`
    State           WorkerState         `json:"state"`

    ConfigInstance  string              `json:"config_instance"`
    ConfigError     string              `json:"config_error,omitempty"`
    ConfigTTL       float64             `json:"config_ttl"` // seconds
    ConfigMap       config.ConfigMap    `json:"config_map,omitempty"`   // detail

    StatsMeta       stats.SeriesKey     `json:"stats_meta"`

    RateConfig      uint                `json:"rate_config,omitempty"`    // config
    RateStats       *stats.SeriesStats  `json:"rate_stats,omitempty"`

    LatencyStats    *stats.SeriesStats  `json:"latency_stats,omitempty"`
}

func (self *Manager) ListWorkers() (workers []WorkerStatus, err error) {
    cache := makeWorkerCache(self, /* eager */ true)

    for _, worker := range self.workers {
        if workerStatus, err := cache.getStatus(worker, false); err != nil {
            return workers, err
        } else {
            workers = append(workers, workerStatus)
        }
    }

    return workers, nil
}

func (self *Manager) WorkerGet(configName string, instance string) (*WorkerStatus, error) {
    cache := makeWorkerCache(self, /* not eager */ false)

    workerName := fmt.Sprintf("%s:%s", configName, instance)

    if worker, found := self.workers[workerName]; !found {
        return nil, nil
    } else if workerStatus, err := cache.getStatus(worker, true); err != nil {
        return nil, err
    } else {
        return &workerStatus, nil
    }
}

func (self *Manager) WorkerDelete(configName string, instance string) error {
    for key, worker := range self.workers {
        if configName != "" && (worker.Config == nil || worker.Config.name != configName) {
            continue
        }
        if instance != "" && worker.Instance != instance {
            continue
        }

        if err := self.docker.Clean(worker.dockerID); err != nil {
            return err
        }

        delete(self.workers, key)
    }

    return nil
}
