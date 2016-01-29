export GOBIN=bin

BUILD_TAGS=

bin: bin/control-web bin/close-worker bin/udp-recv

bin/%:
	go build -o $@ -v -tags=${BUILD_TAGS} $+

bin/control-web: control-web/main.go control-web/debug.go
bin/close-worker: close-worker.go
bin/udp-recv: udp-recv.go
