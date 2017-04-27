.PHONY: cmd build test-payload

cmd:
	go build -o build/conveyor ./cmd/conveyor

build:
	docker build -t remind101/conveyor .

ami:
	packer build -var "sha=$(shell git rev-parse HEAD)" packer.json

ci: test

test:
	go test -short $(shell go list ./... | grep -v /vendor/)

test-payload:
	curl -H "X-GitHub-Event: push" -X POST http://$(shell docker-machine ip default):8080 -d '{"ref":"refs/heads/master","head_commit": {"id":"f2cb43dea101ceaa548746dfd959132e91afb807"},"repository":{"full_name":"remind101/codebuild_test"}}'

bootstrap: database .env

.env: .env.sample
	cp .env.sample .env

bindata.go: db/migrations/*.sql
	go-bindata -pkg conveyor -o bindata.go db/migrations/

database::
	dropdb conveyor || true
	createdb conveyor || true
	dropdb conveyor_api || true
	createdb conveyor_api || true

schema.json: meta.json schemata/*
	bundle exec prmd combine --meta meta.json schemata/ > schema.json

schema.md: schema.json
	bundle exec prmd doc schema.json > schema.md

client/conveyor/conveyor.go: schema.json
	schematic schema.json > client/conveyor/conveyor.go

schema:: schema.md client/conveyor/conveyor.go

lint:
	golint $(shell go list ./... | grep -v /vendor/) | grep -v -E 'exported|comment'

lint-all:
	golint $(shell go list ./... | grep -v /vendor/)
