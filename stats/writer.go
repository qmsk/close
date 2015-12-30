package stats

import (
    "fmt"
    influxdb "github.com/influxdb/influxdb/client/v2"
    "log"
    "os"
    "strings"
    "time"
)

const INFLUXDB_DATABASE = "close"
const INFLUXDB_USER_AGENT = "close-stats"
const INTERVAL = 1.0

type Config struct {
    InfluxDB            influxdb.HTTPConfig
    InfluxDBDatabase    string

    // The hostname this instance is running on to uniquely identify the source of measurements
    // If multiple instances of the same type are running on a single host, they must have a different hostname
    Hostname    string

    // Type of measurements being sent
    Type        string

    // The target being measured, intended to be aggregated from multiple instances of this type running on different hosts
    Instance    string

    // Collection interval
    Interval    float64 // seconds

    // Show stats on stdout
    Print       bool
}

// Wrap a statsd client to uniquely identify the measurements
type Writer struct {
    config          Config
    Interval        time.Duration

    influxdbClient      influxdb.Client
    writeChan           chan *influxdb.Point
}

func NewWriter(config Config) (*Writer, error) {
    if config.Hostname == "" {
        if hostname, err := os.Hostname(); err != nil {
            return nil, err
        } else {
            config.Hostname = hostname
        }
    }
    if strings.Contains(config.Hostname, ".") {
        log.Printf("statsd-hostname: stripping domain\n")
        config.Hostname = strings.Split(config.Hostname, ".")[0]
    }

    if config.Type == "" {
        panic("Invalid stats-type")
    }

    if config.InfluxDB.UserAgent == "" {
        config.InfluxDB.UserAgent = INFLUXDB_USER_AGENT
    }

    self := &Writer{
        config:     config,
        Interval:   time.Duration(config.Interval * float64(time.Second)),
        writeChan:  make(chan *influxdb.Point),
    }

    if influxdbClient, err := influxdb.NewHTTPClient(config.InfluxDB); err != nil {
        return nil, err
    } else {
        self.influxdbClient = influxdbClient
    }

    // start writing
    go self.write()

    return self, nil
}

func (self *Writer) String() string {
    return fmt.Sprintf("%v/%v/%v?hostname=%v&instance=%v", self.config.InfluxDB.Addr, self.config.InfluxDBDatabase, self.config.Type, self.config.Hostname, self.config.Instance)
}

func (self *Writer) write() {
    // TODO: batch up from chan?
    for point := range self.writeChan {
        points, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{Database: self.config.InfluxDBDatabase})
        if err != nil {
            log.Printf("stats.Writer %v: InfluxDB points error: %v\n", self, err)
            continue
        }

        points.AddPoint(point)

        if err := self.influxdbClient.Write(points); err != nil {
            log.Printf("stats.Writer %v: InfluxDB write error: %v\n", self, err)
        }
    }
}

func (self *Writer) Write(instance string, timestamp time.Time, fields map[string]interface{}) {
    log.Printf("stats.Writer %v: write %v@%v %v\n", self, instance, timestamp, fields)

    if instance == "" {
        instance = self.config.Instance
    }

    tags := map[string]string{
        "hostname":   self.config.Hostname,
        "instance":   instance,
    }

    if point, err := influxdb.NewPoint(self.config.Type, tags, fields, timestamp); err != nil {
        log.Printf("stats.Writer %v: InfluxDB point error: %v\n", self, err)
    } else {
        self.writeChan <- point
    }
}

func (self *Writer) WriteStats(stats Stats) {
    if self.config.Print {
        fmt.Printf("%v\n", stats)
    }
    self.Write(stats.StatsInstance(), stats.StatsTime(), stats.StatsFields())
}

func (self *Writer) writeFrom(statsChan chan Stats) {
    log.Printf("stats.Writer %v: writeFrom %v...\n", self, statsChan)

    for stats := range statsChan {
        self.WriteStats(stats)
    }
}

// Start gathering stats 
func (self *Writer) WriteFrom(statsSource StatsSource) {
    go self.writeFrom(statsSource.GiveStats(self.Interval))
}
