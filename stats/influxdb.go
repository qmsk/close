package stats

import (
    "fmt"
    influxdb "github.com/influxdb/influxdb/client/v2"
    "net"
    "net/url"
)

const INFLUXDB_PORT = "8086"
const INFLUXDB_DATABASE = "close"
const INFLUXDB_USER_AGENT = "close-stats"

type InfluxURL url.URL

func (self *InfluxURL) UnmarshalFlag(value string) error {
    if parseURL, err := url.Parse(value); err != nil {
        return err
    } else {
        switch parseURL.Scheme {
        case "http":
            *self = InfluxURL(*parseURL)
        default:
            return fmt.Errorf("Unsupported URL: %v", parseURL)
        }

        return nil
    }
}

func (self *InfluxURL) MarshalFlag() (string, error) {
    return self.String(), nil
}

func (self InfluxURL) String() string {
    return (*url.URL)(&self).String()
}

func (self InfluxURL) Empty() bool {
    return self.Scheme == ""
}

func (self InfluxURL) httpConfig() (httpConfig influxdb.HTTPConfig) {
    if _, port, err := net.SplitHostPort(self.Host); err != nil && port != "" {
        httpConfig.Addr = fmt.Sprintf("%s://%s", self.Scheme, self.Host)
    } else {
        httpConfig.Addr = fmt.Sprintf("%s://%s", self.Scheme, net.JoinHostPort(self.Host, INFLUXDB_PORT))
    }

    if self.User != nil {
        httpConfig.Username = self.User.Username()
        httpConfig.Password, _ = self.User.Password()
    }

    httpConfig.UserAgent = INFLUXDB_USER_AGENT

    return
}

func (self InfluxURL) Connect() (influxdb.Client, error) {
    switch self.Scheme {
    case "http", "https":
        return influxdb.NewHTTPClient(self.httpConfig())
    default:
        return nil, fmt.Errorf("Unsupported URL: %v", self)
    }
}

func (self InfluxURL) Database() string {
    if self.Path != "" {
        return self.Path
    } else {
        return INFLUXDB_DATABASE
    }
}

func (self InfluxURL) BatchPointsConfig() influxdb.BatchPointsConfig {
    return influxdb.BatchPointsConfig{
        Database:   self.Database(),
    }
}

func (self InfluxURL) Query(command string) influxdb.Query {
    return influxdb.Query{
        Command:    command,
        Database:   self.Database(),
    }
}
