package stats

import (
    "fmt"
    influxdb "github.com/influxdb/influxdb/client/v2"
    "log"
    "os"
    "strings"
    "time"
)

type WriterOptions struct {
    InfluxURL   InfluxURL       `long:"influxdb-url" value-name:"http://[USER:[PASSWORD]@]HOST[:PORT]/DATABASE" env:"INFLUXDB_URL"`

    Hostname    string          `long:"stats-hostname" env:"HOSTNAME"`

    Instance    string          `long:"stats-instance" env:"CLOSE_INSTANCE"`

    // Collection interval
    Interval    time.Duration   `long:"stats-interval" value-name:"SECONDS" default:"1s"`

    // Show stats on stdout
    Print       bool            `long:"stats-print"`
}

func (self WriterOptions) Empty() bool {
    return self.InfluxURL.Empty()
}

// Wrap a statsd client to uniquely identify the measurements
type Writer struct {
    options             WriterOptions

    influxdbClient      influxdb.Client
    writeChan           chan *influxdb.Point
}

func NewWriter(options WriterOptions) (*Writer, error) {
    if options.Hostname != "" {

    } else if hostname, err := os.Hostname(); err != nil {
        return nil, err
    } else {
        options.Hostname = hostname
    }

    if strings.Contains(options.Hostname, ".") {
        log.Printf("statsd-hostname: stripping domain\n")
        options.Hostname = strings.Split(options.Hostname, ".")[0]
    }

    self := &Writer{
        options:     options,
        writeChan:  make(chan *influxdb.Point),
    }

    if influxdbClient, err := options.InfluxURL.Connect(); err != nil {
        return nil, err
    } else {
        self.influxdbClient = influxdbClient
    }

    // start writing
    go self.writer()

    return self, nil
}

func (self *Writer) String() string {
    return fmt.Sprintf("%v?hostname=%v", self.options.InfluxURL, self.options.Hostname)
}

func (self *Writer) writer() {
    // TODO: batch up from chan?
    for point := range self.writeChan {
        points, err := influxdb.NewBatchPoints(self.options.InfluxURL.BatchPointsConfig())
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

func (self *Writer) write(stats Stats) {
    if self.options.Print {
        fmt.Printf("%v\n", stats)
    }

    id := stats.StatsID()
    if id.Hostname == "" {
        id.Hostname = self.options.Hostname
    }

    if id.Instance == "" {
        id.Instance = self.options.Instance
    }

    tags := map[string]string{
        "hostname":   id.Hostname,
        "instance":   id.Instance,
    }

    if point, err := influxdb.NewPoint(id.Type, tags, stats.StatsFields(), stats.StatsTime()); err != nil {
        log.Printf("stats.Writer %v: InfluxDB point error: %v\n", self, err)
    } else {
        self.writeChan <- point
    }
}

func (self *Writer) statsWriter(intervalChan <-chan time.Time, statsChan chan Stats) {
    log.Printf("stats.Writer %v: writeFrom %v @%v...\n", self, statsChan, intervalChan)

    for stats := range statsChan {
        self.write(stats)

        // only read stats every interval tick, or continuously if tickChan is nil
        if intervalChan != nil {
            <-intervalChan
        }
    }
}

func (self *Writer) Interval() time.Duration {
    return self.options.Interval
}

func (self *Writer) IntervalTick() (<-chan time.Time) {
    return time.Tick(self.options.Interval)
}

// Return a channel which normally blocks on writes, but accepts a write every tick-interval
// XXX: this does not guarantee any minimum interval for the initial stats cycle, and is thus mostly broken..
func (self *Writer) IntervalStatsWriter() (chan Stats) {
    tickChan := self.IntervalTick()
    statsChan := make(chan Stats)

    go self.statsWriter(tickChan, statsChan)

    return statsChan
}

// Return a channel which continously accepts stats writes
func (self *Writer) StatsWriter() (chan Stats) {
    tickChan := time.Tick(self.options.Interval)
    statsChan := make(chan Stats)

    go self.statsWriter(tickChan, statsChan)

    return statsChan
}
