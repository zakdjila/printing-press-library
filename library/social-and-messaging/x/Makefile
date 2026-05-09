.PHONY: build test lint install clean

build:
	go build -o bin/x-pp-cli ./cmd/x-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/x-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/x-pp-mcp ./cmd/x-pp-mcp

install-mcp:
	go install ./cmd/x-pp-mcp

build-all: build build-mcp
