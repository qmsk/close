package stats

import (
    "fmt"
    influxdb "github.com/influxdb/influxdb/client/v2"
    "log"
    "os"
)

type ReaderConfig struct {
    InfluxDB        influxdb.HTTPConfig
    Database        string
}

type Reader struct {
    config          ReaderConfig
    log             *log.Logger

    influxdbClient      influxdb.Client
}

func NewReader(config ReaderConfig) (*Reader, error) {
    if config.InfluxDB.UserAgent == "" {
        config.InfluxDB.UserAgent = INFLUXDB_USER_AGENT
    }

    self := &Reader{
        config:     config,
        log:        log.New(os.Stderr, fmt.Sprintf("stats.Reader: "), 0),
    }

    if err := self.init(config); err != nil {
        return nil, err
    }

    return self, nil
}

func (self *Reader) init(config ReaderConfig) error {
    if influxdbClient, err := influxdb.NewHTTPClient(config.InfluxDB); err != nil {
        return err
    } else {
        self.influxdbClient = influxdbClient
    }

    return nil
}

func (self *Reader) String() string {
    return fmt.Sprintf("%v/%v/", self.config.InfluxDB.Addr, self.config.Database)
}

// List types
func (self *Reader) ListTypes() (types []string, err error) {
    response, err := self.influxdbClient.Query(influxdb.NewQuery("SHOW MEASUREMENTS", self.config.Database, ""))
    if err != nil {
        return nil, err
    }
    if response.Error() != nil {
        return nil, response.Error()
    }

    for _, result := range response.Results {
        for _, row := range result.Series {
            if row.Name != "measurements" {
                continue
            }
            for colIndex, colName := range row.Columns {
                if colName != "name" {
                    continue
                }

                for _, rowValues := range row.Values {
                    rowValue := rowValues[colIndex]

                    switch value := rowValue.(type) {
                    case string:
                        types = append(types, value)
                    default:
                         return nil, fmt.Errorf("Invalid value for measurements name: %#v", rowValue)
                     }
                }
            }
        }
    }

    return types, nil
}

type SeriesMeta struct {
    Type        string  `json:"type"`
    Hostname    string  `json:"hostname"`
    Instance    string  `json:"instance"`
}

func (self *Reader) ListSeries(filter SeriesMeta) (seriesList []SeriesMeta, err error) {
    query := influxdb.Query{Database: self.config.Database}

    query.Command = "SHOW SERIES"

    if filter.Type != "" {
        // XXX: holy SQL injection batman
        query.Command += fmt.Sprintf(" FROM \"%s\"", filter.Type)

        if filter.Hostname != "" || filter.Instance != "" {
            query.Command += " WHERE"

            if filter.Hostname != "" {
                query.Command += fmt.Sprintf(" hostname='%s'", filter.Hostname)
            }
            if filter.Instance != "" {
                query.Command += fmt.Sprintf(" instance='%s'", filter.Instance)
            }
        }
    }

    self.log.Printf("%v\n", query.Command)

    response, err := self.influxdbClient.Query(query)
    if err != nil {
        return nil, err
    }
    if response.Error() != nil {
        return nil, response.Error()
    }

    for _, result := range response.Results {
        for _, row := range result.Series {
            for _, rowValues := range row.Values {
                series := SeriesMeta{Type: row.Name}

                for colIndex, colName := range row.Columns {
                    fieldValue := rowValues[colIndex]

                    stringValue, _ := fieldValue.(string)

                    if colName == "hostname" {
                        series.Hostname = stringValue
                    } else if colName == "instance" {
                        series.Instance = stringValue
                    } else {
                        // ignore
                        continue
                    }
                }

                seriesList = append(seriesList, series)
            }
        }
    }

    return seriesList, nil
}
