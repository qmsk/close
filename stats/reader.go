package stats

import (
    "fmt"
    influxdb "github.com/influxdb/influxdb/client/v2"
    "encoding/json"
    "log"
    "os"
    "strings"
    "time"
)

type ReaderOptions struct {
    InfluxURL   InfluxURL       `long:"influxdb-url" value-name:"http://[USER:[PASSWORD]@]HOST[:PORT]/DATABASE" env:"INFLUXDB_URL"`
}

func (self ReaderOptions) Empty() bool {
    return self.InfluxURL.Empty()
}

type Reader struct {
    options         ReaderOptions
    log             *log.Logger

    influxdbClient      influxdb.Client
}

func NewReader(options ReaderOptions) (*Reader, error) {
    self := &Reader{
        options:    options,
        log:        log.New(os.Stderr, fmt.Sprintf("stats.Reader: "), 0),
    }

    if influxdbClient, err := options.InfluxURL.Connect(); err != nil {
        return nil, err
    } else {
        self.influxdbClient = influxdbClient
    }

    return self, nil
}

func (self *Reader) String() string {
    return fmt.Sprintf("%v/%v/", self.options.InfluxURL)
}

/* Static list of InfluxDB measurements, and their fields */
type SeriesMeta struct {
    Type    string      `json:"type"`
    Fields  []string    `json:"fields"`
}

func (self *Reader) ListTypes() (metas []SeriesMeta, err error) {
    query := self.options.InfluxURL.Query("SHOW FIELD KEYS")

    response, err := self.influxdbClient.Query(query)
    if err != nil {
        return nil, err
    }
    if response.Error() != nil {
        return nil, response.Error()
    }

    for _, result := range response.Results {
        for _, row := range result.Series {
            meta := SeriesMeta{
                Type:   row.Name,
            }

            for _, rowValues := range row.Values {
                for colIndex, colName := range row.Columns {
                    fieldValue := rowValues[colIndex]
                    stringValue, _ := fieldValue.(string)

                    if colName == "fieldKey" {
                        meta.Fields = append(meta.Fields, stringValue)
                    } else {
                        // ignore
                        continue
                    }
                }
            }

            metas = append(metas, meta)
        }
    }

    return metas, nil
}

/* Dynamic list of InfluxDB measurement-series. Each of these series may have multiple fields */
type SeriesKey struct {
    Type        string  `json:"type"`
    Hostname    string  `json:"hostname,omitempty"`
    Instance    string  `json:"instance,omitempty"`
}

