# Cloud LOad StressEr

Test the performance of a service under load.

### Workers

Workers are run as Docker containers, and perform the actual measurement / load-generation work.

* ICMP Ping
* DNS Ping
* UDP Sender

### Config

Workers subscribe to configuration updates from Redis. The controller shows the running configuration and pushes configuration updates to workers.

### Stats

Workers report statistics to InfluxDB. The controller reports the stats from InfluxDB.

## Install

The web controller needs its JS assets/dependencies installed:

  control-web/static $ npm install

