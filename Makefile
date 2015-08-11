.PHONY: cmd build

cmd:
	godep go build -o build/conveyor ./cmd/conveyor

build:
	docker build -t remind101/conveyor .

ami:
	packer build -var "sha=$(shell git rev-parse HEAD)" packer.json
