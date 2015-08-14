.PHONY: cmd build test-payload

cmd:
	godep go build -o build/conveyor ./cmd/conveyor

build:
	docker build -t remind101/conveyor .

ami:
	packer build -var "sha=$(shell git rev-parse HEAD)" packer.json

test-payload:
	curl -H "X-GitHub-Event: push" -X POST http://$(shell boot2docker ip):8080 -d '{"ref":"refs/heads/master","head_commit": {"id":"827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"},"repository":{"full_name":"remind101/acme-inc"}}'
