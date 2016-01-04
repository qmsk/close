FROM golang:1.5

COPY . /go/src/close
RUN go get -v \
    close/latency \
    close/udp-send \
    close/udp-recv 

