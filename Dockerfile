FROM golang:1.4.2
MAINTAINER Eric Holmes <eric@remind101.com>

ADD . /go/src/github.com/remind101/conveyor
WORKDIR /go/src/github.com/remind101/conveyor
RUN go get github.com/tools/godep && godep go install ./cmd/conveyor

CMD ["/go/bin/conveyor"]
