# AGENTS.md - Universal Multi-Format Multi-Language Ebook Translation System

This file contains project-specific information intended for AI coding agents. The reader is expected to know nothing about the project beforehand.

---

## Project Overview

This is a Go-based ebook translation system that translates books between multiple languages using various LLM providers. It supports multiple input/output formats (FB2, EPUB, TXT, HTML, PDF, DOCX) and provides both CLI tools and API servers for translation workflows.

Key capabilities:
- Multi-format ebook parsing and generation
- Integration with multiple LLM providers (OpenAI, Anthropic, DeepSeek, Zhipu, Qwen, Gemini, Ollama, LlamaCpp)
- Distributed translation via SSH workers
- Real-time WebSocket monitoring dashboard
- REST API and gRPC interfaces
- Translation caching, quality verification, and multi-pass processing
- Preparation phase analysis for improved translation quality
- Markdown workflow (EPUB to Markdown and back)

---

## Technology Stack

- **Language**: Go 1.25.2
- **Module**: `digital.vasic.translator`
- **Version**: 3.0.0 (Makefile), 2.3.0 (VERSION file)
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **WebSocket**: Gorilla WebSocket (github.com/gorilla/websocket)
- **gRPC**: google.golang.org/grpc with Protocol Buffers
- **Authentication**: JWT (github.com/golang-jwt/jwt/v5)
- **Databases**: PostgreSQL (github.com/lib/pq), SQLite (github.com/mattn/go-sqlite3), Redis (github.com/redis/go-redis/v9)
- **Document Processing**: unidoc/unioffice, unidoc/unipdf/v3
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Testing**: testify (github.com/stretchr/testify)
- **Transport**: QUIC support (github.com/quic-go/quic-go)
- **TLS**: golang.org/x/crypto

---

## Build and Test Commands

### Building

```bash
# Build all primary binaries (gRPC server, API server, unified CLI)
make build

# Build specific components
make build-grpc      # ./build/grpc-server
make build-api       # ./build/api-server
make build-cli       # ./build/unified-translator

# Build for all platforms (Linux AMD64/ARM64, macOS AMD64/ARM64, Windows AMD64)
make build-all

# Clean build artifacts
make clean
```

Additional entry points that can be built individually:
```bash
go build -o build/monitor-server ./cmd/monitor-server
go build -o build/translator ./cmd/translator
go build -o build/server ./cmd/server
go build -o build/markdown-translator ./cmd/markdown-translator
go build -o build/preparation-translator ./cmd/preparation-translator
go build -o build/ebook-translator ./cmd/ebook-translator
go build -o build/translate-ssh ./cmd/translate-ssh
go build -o build/deployment ./cmd/deployment
```

### Testing

```bash
# Run all tests
make test
# or
go test ./... -v

# Run tests with coverage (generates coverage.html)
make test-coverage

# Run a specific package
go test -v ./pkg/translator
go test -v ./pkg/events
go test -v ./pkg/websocket

# Run a specific test function
go test -v -run TestFunctionName ./pkg/package

# Quick development cycle (format + vet + test)
make quick-test

# Pre-commit checks
make pre-commit
```

### Code Quality

```bash
# Format code
make fmt

# Vet code
make vet

# Lint (requires golangci-lint installation)
golangci-lint run
# Configuration is in .golangci.yml
```

### Running Services (Development)

```bash
# Start development environment (gRPC on :50051 + API on :8080 in debug mode)
make dev

# Run full system (builds and runs both servers)
make run-system

# Individual servers
make run-grpc    # gRPC server on port 50051
make run-api     # API server on port 8080

# WebSocket monitoring server
go run ./cmd/monitor-server
# Dashboard available at http://localhost:8090/monitor
```

### Docker

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Full stack with PostgreSQL and Redis
docker-compose up -d
```

---

## Code Organization

### Directory Structure

```
cmd/                    # Command-line entry points
  api-server/          # REST API server (port 8080)
  cli/                 # Basic CLI tool
  deployment/          # Deployment management tool
  ebook-translator/    # Specialized ebook translator
  grpc-server/         # gRPC translation service (port 50051)
  markdown-translator/ # Markdown workflow tool
  monitor-server/      # WebSocket monitoring server (port 8090)
  preparation-translator/ # Pre-translation analysis tool
  server/              # Main REST API server (older entry point)
  ssh-translation/     # SSH translation worker
  translate-ssh/       # SSH worker standalone binary
  translator/          # Legacy translator CLI
  unified-translator/  # Primary unified CLI with full provider support

