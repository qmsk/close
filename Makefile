export GOBIN=bin

bin: bin/control-web bin/icmp-ping bin/udp-send bin/udp-recv

bin/%:
	go build -o $@ -v $+

bin/control-web: control-web/*.go
bin/icmp-ping: icmp-ping/*.go
bin/udp-recv: udp-recv/*.go
bin/udp-send: udp-send/*.go
