FROM golang:1.7.6
MAINTAINER Eric Holmes <eric@remind101.com>

ENV DOCKER_HOST unix:///var/run/docker.sock

ADD . /go/src/github.com/remind101/conveyor
WORKDIR /go/src/github.com/remind101/conveyor
RUN go install ./cmd/conveyor

ENTRYPOINT ["/go/bin/conveyor"]
