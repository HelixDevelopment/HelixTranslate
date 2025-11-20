.PHONY: all build clean test test-unit test-integration test-e2e test-performance test-stress run-cli run-server install deps fmt lint docker-build docker-run help

# Variables
BINARY_CLI=translator
BINARY_SERVER=translator-server
BUILD_DIR=build
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-w -s"

# Default target
all: deps build test

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build CLI and server binaries"
	@echo "  build-cli          - Build CLI binary only"
	@echo "  build-server       - Build server binary only"
	@echo "  clean              - Remove build artifacts"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests"
	@echo "  test-integration   - Run integration tests"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  test-performance   - Run performance tests"
	@echo "  test-stress        - Run stress tests"
	@echo "  run-cli            - Run CLI application"
	@echo "  run-server         - Run server application"
	@echo "  install            - Install binaries to GOPATH/bin"
	@echo "  deps               - Download dependencies"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  generate-certs     - Generate self-signed TLS certificates"

# Build targets
build: build-cli build-server

build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI) ./cmd/cli

build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/server

# Clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean

# Test targets
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	$(GO) test -v -race -coverprofile=coverage-unit.out ./...

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -race -tags=integration ./test/integration/...

test-e2e:
	@echo "Running E2E tests..."
	$(GO) test -v -tags=e2e ./test/e2e/...

test-performance:
	@echo "Running performance tests..."
	$(GO) test -v -bench=. -benchmem -tags=performance ./test/performance/...

test-stress:
	@echo "Running stress tests..."
	$(GO) test -v -timeout=30m -tags=stress ./test/stress/...

test-coverage:
	@echo "Generating coverage report..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run targets
run-cli:
	$(GO) run ./cmd/cli/main.go

run-server:
	$(GO) run ./cmd/server/main.go

# Install
install:
	@echo "Installing binaries..."
	$(GO) install $(LDFLAGS) ./cmd/cli
	$(GO) install $(LDFLAGS) ./cmd/server

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Code quality
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Docker
docker-build:
	@echo "Building Docker image..."
	docker build -t translator:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8443:8443 -v $(PWD)/certs:/app/certs translator:latest

# Certificate generation
generate-certs:
	@echo "Generating self-signed TLS certificates..."
	@mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"
	@echo "Certificates generated in certs/"

# Development
dev-server:
	@echo "Starting development server with auto-reload..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air -c .air.toml

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI)-linux-amd64 ./cmd/cli
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER)-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI)-darwin-amd64 ./cmd/cli
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI)-darwin-arm64 ./cmd/cli
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI)-windows-amd64.exe ./cmd/cli
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER)-windows-amd64.exe ./cmd/server
