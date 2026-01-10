.PHONY: build test test-cover lint clean install help

# Binary name
BINARY := x402

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: build

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/x402

## install: Install the binary to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/x402

## test: Run all tests
test:
	go test ./...

## test-cover: Run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-race: Run tests with race detector
test-race:
	go test -race ./...

## lint: Run linter
lint:
	golangci-lint run

## fmt: Format code
fmt:
	go fmt ./...
	gofmt -s -w .

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy and verify dependencies
tidy:
	go mod tidy
	go mod verify

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## release-dry: Test goreleaser locally (dry run)
release-dry:
	goreleaser release --snapshot --clean

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