pkg/                    # Public packages
  api/                 # REST API handlers, middleware, routing
  batch/               # Batch processing logic
  coordination/        # Multi-LLM coordination
  deployment/          # Docker and SSH deployment utilities
  distributed/         # Distributed processing over SSH workers
  ebook/               # Universal ebook parsing (FB2, EPUB, TXT, HTML, PDF, DOCX)
  events/              # Event bus system for real-time updates
  fb2/                 # FB2 format-specific handling
  format/              # Format detection and validation
  grpc/                # gRPC service implementation and proto definitions
  hardware/            # Hardware detection for optimization
  language/            # Language detection utilities
  logger/              # Structured logging utilities
  markdown/            # EPUB to Markdown conversion workflow
  models/              # Data models and registry
  preparation/         # Pre-translation preparation phase
  progress/            # Progress tracking
  report/              # Report generation
  script/              # Script conversion (Cyrillic/Latin)
  security/            # JWT authentication, rate limiting
  sshworker/           # SSH worker management
  storage/             # Database abstraction (PostgreSQL, Redis, SQLite)
  translator/          # Translation engine core
    llm/               # LLM provider implementations
  verification/        # Translation quality verification
  version/             # Version information
  websocket/           # WebSocket hub and connection management

internal/               # Internal packages
  cache/               # Caching layer
  config/              # Configuration loading and validation
  scripts/             # Production and deployment shell scripts
  working/             # Working configuration files and runtime data

test/                   # Test suites
  distributed/         # Distributed system tests
  e2e/                 # End-to-end tests
  fixtures/            # Test data (ebooks, configs, translations)
  integration/         # Cross-package integration tests
  mocks/               # testify/mock implementations
  performance/         # Performance benchmarks
  security/            # Security-focused tests
  stress/              # Load testing
  translator/          # Translator-specific tests
  unit/                # Unit tests for individual components
  utils/               # Test helper utilities

api/                    # API documentation
  examples/            # API usage examples
  openapi/             # OpenAPI 3.0 specification

web/                    # Web interface assets
  templates/           # HTML templates (dashboard.html)

scripts/                # Utility and demonstration scripts
  ebook_translation_workflow.sh
  python_translation.sh
  run_monitoring_demo.sh

build/                  # Build artifacts directory
docs/                   # Project documentation
Documentation/          # Additional documentation and guides
```

---

## Code Style Guidelines

### Go Conventions

- **Naming**: PascalCase for exported identifiers, camelCase for unexported.
- **Imports order**: Standard library → third-party → local packages (`digital.vasic.translator/...`). Keep alphabetical within each group.
- **Types**: Prefer `any` over `interface{}` where appropriate. Use interfaces to define behavior contracts.
- **Error handling**: Always handle errors explicitly. Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`.
- **Comments**: Document all exported functions, types, and packages. Avoid commenting the obvious.
- **Line length**: Keep lines under 140 characters (per `.golangci.yml`).
- **Function length**: Aim for under 100 lines or 50 statements.
- **Cyclomatic complexity**: Keep under 15 (gocyclo setting).

### Security Rules

- **Never hardcode API keys or secrets**. Use environment variables or secure config files.
- **Never commit files containing secrets**. `.gitignore` already excludes `config_with_keys.json`, `.env`, `secrets.*`, etc.
- Input validation is required on all API endpoints.
- Use parameterized queries or ORM patterns for database access.

### Linting Configuration

The `.golangci.yml` file enables a strict set of linters including:
- `govet`, `staticcheck`, `gosimple`, `ineffassign` for correctness
- `gocyclo`, `funlen`, `gocognit`, `nestif` for complexity
- `gosec` for security issues
- `goimports` with local prefix `digital.vasic.translator`
- `misspell` with US English locale

Test files and `cmd/` entry points have relaxed rules for `funlen`, `gochecknoinits`, etc.

---

## Testing Strategy

### Test Organization

Tests are organized by scope rather than co-located with source files:

