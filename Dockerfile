FROM golang:1.4.2
MAINTAINER Eric Holmes <eric@remind101.com>

RUN go get github.com/tools/godep

ENV DOCKER_HOST unix:///var/run/docker.sock

ADD . /go/src/github.com/remind101/conveyor
WORKDIR /go/src/github.com/remind101/conveyor
RUN godep go install ./cmd/conveyor

CMD ["/go/bin/conveyor"]
