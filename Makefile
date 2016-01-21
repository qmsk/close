export GOBIN=bin

bin: bin/control-web bin/close-worker bin/udp-recv

bin/%:
	go build -o $@ -v $+

bin/control-web: control-web/main.go
bin/close-worker: close-worker.go
bin/udp-recv: udp-recv.go
