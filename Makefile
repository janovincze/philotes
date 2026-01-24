# Philotes Makefile
# Run 'make help' to see all available targets

.PHONY: help build test lint fmt clean run docker-up docker-down generate install-tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Binary names
BINARY_WORKER=philotes-worker
BINARY_API=philotes-api
BINARY_CLI=philotes

# Build directories
BUILD_DIR=./bin
CMD_DIR=./cmd

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Docker compose file
DOCKER_COMPOSE=docker compose -f deployments/docker/docker-compose.yml

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Philotes - CDC Pipeline Platform"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build all binaries
build: build-worker build-api build-cli

## build-worker: Build the CDC worker
build-worker:
	@echo "Building $(BINARY_WORKER)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_WORKER) $(CMD_DIR)/philotes-worker

## build-api: Build the API server
build-api:
	@echo "Building $(BINARY_API)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_API) $(CMD_DIR)/philotes-api

## build-cli: Build the CLI tool
build-cli:
	@echo "Building $(BINARY_CLI)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI) $(CMD_DIR)/philotes-cli

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-short: Run tests without race detector
test-short:
	@echo "Running tests (short)..."
	$(GOTEST) -v -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

## lint-fix: Run linter and fix issues
lint-fix:
	@echo "Running linter with auto-fix..."
	$(GOLINT) run --fix ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## tidy: Tidy and verify Go modules
tidy:
	@echo "Tidying Go modules..."
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## run-api: Run the API server locally
run-api: build-api
	@echo "Starting API server..."
	$(BUILD_DIR)/$(BINARY_API)

## run-worker: Run the CDC worker locally
run-worker: build-worker
	@echo "Starting CDC worker..."
	$(BUILD_DIR)/$(BINARY_WORKER)

## docker-up: Start development environment
docker-up:
	@echo "Starting development environment..."
	$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "Services started:"
	@echo "  PostgreSQL:  localhost:5432 (user: philotes, pass: philotes)"
	@echo "  PostgreSQL (source): localhost:5433 (user: source, pass: source)"
	@echo "  MinIO:       http://localhost:9000 (user: minioadmin, pass: minioadmin)"
	@echo "  MinIO Console: http://localhost:9001"
	@echo "  Lakekeeper:  http://localhost:8181"
	@echo "  Prometheus:  http://localhost:9090"
	@echo "  Grafana:     http://localhost:3000 (user: admin, pass: admin)"

## docker-down: Stop development environment
docker-down:
	@echo "Stopping development environment..."
	$(DOCKER_COMPOSE) down

## docker-clean: Stop development environment and remove volumes
docker-clean:
	@echo "Stopping development environment and removing volumes..."
	$(DOCKER_COMPOSE) down -v

## docker-logs: Show logs from development environment
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-ps: Show running containers
docker-ps:
	$(DOCKER_COMPOSE) ps

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	@echo "Tools installed successfully"

## generate: Generate code (OpenAPI, mocks, etc.)
generate:
	@echo "Generating code..."
	$(GOCMD) generate ./...

## setup-hooks: Set up git hooks
setup-hooks:
	@echo "Setting up git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make lint fmt' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

## check: Run all checks (lint, vet, test)
check: lint vet test

## all: Build and test
all: clean lint vet test build
