.PHONY: build test lint run clean

BINARY=slk
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -trimpath -o $(BUILD_DIR)/$(BINARY) ./cmd/slk

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

run: build
	./$(BUILD_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
