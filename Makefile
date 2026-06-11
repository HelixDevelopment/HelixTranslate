# Makefile for Translation System v3.0.0

# Variables
APP_NAME = translator
VERSION = 3.0.0
BUILD_DIR = build
DIST_DIR = dist

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOFMT = $(GOCMD) fmt
GOVET = $(GOCMD) vet

# Build flags
LDFLAGS = -ldflags "-X main.appVersion=$(VERSION)"

# Docker variables
DOCKER_IMAGE = translator-system
DOCKER_TAG = v$(VERSION)

# Default target
.PHONY: all
all: clean deps fmt vet test build

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
.PHONY: fmt
fmt:
	$(GOFMT) ./...

# Vet code
.PHONY: vet
vet:
	$(GOVET) ./...

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -cover ./...

# Run integration tests
.PHONY: test-integration
test-integration:
	$(GOTEST) -tags=integration -v ./test/integration/...

# Run e2e tests
.PHONY: test-e2e
test-e2e:
	$(GOTEST) -tags=e2e -v ./test/e2e/...

# Run security tests
.PHONY: test-security
test-security:
	$(GOTEST) -tags=security -v ./test/security/...

# Run performance benchmarks
.PHONY: test-performance
test-performance:
	$(GOTEST) -tags=performance -v -bench=. ./test/performance/...

# Run stress tests
.PHONY: test-stress
test-stress:
	$(GOTEST) -tags=stress -v ./test/stress/...

# Run all test categories
.PHONY: test-all
test-all: test test-integration test-e2e test-security test-performance test-stress

# Run challenges
.PHONY: challenges
challenges:
	bash pkg/challenge_runner/scripts/run_all_challenges.sh

# Build all binaries
.PHONY: build
build: verify-constitution build-grpc build-api build-cli

# Build gRPC server
.PHONY: build-grpc
build-grpc:
	@echo "Building gRPC server..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/grpc-server ./cmd/grpc-server

# Build API server
.PHONY: build-api
build-api:
	@echo "Building API server..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/api-server ./cmd/api-server

# Build unified CLI
.PHONY: build-cli
build-cli:
	@echo "Building unified CLI..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/unified-translator ./cmd/unified-translator

# Build for all platforms
.PHONY: build-all
build-all: clean deps
	@echo "Building for all platforms..."
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/grpc-server-linux-amd64 ./cmd/grpc-server
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/api-server-linux-amd64 ./cmd/api-server
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/unified-translator-linux-amd64 ./cmd/unified-translator
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/grpc-server-linux-arm64 ./cmd/grpc-server
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/api-server-linux-arm64 ./cmd/api-server
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/unified-translator-linux-arm64 ./cmd/unified-translator
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/grpc-server-darwin-amd64 ./cmd/grpc-server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/api-server-darwin-amd64 ./cmd/api-server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/unified-translator-darwin-amd64 ./cmd/unified-translator
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/grpc-server-darwin-arm64 ./cmd/grpc-server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/api-server-darwin-arm64 ./cmd/api-server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/unified-translator-darwin-arm64 ./cmd/unified-translator
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/grpc-server-windows-amd64.exe ./cmd/grpc-server
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/api-server-windows-amd64.exe ./cmd/api-server
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/unified-translator-windows-amd64.exe ./cmd/unified-translator

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f *.log

# Run gRPC server
.PHONY: run-grpc
run-grpc: build-grpc
	@echo "Starting gRPC server..."
	./$(BUILD_DIR)/grpc-server

# Run API server
.PHONY: run-api
run-api: build-api
	@echo "Starting API server..."
	./$(BUILD_DIR)/api-server

# Run full system
.PHONY: run-system
run-system: build-grpc build-api
	@echo "Starting translation system..."
	@echo "Starting gRPC server in background..."
	./$(BUILD_DIR)/grpc-server &
	sleep 2
	@echo "Starting API server..."
	./$(BUILD_DIR)/api-server

# Run development environment
.PHONY: dev
dev:
	@echo "Starting development environment..."
	@echo "Starting gRPC server (port 50051)..."
	go run ./cmd/grpc-server -log-level debug &
	sleep 2
	@echo "Starting API server (port 8080)..."
	go run ./cmd/api-server -log-level debug

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 50051:50051 -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Show help
.PHONY: help
help:
	@echo "Translation System Makefile v$(VERSION)"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Clean, deps, fmt, vet, test, and build"
	@echo "  build         - Build all binaries for current platform"
	@echo "  build-grpc    - Build gRPC server"
	@echo "  build-api     - Build API server"
	@echo "  build-cli     - Build unified CLI"
	@echo "  build-all     - Build for all platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  run-grpc      - Run gRPC server"
	@echo "  run-api       - Run API server"
	@echo "  run-system    - Run full system (gRPC + API)"
	@echo "  dev           - Run development environment"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make dev                    # Start development environment"
	@echo "  make build                  # Build all binaries"
	@echo "  make run-system             # Run full system"
	@echo "  make docker-build docker-run # Build and run with Docker"

# Development shortcuts
.PHONY: quick-test
quick-test: fmt vet test

.PHONY: verify-constitution
verify-constitution:
	@bash tests/constitution_inheritance_gate.sh

.PHONY: verify-constitution-meta
verify-constitution-meta:
	@bash scripts/testing/meta_test_constitution_inheritance.sh

.PHONY: test-constitution-inheritance
test-constitution-inheritance:
	@bash tests/test_constitution_inheritance.sh

.PHONY: pre-commit
pre-commit: verify-constitution quick-test
# Definition of Done gates — portable drop-in from HelixAgent
.PHONY: no-silent-skips no-silent-skips-warn demo-all demo-all-warn demo-one ci-validate-all

no-silent-skips:
	@bash scripts/no-silent-skips.sh

no-silent-skips-warn:
	@NO_SILENT_SKIPS_WARN_ONLY=1 bash scripts/no-silent-skips.sh

demo-all:
	@bash scripts/demo-all.sh

demo-all-warn:
	@DEMO_ALL_WARN_ONLY=1 DEMO_ALLOW_TODO=1 bash scripts/demo-all.sh

demo-one:
	@DEMO_MODULES="$(MOD)" bash scripts/demo-all.sh

ci-validate-all: verify-constitution verify-constitution-meta no-silent-skips-warn demo-all-warn
	@echo "ci-validate-all: all gates executed"
