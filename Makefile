.PHONY: cmd build

cmd:
	godep go build -o build/conveyor ./cmd/conveyor
