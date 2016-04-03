FROM golang:1.6
MAINTAINER Eric Holmes <eric@remind101.com>

RUN go get -u github.com/kardianos/govendor

ENV DOCKER_HOST unix:///var/run/docker.sock

ADD . /go/src/github.com/remind101/conveyor
WORKDIR /go/src/github.com/remind101/conveyor
RUN govendor install ./cmd/conveyor

ENTRYPOINT ["/go/bin/conveyor"]
