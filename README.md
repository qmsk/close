# Cloud LOad StressEr

Cloud LOad StressEr is a collection of tools for generating network traffic and
collecting metrics from a network service that is the target for the load
testing. The package includes several utilities for testing different kind of
network protocols and a top-level manager for them. The utilities are called
workers. The manager (or the controller) consists of a daemon providing JSON
REST API, and a Web application communicating with the daemon and implementing a
user interface.

The workers configuration is communicated via Redis, where the currently running
configuration is stored. It provides the mechanism for changing worker
parameters without restarting a worker, by pushing a new configuration to
the Redis instance. The controller displays the current configuration and allows
to update it and re-configure workers on the fly.

The statistics from the workers are collected into InfluxDB, and the controller
implements basic reporting functionality of the statistics, generating plots.

## Install

The web controller needs its JS assets/dependencies installed:

  control-web/static $ npm install

## Workers

Workers are run as Docker containers, and perform the actual measurement /
load-generation work. Currently, there are three types of workers implemented:

* ICMP Ping
* DNS Ping
* UDP Sender

## Config

Workers subscribe to configuration updates from Redis. The controller shows the running configuration and pushes configuration updates to workers.

## Stats

Workers report statistics to InfluxDB. The controller reports the stats from InfluxDB.