- `test/unit/` - Unit tests for individual packages and components
- `test/integration/` - Cross-package integration tests
- `test/e2e/` - End-to-end tests with realistic scenarios
- `test/security/` - Authentication, authorization, and security tests
- `test/performance/` - Benchmarks and performance tests
- `test/stress/` - Load and stress tests
- `test/distributed/` - Distributed system and SSH worker tests
- `test/mocks/` - Shared mock implementations using testify/mock
- `test/fixtures/` - Sample ebooks, configs, and translation data
- `test/utils/` - Shared test helpers and infrastructure

Some packages also have `*_test.go` files alongside source code, particularly in `cmd/` and `pkg/api/`.

### Testing Patterns

- **Table-driven tests**: Preferred style. Define a slice of anonymous structs with inputs and expected outputs, then iterate with `t.Run()`.
- **Sub-tests**: Use `t.Run("descriptive_name", func(t *testing.T) { ... })` for clarity.
- **Mocking**: Use `testify/mock` for interface mocking. Mocks are defined in `test/mocks/providers.go`.
- **Build tags**: Use `//go:build integration` and `//go:build e2e` for tests that should not run in the default quick-test cycle.
- **HTTP testing**: Use `httptest` package and `gin.SetMode(gin.TestMode)` for API handler tests.
- **Coverage**: Current overall coverage is approximately 43.6%. HTML reports are generated at `coverage.html`.

### Test Utilities

- `test/utils/helpers.go` provides helpers for creating temporary test files, test configs, and HTTP test servers.
- `test/mocks/mock_translator.go` provides `MockTranslator` and `MockLLMProvider`.

---

## Architecture and Key Patterns

### Event-Driven Architecture

The system uses a central event bus (`pkg/events/event_bus.go`) for component communication and real-time progress tracking:

- **Event types**: `EventTranslationStarted`, `EventTranslationProgress`, `EventTranslationCompleted`, `EventTranslationError`, `EventConversionStarted`, `EventConversionProgress`, `EventConversionCompleted`, `EventConversionError`.
- **Subscription**: `Subscribe(eventType, handler)` for type-specific; `SubscribeAll(handler)` for global listeners.
- **WebSocket Integration**: `pkg/websocket/hub.go` subscribes to all events and broadcasts to connected dashboard clients.
- **Session tracking**: Every translation operation has a unique `SessionID`. Events are tagged with session IDs so the dashboard can filter per-client.

### Translation Pipeline

1. **Format detection** (`pkg/format/detector.go`) identifies the input file type.
2. **Parsing** (`pkg/ebook/parser.go` and format-specific parsers) extracts content.
3. **Preparation** (optional, `pkg/preparation/`) analyzes content for characters, terminology, culture.
4. **Translation** (`pkg/translator/translator.go` and `pkg/translator/llm/`) sends segments to LLM providers.
5. **Verification** (`pkg/verification/`) runs quality checks and multi-pass refinement.
6. **Output generation** writes translated content back to the target format.

### LLM Provider Pattern

All LLM providers implement the `LLMClient` interface in `pkg/translator/llm/`:

```go
type LLMClient interface {
    Translate(ctx context.Context, text string, prompt string) (string, error)
    GetProviderName() string
}
```

**Supported providers**:
- API-based: OpenAI, Anthropic, Zhipu, DeepSeek, Qwen, Gemini
- Self-hosted: Ollama, LlamaCpp (executes local binary)
- Distributed: SSH workers for remote processing

Provider selection uses a factory pattern (`NewLLMTranslator()`) that validates provider/model and creates the appropriate client.

### Configuration System

- `internal/config/config.go` defines the canonical config struct.
- Config is loaded from JSON files (e.g., `config.json`, files in `internal/working/`).
- Environment variables override file values, especially for secrets and API keys.
- Provider-specific configs may live in separate JSON files (e.g., `config_openai.json`).

### Distributed Processing

- `pkg/distributed/coordinator.go` manages remote LLM instances across SSH workers.
- `pkg/sshworker/worker.go` handles SSH connection management and remote command execution.
- `internal/scripts/` contains deployment and health-check scripts for workers.
- Workers must be reachable from the coordinator. Version synchronization is required.

---

## Security Considerations

- **TLS required**: All API communication uses HTTPS/HTTP3. TLS certificates are expected in the `certs/` directory (`server.crt`, `server.key`).
- **JWT authentication**: Configurable via `security.enable_auth` in config. Secret must be at least 16 characters.
- **Rate limiting**: Configurable requests-per-second and burst limits via `security.rate_limit_rps` and `security.rate_limit_burst`.
- **API key management**: API keys must be provided via environment variables (e.g., `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`). Never commit them.
- **CORS**: Configurable allowed origins via `security.cors_origins`.
- **Input validation**: All API endpoints validate request bodies using Gin binding.
- **SSH security**: Key-based authentication is preferred for SSH workers. Password auth is supported but discouraged for production.

