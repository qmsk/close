package dns

import (
    "github.com/qmsk/close/config"
    "github.com/miekg/dns"
    "fmt"
    "log"
    "net"
    "github.com/qmsk/close/stats"
    "strings"
    "time"
    "github.com/qmsk/close/worker"
)

type Config struct {
    Network     string          `json:"network" long:"dns-network" default:"udp"`
    Timeout     time.Duration   `json:"timeout" long:"dns-timeout" default:"2s"`

    Server      string          `json:"server" long:"server"`
    Interval    time.Duration   `json:"interval" long:"interval" default:"1s"`

    QueryType   string          `json:"query_type" long:"query-type" default:"A"`
    QueryName   string          `json:"query_name" long:"query-name" default:"www.google.com"`
}

func (config Config) Worker() (worker.Worker, error) {
    worker := &Worker{
        config:     config,
        resultChan: make(chan Query),
    }

    if err := worker.apply(config); err != nil {
        return nil, err
    }

    return worker, nil
}

type Query struct {
    server      string
    msg         *dns.Msg        // updated on recv

    // send
    start       time.Time

    // recv
    RTT         time.Duration
    Error       error
}

func (q Query ) StatsTime() time.Time {
    return q.start
}
func (q Query) StatsID() stats.ID {
    return stats.ID{
        Type:       "dns_query",
    }
}
func (q Query) Errors() int {
    if q.Error != nil {
        return 1
    } else {
        return 0
    }
}
func (q Query) Answers() int {
    return len(q.msg.Answer)
}
func (q Query) StatsFields() map[string]interface{} {
    return map[string]interface{}{
        // timing
        "rtt":      q.RTT.Seconds(),

        // counters
        "errors":   q.Errors(),
        "answers":  q.Answers(),
    }
}

func (q Query) String() string {
    question := fmt.Sprintf("%v %v", q.msg.Question[0].Name, dns.Type(q.msg.Question[0].Qtype))

    if q.Error != nil {
        return fmt.Sprintf("%v: err=%v", question, q.Error)
    } else if q.RTT != 0 {
        return fmt.Sprintf("%v: rtt=%.2fms answers=%d", question,
            q.RTT.Seconds() * 1000,
            q.Answers(),
        )
    } else {
        return fmt.Sprintf("%v...", question)
    }
}

type Worker struct {
    config      Config

    server      string
    queryType   dns.Type
    queryName   string

    dnsClient   dns.Client

    statsChan   chan stats.Stats
    resultChan  chan Query
}

func (worker *Worker) apply(config Config) error {
    worker.dnsClient = dns.Client{
        Net:    config.Network,
        DialTimeout:        config.Timeout,
        ReadTimeout:        config.Timeout,
        WriteTimeout:       config.Timeout,
    }

    if config.Server == "" {
        return fmt.Errorf("Invalid Server: %#v", config.Server)
    } else if host, port, err := net.SplitHostPort(config.Server); err == nil && host != "" && port != "" {
        worker.server = net.JoinHostPort(host, port)
    } else {
        worker.server = net.JoinHostPort(config.Server, "53")
    }

    if queryType, exists := dns.StringToType[strings.ToUpper(config.QueryType)]; !exists {
        return fmt.Errorf("Invalid QueryType: %v", config.QueryType)
    } else {
        worker.queryType = dns.Type(queryType)
    }

    if config.QueryName == "" {
        return fmt.Errorf("Invalid QueryName: %v", config.QueryName)
    } else {
        worker.queryName = dns.Fqdn(config.QueryName)
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

func (worker *Worker) generateQuery() Query {
    msg := new(dns.Msg).SetQuestion(worker.queryName, uint16(worker.queryType))

    return Query{
        server: worker.server,
        msg:    msg,
    }
}

func (worker *Worker) query(query Query) {
    if msg, rtt, err := worker.dnsClient.Exchange(query.msg, query.server); err != nil {
        query.Error = err
    } else {
        query.msg = msg
        query.RTT = rtt
    }

    // log.Printf("Query %v\n%v\n", query, query.msg)

    worker.resultChan <- query
}

func (worker *Worker) Run() error {
    intervalChan := time.Tick(worker.config.Interval)

    for {
        select {
        case tick := <-intervalChan:
            query := worker.generateQuery()
            query.start = tick

            go worker.query(query)

        case query := <-worker.resultChan:
            if query.Error != nil {
                log.Printf("Error: %v\n", query)
            }

            if worker.statsChan != nil {
                worker.statsChan <- query
            }
        }
    }
}
