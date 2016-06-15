export GOBIN=bin

BUILD_TAGS=

bin: bin/control-web bin/close-worker bin/udp-recv

bin/%: cmd/%
	go install -v -tags=${BUILD_TAGS} ./$<

bin/control-web:
bin/close-worker:
bin/udp-recv:
