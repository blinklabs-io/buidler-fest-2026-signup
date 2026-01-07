# Buidler Fest 2026 Signup CLI
.PHONY: build clean test lint run help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

build: ## Build the binary
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/buidlerfest ./cmd/buidlerfest

clean: ## Clean build artifacts
	rm -rf bin/ dist/

test: ## Run tests
	go test -v -race ./...

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

mod-tidy: ## Tidy go modules
	go mod tidy

run: build ## Build and run with preview profile
	./bin/buidlerfest --help

run-info: build ## Show signup info
	./bin/buidlerfest info --profile preview

install: ## Install binary to GOPATH/bin
	go install $(LDFLAGS) ./cmd/buidlerfest

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
