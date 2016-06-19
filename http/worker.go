package http

import (
    "github.com/qmsk/close/config"
    "fmt"
    "net/http"
    "log"
    "github.com/qmsk/close/stats"
    "time"
    "net/url"
    "github.com/qmsk/close/worker"
)

type Config struct {
    URL     string     `json:"url" long:"url"`

    Interval    time.Duration   `json:"interval" long:"interval" default:"10s"`
}

func (config Config) Worker() (worker.Worker, error) {
    worker := &Worker{
        config:     config,
        resultChan: make(chan RequestStats),
    }

    if err := worker.apply(config); err != nil {
        return nil, err
    }

    return worker, nil
}

type RequestStats struct {
    url          string
    response     *http.Response

    start        time.Time

    RTT          time.Duration
    Error        error
}

func (rs RequestStats) StatsTime() time.Time {
    return rs.start
}
func (rs RequestStats) StatsID() stats.ID {
    return stats.ID{
        Type:       "http_request",
    }
}
func (rs RequestStats) Errors() int {
    if rs.Error != nil {
        return 1
    } else {
        return 0
    }
}
func (rs RequestStats) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // timing
        "rtt":      rs.RTT.Seconds(),

        // counters
        "errors":   rs.Errors(),
    }
}

func (rs RequestStats) String() string {
    if rs.Error != nil {
        return fmt.Sprintf("%v: err=%v", rs.url, rs.Error)
    } else if rs.RTT != 0 {
        return fmt.Sprintf("%v: rtt=%.2fms", rs.url,
            rs.RTT.Seconds() * 1000,
        )
    } else {
        return fmt.Sprintf("%v...", rs.url)
    }
}

type Worker struct {
    config      Config

    url         string

    statsChan   chan stats.Stats
    resultChan  chan RequestStats
}

func (worker *Worker) apply(config Config) error {
	if config.URL == "" {
        return fmt.Errorf("Empty URL: %#v", config.URL)
	}
    if parsedURL, err := url.Parse(config.URL); err != nil {
        return fmt.Errorf("Invalid URL: %#v", config.URL)
    } else if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
        worker.url = parsedURL.String()
    } else {
		worker.url = parsedURL.String()
	}

    worker.config = config

    return nil
}

func (worker *Worker) StatsWriter(statsWriter *stats.Writer) error {
    worker.statsChan = statsWriter.StatsWriter()

    return nil
}

func (worker *Worker) ConfigSub(configSub *config.Sub) error {
    if err := configSub.Register(worker.config); err != nil {
        return err
    }

    return nil
}

func (worker *Worker) request(request RequestStats) {
    if resp, err := http.Get(request.url); err != nil {
        request.Error = err
    } else {
		request.response = resp
		resp.Body.Close()
        request.RTT = time.Now().Sub(request.start)
    }

    // log.Printf("Request %v, response %v, %v, content length %v\n", request, request.response.Status, request.response.Proto, request.response.ContentLength)

    worker.resultChan <- request
}

func (worker *Worker) Run() error {
    intervalChan := time.Tick(worker.config.Interval)

    for {
        select {
        case tick := <-intervalChan:
            request := RequestStats{
                url:    worker.url,
                start:  tick,
            }

            go worker.request(request)

        case request := <-worker.resultChan:
            if request.Error != nil {
                log.Printf("Error: %v\n", request)
            }

            if worker.statsChan != nil {
                worker.statsChan <- request
            }
        }
    }
}
