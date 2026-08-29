.PHONY: build run test frontend

frontend:
	NODE_ENV=development npm --prefix frontend install
	npm --prefix frontend run build

build: frontend
	go build ./cmd/... ./pkg/...

run: frontend
	go run ./cmd/atlas-server

test:
	go test ./cmd/... ./pkg/...
	NODE_ENV=development npm --prefix frontend install
	npm --prefix frontend run test
