all: test

prepare:
	# dependencies
	go get code.google.com/p/go-uuid/uuid
	go get github.com/shirou/gopsutil/load
	# needed for `make fmt`
	go get golang.org/x/tools/cmd/goimports
	# linters
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	# needed for `make cover`
	go get golang.org/x/tools/cmd/cover
	@echo Now you should be ready to run "make"

test:
	@go test -parallel 4 ./...

# goimports produces slightly different formatted code from go fmt
fmt:
	find . -name "*.go" -exec goimports -w {} \;

lint:
	gometalinter

cover:
	go test -cover -coverprofile cover.out
	go tool cover -html=cover.out

.PHONY: all prepare test fmt lint cover
