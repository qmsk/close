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

The package installation requires [Go](https://golang.org/) utilities. To build
and install the whole package into $GOPATH/bin

    go get github.com/qmsk/close

Another way is to clone the repository and use a provided Makefile that will
build the utilities locally, without installing them.

Workers and the controller can be installed separately:

    go get github.com/qmsk/workers
    go get github.com/qmsk/control-web

The web controller is a Javascript application and it requires its JS
assets/dependencies installed via [Node.js](https://nodejs.org/en/) package
manager.

    git clone github.com/qmsk/close
    cd close/control-web/static
    npm install

There is a Docker build file provided as well, so

    make
    docker build .

will produce a Docker image with all the binaries installed.

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


