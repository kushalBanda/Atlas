.PHONY: build run test

build:
	go build ./cmd/... ./pkg/...

run:
	go run ./cmd/atlas-server

test:
	go test ./cmd/... ./pkg/...