### Sensitive Files (Already in .gitignore)

- `config_with_keys.json`, `api_keys.json`
- `.env`, `.env.*`, `secrets.*`, `keys.*`
- `qwen_credentials.json`, `oauth_creds.json`
- `.translator/` directory

---

## Deployment Processes

### Docker Deployment

A multi-stage `Dockerfile` is provided:
- **Builder stage**: `golang:1.24-alpine` with `git` and `make`
- **Production stage**: `alpine:latest` with `ca-certificates` and `tzdata`
- Runs as non-root user (`appuser`, UID 1000)
- Exposes port 8443 (TCP and UDP for HTTP/3)
- Health check endpoint: `https://localhost:8443/health`
- Default command: `/app/translator-server -config /app/config/config.json`

### Docker Compose

`docker-compose.yml` provides a full production stack:
- PostgreSQL 16 (with health checks)
- Redis 7 (with password and memory limits)
- Translator API server (depends on PostgreSQL and Redis)

### Manual Deployment

```bash
# Build binaries
make build

# Start gRPC server
./build/grpc-server

# Start API server
./build/api-server

# Start monitoring server
./build/monitor-server
```

### Deployment Scripts

- `internal/scripts/deploy-system.sh` - System deployment
- `internal/scripts/deploy_worker.sh` - Worker deployment
- `internal/scripts/check_health.sh` - Health checks
- `internal/scripts/check_worker.sh` - Worker status checks

---

## Development Conventions

### Adding a New LLM Provider

1. Create a new package under `pkg/translator/llm/<provider>/`.
2. Implement the `LLMClient` interface.
3. Add provider-specific configuration to `internal/config/config.go`.
4. Register the provider in the factory/initialization code.
5. Add unit tests in `test/unit/` or alongside the package.
6. Update `api/openapi/openapi.yaml` if the provider appears in API enums.

### Adding a New File Format

1. Implement the parser interface in `pkg/ebook/` or a format-specific package.
2. Register the format in `pkg/format/detector.go`.
3. Add writer support if output generation is required.
4. Add sample files to `test/fixtures/ebooks/`.
5. Add parser tests.

### Session and Progress Tracking

- Always generate a unique `SessionID` (e.g., UUID) for each translation operation.
- Emit progress events via the event bus so WebSocket clients receive updates.
- The monitoring dashboard (`web/templates/dashboard.html`) displays sessions, progress bars, event logs, and worker status.

---

## Essential Files Quick Reference

| File | Purpose |
|------|---------|
| `go.mod` / `go.sum` | Module definition and dependency checksums |
| `Makefile` | Build, test, and development commands |
| `config.json` | Default application configuration |
| `Dockerfile` | Multi-stage container build |
| `docker-compose.yml` | Full stack with PostgreSQL and Redis |
| `.golangci.yml` | Strict linting configuration |
| `internal/config/config.go` | Canonical configuration structs and loading |
| `pkg/events/event_bus.go` | Central pub/sub event system |
| `pkg/websocket/hub.go` | WebSocket hub for real-time monitoring |
| `pkg/translator/llm/llm.go` | LLM provider interface and factory |
| `api/openapi/openapi.yaml` | OpenAPI 3.0 specification |
| `VERSION` | Current application version (2.3.0) |

---

## Common Pitfalls

- **FB2 XML**: Always register the FB2 namespace before parsing. Use UTF-8 encoding. Preserve XML hierarchy during translation.
- **LLM Rate Limits**: Each provider has different limits. Implement backoff and respect `max_concurrent` settings.
- **SSH Workers**: Ensure key-based auth is configured. Workers must run the same binary version as the coordinator.
- **Large Files**: Use streaming for large ebook translations to avoid excessive memory usage.
- **TLS Certificates**: The API server requires valid certificates in `certs/` for HTTPS/HTTP3. Generate or provide them before starting the server.
- **Import Cycles**: The `pkg/translator/llm/` package uses a type alias to avoid cycles with `pkg/translator/`. Be careful when adding new cross-package dependencies.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->