func (self *Reader) ListSeries(filter SeriesKey) (seriesList []SeriesKey, err error) {
    query := self.options.InfluxURL.Query("SHOW SERIES")

    if filter.Type != "" {
        // XXX: holy SQL injection batman
        query.Command += fmt.Sprintf(" FROM \"%s\"", filter.Type)

        if filter.Hostname != "" || filter.Instance != "" {
            query.Command += " WHERE"

            if filter.Hostname != "" {
                query.Command += fmt.Sprintf(" hostname='%s'", filter.Hostname)
            }
            if filter.Hostname != "" && filter.Instance != "" {
                query.Command += " AND"
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
                series := SeriesKey{Type: row.Name}

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

/* Temporal InfluxDB measurement-series-data */
type SeriesPoint struct {
    Time        time.Time   `json:"time"`
    Value       float64     `json:"value"`
}

type SeriesData struct {
    SeriesKey
    Field       string          `json:"field"`

    Points      []SeriesPoint   `json:"points"`
}

func (self SeriesData) String() string {
    return fmt.Sprintf("%s/%s@%s:%s", self.Type, self.Field, self.Hostname, self.Instance)
}

// Get full time-series data for given series's fields over given duration
func (self *Reader) GetSeries(series SeriesKey, fields []string, duration time.Duration) (dataList []SeriesData, err error) {
    var queryFields string

    if fields == nil || len(fields) == 0 {
        queryFields = "*"
    } else {
        queryFields = strings.Join(fields, ", ")
    }

    query := self.options.InfluxURL.Query(fmt.Sprintf("SELECT %s FROM \"%s\" WHERE time > now() - %vs", queryFields, series.Type, duration.Seconds()))

    if series.Hostname != "" {
        query.Command += fmt.Sprintf(" AND hostname='%s'", series.Hostname)
    }
    if series.Instance != "" {
        query.Command += fmt.Sprintf(" AND instance='%s'", series.Instance)
    }

    query.Command += " GROUP BY hostname, instance"

    self.log.Printf("%v\n", query.Command)

    response, err := self.influxdbClient.Query(query)
    if err != nil {
        return nil, err
    }
    if response.Error() != nil {
        return nil, response.Error()
    }

    for _, result := range response.Results {
        for _, series := range result.Series {
            for colIndex, colName := range series.Columns[1:] {
                seriesData := SeriesData{
                    SeriesKey: SeriesKey{
                        Type:       series.Name,
                        Hostname:   series.Tags["hostname"],
                        Instance:   series.Tags["instance"],
                    },
                    Field:     colName,
                }

                for _, rowValues := range series.Values {
                    fieldValue := rowValues[1+colIndex]

                    point := SeriesPoint{}

                    if stringValue, ok := rowValues[0].(string); !ok {
                        return nil, fmt.Errorf("invalid time value: %#v", rowValues[0])
                    } else if timeValue, err := time.Parse(time.RFC3339, stringValue); err != nil {
                        return nil, fmt.Errorf("invalid time value %v: %v", stringValue, err)
                    } else {
                        point.Time = timeValue
                    }

                    if jsonValue, ok := fieldValue.(json.Number); !ok {
                        return nil, fmt.Errorf("invalid value for %v(%v.%v): %#v", colName, series.Name, colName, fieldValue)
                    } else if floatValue, err := jsonValue.Float64(); err != nil {
                        return nil, err
                    } else {
                        point.Value = floatValue
                    }

                    seriesData.Points = append(seriesData.Points, point)
                }

                dataList = append(dataList, seriesData)
            }
        }
    }

    return dataList, nil
}

/* Summarized InfluxDB measurement-series-data for table */
type SeriesTab struct {
    Time        time.Time   `json:"time"`

    Mean        float64     `json:"mean"`
    Min         float64     `json:"min"`
    Max         float64     `json:"max"`
    Last        float64     `json:"last"`
}

type SeriesStats struct {
    SeriesKey
    Field       string      `json:"field"`

    SeriesTab
}

func (self SeriesStats) String() string {
    return fmt.Sprintf("%s/%s@%s:%s", self.Type, self.Field, self.Hostname, self.Instance)
}

func (self *Reader) GetStats(series SeriesKey, field string, duration time.Duration) (statsList []SeriesStats, err error) {
    query := self.options.InfluxURL.Query(fmt.Sprintf("SELECT MEAN(\"%s\") AS mean, MIN(\"%s\") AS min, MAX(\"%s\") AS max, LAST(\"%s\") AS last FROM \"%s\" WHERE time > now() - %vs", field, field, field, field, series.Type, duration.Seconds()))

    if series.Hostname != "" {
        query.Command += fmt.Sprintf(" AND hostname='%s'", series.Hostname)
    }
    if series.Instance != "" {
        query.Command += fmt.Sprintf(" AND instance='%s'", series.Instance)
    }

    query.Command += " GROUP BY hostname, instance"

    self.log.Printf("%v\n", query.Command)

    response, err := self.influxdbClient.Query(query)
    if err != nil {
        return nil, err
    }
    if response.Error() != nil {
        return nil, response.Error()
    }

    for _, result := range response.Results {
        for _, series := range result.Series {
            for _, rowValues := range series.Values {
                stats := SeriesStats{
                    SeriesKey: SeriesKey{
                        Type:       series.Name,
                        Hostname:   series.Tags["hostname"],
                        Instance:   series.Tags["instance"],
                    },
                    Field: field,
                }

                for colIndex, colName := range series.Columns {
                    fieldValue := rowValues[colIndex]

                    if colName == "time" {
                        stringValue, ok := fieldValue.(string)
                        if !ok {
                            return nil, fmt.Errorf("invalid time value: %#v", fieldValue)
                        }

                        timeValue, err := time.Parse(time.RFC3339, stringValue)
                        if err != nil {
                            return nil, fmt.Errorf("invalid time value %v: %v", stringValue, err)
                        }

                        stats.Time = timeValue

                    } else {
                        var value float64

                        if jsonValue, ok := fieldValue.(json.Number); !ok {
                            return nil, fmt.Errorf("invalid value for %v(%v.%v): %T(%#v)", colName, series.Name, field, fieldValue, fieldValue)
                        } else if floatValue, err := jsonValue.Float64(); err != nil {
                            return nil, err
                        } else {
                            value = floatValue

                        }

                        switch colName {
                        case "mean":
                            stats.Mean = value
                        case "min":
                            stats.Min = value
                        case "max":
                            stats.Max = value
                        case "last":
                            stats.Last = value
                        }
                    }
                }

                statsList = append(statsList, stats)
            }
        }
    }

    return statsList, nil
}
