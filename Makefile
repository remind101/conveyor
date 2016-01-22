.PHONY: cmd build test-payload

cmd:
	godep go build -o build/conveyor ./cmd/conveyor

build:
	docker build -t remind101/conveyor .

ami:
	packer build -var "sha=$(shell git rev-parse HEAD)" packer.json

ci: test diffbindata

test:
	godep go test -short ./...

diffbindata:
	@echo "Testing if bindata.go has changed"
	@test -z "$(shell git diff --name-only | grep bindata)"

test-payload:
	curl -H "X-GitHub-Event: push" -X POST http://$(shell docker-machine ip default):8080 -d '{"ref":"refs/heads/master","head_commit": {"id":"827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"},"repository":{"full_name":"remind101/acme-inc"}}'

bootstrap: database .env
	$(MAKE) -C builder/docker bootstrap

.env: .env.sample
	cp .env.sample .env

bindata.go: db/migrations/*.sql
	go-bindata -pkg conveyor -o bindata.go db/migrations/

database:: bindata.go
	dropdb conveyor || true
	createdb conveyor || true
