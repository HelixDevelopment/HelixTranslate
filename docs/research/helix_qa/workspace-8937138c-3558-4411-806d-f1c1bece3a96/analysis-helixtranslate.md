# HelixTranslate Repository — Comprehensive Analysis

> **Repository:** https://github.com/HelixDevelopment/HelixTranslate
> **Analysis Date:** 2025-10-05
> **Commit Branch:** `main`

---

## 1. Project Overview

HelixTranslate is a **high-performance, enterprise-grade universal ebook translation toolkit** written in Go. It translates books between any language pair, supporting multiple ebook formats (FB2, EPUB, TXT, HTML, PDF, DOCX) and integrating with numerous LLM providers for AI-powered translation.

### Core Identity
- **Module:** `digital.vasic.translator`
- **Go Version:** 1.25.2
- **Authentic Version:** `2.3.0` (file `VERSION`) — Makefile references `3.0.0`; treat `VERSION` as authoritative per CLAUDE.md
- **License:** MIT
- **Organization:** HelixDevelopment (github.com/HelixDevelopment)

### Key Capabilities
1. Multi-format ebook parsing and generation
2. Integration with 8+ LLM providers (cloud and self-hosted)
3. Distributed translation via SSH workers
4. Real-time WebSocket monitoring dashboard
5. REST API with HTTP/3 (QUIC) support
6. gRPC service interface
7. Translation caching, quality verification, multi-pass processing
8. Preparation phase analysis for improved translation quality
9. Markdown workflow (EPUB ↔ Markdown)
10. Serbian Cyrillic↔Latin script conversion

---

## 2. Technology Stack

### Language & Runtime
- **Go 1.25.2** (primary language, entire codebase)

### Web Frameworks & Protocols
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.11.0 | HTTP REST API framework |
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket for real-time monitoring |
| `github.com/quic-go/quic-go` | v0.56.0 | HTTP/3 (QUIC) support |
| `google.golang.org/grpc` | v1.77.0 | gRPC service framework |
| `google.golang.org/protobuf` | v1.36.10 | Protocol Buffers |

### Authentication & Security
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | JWT authentication |
| `golang.org/x/crypto` | v0.48.0 | TLS, crypto primitives |

### Databases & Caching
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/lib/pq` | v1.10.9 | PostgreSQL driver |
| `github.com/mattn/go-sqlite3` | v1.14.24 | SQLite driver |
| `github.com/redis/go-redis/v9` | v9.7.0 | Redis caching |

### Document Processing
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/unidoc/unioffice` | v1.39.0 | DOCX processing |
| `github.com/unidoc/unipdf/v3` | v3.69.0 | PDF processing |

### CLI & Utilities
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10.1 | CLI framework |
| `github.com/stretchr/testify` | v1.11.1 | Testing assertions/mocks |
| `github.com/google/uuid` | v1.6.0 | UUID generation |
| `golang.org/x/text` | v0.34.0 | Text processing |

### Frontend (Dashboard)
- Chart.js — progress visualization
- Tailwind CSS — dashboard styling
- Vanilla HTML/JS — dashboard templates

---

## 3. Directory Structure

```
HelixTranslate/
├── .gitmodules                  # Git submodule definitions
├── .gitignore                   # Exclusions for secrets, artifacts
├── .golangci.yml                # Strict linting configuration
├── AGENTS.md                    # AI agent guidance (511 lines)
├── CLAUDE.md                    # Claude Code guidance (182 lines)
├── CONSTITUTION.md              # Authoritative project rules (98 lines)
├── Dockerfile                   # Multi-stage Docker build
├── Dockerfile                   # Multi-stage Docker build
├── Makefile                     # Build/test/deploy commands (222 lines)
├── VERSION                      # "2.3.0" (authoritative version)
├── README.md                    # Project overview & quick start
├── go.mod / go.sum              # Module definition & deps
├── docker-compose.yml           # Full stack (PostgreSQL, Redis, API)
│
├── cmd/                         # Command-line entry points
│   ├── unified-translator/      # PRIMARY CLI with full provider support
│   ├── grpc-server/             # gRPC translation service (:50051)
│   ├── api-server/              # REST/WebSocket API (:8080, HTTP/3 capable)
│   ├── server/                  # Main REST API server (older entry point)
│   ├── monitor-server/          # WebSocket monitoring hub (:8090)
│   ├── translate-ssh/           # Standalone SSH worker binary
│   ├── ssh-translation/         # SSH translation worker
│   ├── preparation-translator/  # Pre-translation analysis tool
│   ├── markdown-translator/     # EPUB↔Markdown workflow
│   ├── ebook-translator/        # Specialized ebook translator
│   ├── cli/                     # Basic CLI tool
│   ├── translator/              # Legacy translator CLI
│   └── deployment/              # Deployment management tool
│
├── pkg/                         # Public packages
│   ├── api/                     # REST API handlers, middleware, routing
│   ├── batch/                   # Batch processing logic
│   ├── coordination/            # Multi-LLM coordination
│   ├── deployment/              # Docker and SSH deployment utilities
│   ├── distributed/             # Distributed processing over SSH workers
│   │   ├── coordinator.go       # Work distribution & aggregation
│   │   ├── ssh_pool.go          # Pooled SSH connections
│   │   ├── pairing.go           # Worker discovery & handshake
│   │   ├── fallback.go          # Graceful degradation
│   │   ├── version_manager.go   # Version sync across workers
│   │   ├── performance.go       # Instrumentation
│   │   └── security.go          # Hardening
│   ├── ebook/                   # Universal ebook parsing (FB2, EPUB, TXT, HTML, PDF, DOCX)
│   │   ├── fb2_parser.go        # FB2 integration parser
│   │   ├── epub_parser.go       # EPUB parser
│   │   ├── epub_writer.go       # EPUB writer
│   │   └── *_parser.go / *_writer.go  # Format-specific handlers
│   ├── events/                  # Event bus system (pub/sub)
│   │   └── events.go            # Thread-safe EventBus
│   ├── fb2/                     # FB2 format-specific XML handling
│   │   └── parser.go            # FB2-specific XML logic
│   ├── format/                  # Format detection and validation
│   │   └── detector.go          # Input file type identification
│   ├── grpc/                    # gRPC service implementation
│   │   └── translator.proto     # Protocol Buffers definition
│   ├── hardware/                # Hardware detection for optimization
│   ├── hash/                    # Hashing utilities
│   ├── language/                # Language detection utilities
│   ├── logger/                  # Structured logging
│   ├── markdown/                # EPUB↔Markdown conversion
│   ├── models/                  # Data models and registry
│   ├── preparation/             # Pre-translation content analysis
│   ├── progress/                # Progress tracking
│   ├── report/                  # Report generation
│   ├── script/                  # Serbian Cyrillic↔Latin conversion
│   ├── security/                # JWT auth, rate limiting, CORS
│   ├── sshworker/               # SSH worker management
│   ├── storage/                 # Database abstraction (PostgreSQL, Redis, SQLite)
│   ├── translator/              # Translation engine core
│   │   ├── translator.go        # Main translator implementation
│   │   └── llm/                 # LLM provider implementations
│   │       ├── llm.go           # LLMClient interface + factory
│   │       ├── openai.go        # OpenAI provider
│   │       ├── anthropic.go     # Anthropic provider
│   │       ├── zhipu.go         # Zhipu/GLM provider
│   │       ├── deepseek.go      # DeepSeek provider
│   │       ├── qwen.go          # Qwen provider
│   │       ├── gemini.go        # Gemini provider
│   │       ├── ollama.go        # Ollama provider
│   │       ├── llamacpp.go      # LlamaCpp provider
│   │       └── mock.go          # Mock provider for tests
│   ├── verification/            # Translation quality verification
│   ├── version/                 # Version information
│   └── websocket/               # WebSocket hub & connection management
│       └── hub.go               # Subscribes to all events, fans out to clients
│
├── internal/                    # Internal packages
│   ├── cache/                   # Caching layer
│   ├── config/                  # Configuration loading & validation
│   │   └── config.go            # Canonical Config structs (308 lines)
│   ├── scripts/                 # Production & deployment scripts
│   └── working/                 # Working configs & runtime data
│
├── test/                        # Cross-cutting test suites
│   ├── unit/                    # Unit tests
│   ├── integration/             # Cross-package integration tests
│   ├── e2e/                     # End-to-end tests
│   ├── performance/             # Performance benchmarks
│   ├── stress/                  # Load & stress tests
│   ├── security/                # Security-focused tests
│   ├── distributed/             # Distributed system & SSH worker tests
│   ├── mocks/                   # Shared testify/mock implementations
│   ├── fixtures/                # Sample ebooks, configs, translation data
│   ├── utils/                   # Test helpers
│   │   └── helpers.go
│   └── translator/              # Translator-specific tests
│
├── tests/                       # Top-level integration tests
│   └── websocket_monitoring_test.go
│
├── api/                         # API documentation
│   ├── examples/                # API usage examples
│   └── openapi/                 # OpenAPI 3.0 specification
│       └── openapi.yaml
│
├── web/                         # Web interface assets
│   └── templates/
│       └── dashboard.html
│
├── docs/                        # Project documentation
│   ├── API.md
│   ├── API_Documentation.md
│   ├── HOST_POWER_MANAGEMENT.md
│   ├── WebSocket_Monitoring_Guide.md
│   ├── User_Guide.md
│   ├── Troubleshooting_Guide.md
│   ├── USER_MANUAL.md
│   ├── COMPREHENSIVE_TESTING_FRAMEWORK_GUIDE.md
│   ├── TESTING_FRAMEWORK_SPECIFICATION.md
│   ├── PHASE{0-4}_EXECUTION_PLAN.md
│   ├── PROJECT_COMPLETION_ROADMAP.md
│   └── [30+ other planning/report docs]
│
├── Documentation/               # Additional documentation & guides
│   ├── AGENTS.md                # Detailed agent guidance
│   ├── ARCHITECTURE.md          # Architecture documentation
│   └── CLAUDE.md                # Legacy Python implementation docs (NOT authoritative)
│
├── scripts/                     # Utility & demonstration scripts
│   ├── run_monitoring_demo.sh
│   ├── ebook_translation_workflow.sh
│   └── python_translation.sh
│
├── challenges/                  # Challenge scripts & CI gates
│   └── scripts/
│       ├── no_suspend_calls_challenge.sh
│       └── host_no_auto_suspend_challenge.sh
│
├── build/                       # Build artifacts directory
├── certs/                       # TLS certificates (server.crt, server.key)
├── research/                    # Research materials
├── tools/                       # Development tools
├── e2e_test_dir/                # E2E test directory
├── test_batch/                  # Batch test data
│
├── Containers/                  # [SUBMODULE] vasic-digital/Containers
│   └── url: git@github.com:vasic-digital/Containers.git
│
├── Challenges/                  # [SUBMODULE] vasic-digital/Challenges
│   └── url: git@github.com:vasic-digital/Challenges.git
│
├── monitor.html                 # Basic monitoring dashboard
├── enhanced-monitor.html        # Advanced dashboard with SSH worker support
├── config.json                  # Main application configuration
│
└── demo-*.go                    # Demo scripts (from prior sessions, not source of truth)
    ├── demo-translation-with-monitoring-fixed.go
    ├── demo-comprehensive-monitoring.go
    ├── demo-ssh-worker-with-monitoring.go
    ├── demo-real-llm-with-monitoring.go
    └── demo-websocket-client.go
```

---

## 4. Submodules

Two git submodules are defined in `.gitmodules`:

| Submodule | Path | Repository |
|-----------|------|------------|
| **Containers** | `./Containers` | `git@github.com:vasic-digital/Containers.git` |
| **Challenges** | `./Challenges` | `git@github.com:vasic-digital/Challenges.git` |

Both are referenced in `go.mod` via `replace` directives:
```go
replace digital.vasic.challenges => ./Challenges
replace digital.vasic.containers => ./Containers
```

---

## 5. APIs and Services

### 5.1 REST API (Port 8080/8443)

The REST API is served via Gin framework with HTTP/3 (QUIC) support. Endpoints include:
- **File upload** for ebook translation
- **Translation status** queries
- **Health check** (`/health`)
- **Real-time WebSocket** events at `/ws`
- **Monitoring dashboard** at `/monitor`
- JWT authentication, rate limiting, CORS configuration

OpenAPI 3.0 specification at `api/openapi/openapi.yaml`.

### 5.2 gRPC Service (Port 50051)

Full Protocol Buffers definition at `pkg/grpc/translator.proto`:

```protobuf
service TranslationService {
  rpc StartTranslation(TranslationRequest) returns (TranslationResponse);
  rpc GetTranslationStatus(TranslationStatusRequest) returns (TranslationStatusResponse);
  rpc ListTranslations(google.protobuf.Empty) returns (TranslationListResponse);
  rpc CancelTranslation(CancelTranslationRequest) returns (CancelTranslationResponse);
  rpc StreamTranslationProgress(TranslationStreamRequest) returns (stream TranslationProgressEvent);
  rpc GetProviders(google.protobuf.Empty) returns (ProvidersResponse);
  rpc SubscribeEvents(EventSubscriptionRequest) returns (stream SystemEvent);
}
```

**Key message types:**
- `TranslationRequest` — session_id, input/output files, source/target language, provider config, options
- `ProviderConfig` — type, model, temperature, max_tokens, API/SSH/LlamaCpp settings
- `TranslationStatusResponse` — progress, steps, files, error tracking
- `TranslationProgressEvent` — streaming progress with step info, errors, metadata
- `ProvidersResponse` — lists all available providers with models and capabilities

### 5.3 WebSocket Monitoring (Port 8090)

- Central hub subscribes to all EventBus events
- Fans out to connected dashboard clients
- Filters by session ID per client
- Endpoints: `/ws` (WebSocket), `/monitor` (dashboard)

### 5.4 Monitoring Dashboard

- **Basic:** `monitor.html` — real-time progress bars and charts
- **Enhanced:** `enhanced-monitor.html` — SSH worker support, session history
- **Template:** `web/templates/dashboard.html`

---

## 6. Client Applications

| Client | Location | Description |
|--------|----------|-------------|
| **Unified Translator (PRIMARY)** | `cmd/unified-translator/` | Full-featured CLI with all provider support |
| **gRPC Server** | `cmd/grpc-server/` | gRPC translation service |
| **API Server** | `cmd/api-server/` | REST/WebSocket API server |
| **Monitor Server** | `cmd/monitor-server/` | WebSocket monitoring hub |
| **SSH Worker** | `cmd/translate-ssh/` | Standalone SSH worker for distributed mode |
| **Preparation Translator** | `cmd/preparation-translator/` | Pre-translation analysis |
| **Markdown Translator** | `cmd/markdown-translator/` | EPUB↔Markdown workflow |
| **Ebook Translator** | `cmd/ebook-translator/` | Specialized ebook translator |
| **CLI** | `cmd/cli/` | Basic CLI tool |
| **Legacy Translator** | `cmd/translator/` | Legacy translator CLI |
| **Deployment Tool** | `cmd/deployment/` | Deployment management |
| **Server** | `cmd/server/` | Older REST API entry point |

### Unified Translator CLI Flags
```
-input/-i, -output/-o, -source-lang, -target-lang, -script {cyrillic,latin}
-provider {openai,anthropic,zhipu,deepseek,qwen,gemini,ollama,llamacpp,ssh}
-model, -api-key, -base-url, -temperature, -max-tokens, -timeout
-ssh-host/-ssh-user/-ssh-password/-ssh-port
-llama-binary/-llama-model/-context-size
-workers, -chunk-size, -concurrency, -verify
-monitoring, -monitoring-port
```

### Web Dashboard (Interactive)
- Served at `http://localhost:8090/monitor`
- Real-time progress visualization
- Event logging with filtering
- SSH worker status monitoring
- Responsive design (desktop + mobile)

---

## 7. Supported Translation Providers & Models

All providers implement the `LLMClient` interface:
```go
type LLMClient interface {
    Translate(ctx context.Context, text string, prompt string) (string, error)
    GetProviderName() string
}
```

### Cloud API Providers

| Provider | File | Models |
|----------|------|--------|
| **OpenAI** | `pkg/translator/llm/openai.go` | gpt-3.5-turbo, gpt-4, gpt-4-turbo, gpt-4o |
| **Anthropic** | `pkg/translator/llm/anthropic.go` | claude-3-opus-20240229, claude-3-sonnet-20240229, claude-3-haiku-20240307 |
| **Zhipu (GLM)** | `pkg/translator/llm/zhipu.go` | glm-4, glm-3-turbo |
| **DeepSeek** | `pkg/translator/llm/deepseek.go` | deepseek-chat, deepseek-coder |
| **Qwen (Alibaba)** | `pkg/translator/llm/qwen.go` | qwen-max, qwen-plus, qwen-turbo |
| **Gemini (Google)** | `pkg/translator/llm/gemini.go` | gemini-pro, gemini-pro-vision |

### Self-Hosted Providers

| Provider | File | Notes |
|----------|------|-------|
| **Ollama** | `pkg/translator/llm/ollama.go` | Local LLM (llama2, codellama, mistral, vicuna + custom) |
| **LlamaCpp** | `pkg/translator/llm/llamacpp.go` | Local binary execution (llama2, mistral, vicuna + custom) |

### Special Modes

| Provider | Purpose |
|----------|---------|
| **SSH** | Distributed remote translation via SSH workers |
| **Mock** | Testing provider (`pkg/translator/llm/mock.go`) |

---

## 8. Testing Infrastructure

### 8.1 Test Organization

Tests are organized in **two locations** (per CLAUDE.md):

1. **Alongside source** — `*_test.go` files in each `pkg/*` directory (majority of tests)
2. **Centralized test directory** — `test/` organized by scope:
   - `test/unit/` — Individual component tests
   - `test/integration/` — Cross-package integration tests
   - `test/e2e/` — End-to-end tests with realistic scenarios
   - `test/performance/` — Benchmarks and performance tests
   - `test/stress/` — Load and stress tests
   - `test/security/` — Auth, authorization, security tests
   - `test/distributed/` — Distributed system & SSH worker tests
   - `test/mocks/` — Shared testify/mock implementations (`mock_translator.go`, `mocks/providers.go`)
   - `test/fixtures/` — Sample ebooks, configs, translation data
   - `test/utils/` — Test helper utilities (`helpers.go`)
   - `test/translator/` — Translator-specific tests

3. **Top-level integration tests** — `tests/websocket_monitoring_test.go`

### 8.2 Test Framework & Tools
- **testify** (`github.com/stretchr/testify`) — Assertions and mocks
- **Go testing** — Standard library `testing` package
- **Build tags** — `//go:build integration`, `//go:build e2e` for selective execution
- **HTTP testing** — `httptest` package + `gin.SetMode(gin.TestMode)`
- **Coverage** — ~43.6% overall; HTML reports at `coverage.html`

### 8.3 Test Commands
```bash
make test                    # go test -v ./...
make test-coverage           # go test -v -cover ./...
make quick-test              # fmt + vet + test
go test -v -run TestFunctionName ./pkg/package  # Single test
go test -bench=. ./tests/websocket_monitoring_test.go  # Benchmarks
```

### 8.4 Test Patterns
- **Table-driven tests** — Preferred style with anonymous struct slices
- **Sub-tests** — `t.Run("descriptive_name", ...)` 
- **Mocking** — `testify/mock` via `test/mocks/`
- **Naming convention** — `TestFunctionName_Scenario`

### 8.5 Definition of Done (Testing Requirements)

Per CLAUDE.md, "Done" requires:
1. **No self-certification** — Forbidden words: "verified, tested, working, complete, fixed, passing" without real terminal output
2. **Demo before code** — Every task starts with runnable acceptance demo
3. **Real system** — Demos run against real artifacts, not mocks
4. **Loud skips** — `t.Skip` requires `SKIP-OK: #<ticket>` annotation
5. **Contract tests** on every module boundary
6. **Evidence in PR** — Fenced `## Demo` block with exact commands + output

---

## 9. Documentation Structure

### Root-Level Documentation
| File | Purpose |
|------|---------|
| `README.md` | Project overview, WebSocket monitoring system quick start (441 lines) |
| `CLAUDE.md` | Claude Code guidance — project, commands, architecture, conventions (182 lines) |
| `AGENTS.md` | Universal AI agent guidance — full codebase orientation (511 lines) |
| `CONSTITUTION.md` | Authoritative project rules, CONST-033 (98 lines) |

### docs/ Directory (30+ files)
| File | Purpose |
|------|---------|
| `API.md` / `API_Documentation.md` | REST & WebSocket API reference |
| `HOST_POWER_MANAGEMENT.md` | CONST-033 background and runbook |
| `WebSocket_Monitoring_Guide.md` | Complete technical documentation |
| `User_Guide.md` | Step-by-step user instructions |
| `Troubleshooting_Guide.md` | Common issues and solutions |
| `USER_MANUAL.md` | Comprehensive user manual |
| `COMPREHENSIVE_TESTING_FRAMEWORK_GUIDE.md` | Testing guide |
| `TESTING_FRAMEWORK_SPECIFICATION.md` | Test framework spec |
| `PHASE{0-4}_EXECUTION_PLAN.md` | Development phase plans |
| `PROJECT_COMPLETION_ROADMAP.md` | Project completion roadmap |
| `DEVELOPER.md` | Developer reference |
| `memory.md` | Project memory/notes |

### Documentation/ Directory
| File | Purpose |
|------|---------|
| `AGENTS.md` | Detailed agent guidance |
| `ARCHITECTURE.md` | Architecture documentation |
| `CLAUDE.md` | Legacy Python implementation docs (**NOT authoritative** for Go codebase) |

### Other Documentation
| Location | Content |
|----------|---------|
| `api/openapi/openapi.yaml` | OpenAPI 3.0 specification |
| `web/templates/dashboard.html` | Dashboard template |
| `challenges/` | CI challenge scripts |
| `scripts/` | Demo and workflow scripts |

---

## 10. CLAUDE.md Content (Full Summary)

**File:** `CLAUDE.md` (182 lines)

### Key Sections:
1. **Project** — Go 1.25.2, module `digital.vasic.translator`, v2.3.0, 8+ LLM providers, SSH distributed workers, WebSocket monitoring
2. **Commands** — Build/test/lint via Makefile: `make deps/build/test/fmt/vet/quick-test/dev/run-system/docker-build`
3. **Lint** — `golangci-lint run` with `.golangci.yml`, local prefix `digital.vasic.translator`, lll@140
4. **Entry points** — 11 cmd binaries documented, `unified-translator` is PRIMARY
5. **Architecture** — Pipeline: format detection → parsing → translation → output
6. **LLM provider layer** — `LLMClient` interface, factory pattern, 8 provider files + mock
7. **Event-driven core** — `pkg/events/events.go` pub/sub bus, WebSocket hub subscribes automatically
8. **Distributed/SSH workers** — `pkg/distributed/` coordinator, pool, pairing, fallback, version_manager
9. **Storage & caching** — PostgreSQL, Redis, SQLite abstraction in `pkg/storage/`
10. **Other packages** — preparation, verification, script (Serbian), markdown, security, batch, coordination, hardware
11. **Configuration** — `config.json` root, env vars for API keys, `.gitignore` for secrets
12. **Testing layout** — Both `pkg/*_test.go` and `test/` directory (different directories!)
13. **Conventions** — Module-local imports, `fmt.Errorf("...: %w", err)`, `any` over `interface{}`, lll@140
14. **Definition of Done** — 6 rules requiring real terminal output, no self-certification
15. **CONST-033** — Hard ban on host power management operations

---

## 11. AGENTS.md Content (Full Summary)

**File:** `AGENTS.md` (511 lines)

### Key Sections:
1. **Project Overview** — Go-based ebook translation, multi-format, multi-provider, distributed
2. **Technology Stack** — Full dependency table with versions
3. **Build & Test Commands** — make targets, individual builds, Docker
4. **Code Organization** — Complete directory tree with descriptions
5. **Code Style Guidelines** — Go conventions, naming, error handling, line length
6. **Security Rules** — Never hardcode secrets, input validation, parameterized queries
7. **Linting Configuration** — `.golangci.yml` enabled linters, relaxed rules for test/cmd/pkg
8. **Testing Strategy** — Scope-based organization, patterns, ~43.6% coverage
9. **Architecture** — Event-driven, translation pipeline, LLM provider pattern, config system, distributed
10. **Security** — TLS required, JWT auth, rate limiting, CORS, SSH key auth
11. **Deployment** — Docker multi-stage, docker-compose (PostgreSQL 16, Redis 7), manual
12. **Development Conventions** — Adding providers, file formats, session/progress tracking
13. **Essential Files Quick Reference** — Table of 12 critical files
14. **Common Pitfalls** — FB2 XML, LLM rate limits, SSH workers, large files, TLS certs, import cycles
15. **CONST-033** — Host power management ban with defense-in-depth

---

## 12. CONSTITUTION.md Content (Full Summary)

**File:** `CONSTITUTION.md` (98 lines)

### Status: Active — Authoritative rule set (overrides CLAUDE.md, AGENTS.md)

### Mandatory Standards:
1. **Reproducibility** — Every change reproducible from clean clone
2. **Tests track behavior, not code**
3. **No silent skips, no silent mocks above unit tests**
4. **Conventional Commits** for all commits
5. **SSH-only for git operations** — HTTPS prohibited

### Numbered Rules:
- **CONST-033** — Host Power Management is Forbidden (mandatory, non-negotiable)
  - Complete ban on suspend/hibernate/poweroff/reboot operations
  - Defense-in-depth: 5 mandatory artifacts (installer, user bootstrap, scanner, 2 challenges)
  - Both challenges must run in CI; violations block merge
  - Background: 2026-04-26 incident — auto-suspend killed HelixAgent and 41 services

### Definition of Done:
1. Code change committed
2. All project-level tests pass on clean clone
3. All challenges in `challenges/scripts/` pass on running host
4. Governance docs coherent with change

---

## 13. Build/Deploy Information

### 13.1 Build Commands (Makefile v3.0.0)

```bash
make deps             # go mod download + tidy
make build            # builds grpc-server, api-server, unified-translator into ./build/
make build-all        # cross-compile: Linux/macOS/Windows (amd64 + arm64) into ./dist/
make test             # go test -v ./...
make test-coverage    # go test -v -cover ./...
make fmt              # go fmt ./...
make vet              # go vet ./...
make quick-test       # fmt + vet + test
make dev              # grpc-server (:50051) + api-server (:8080) in debug
make run-system       # builds then runs grpc + api
make docker-build     # docker build -t translator-system:v3.0.0
make docker-run       # docker run -p 50051:50051 -p 8080:8080
make clean            # rm -rf build/ dist/ *.log
```

### 13.2 Docker Build

**Dockerfile** (multi-stage):
- **Builder:** `golang:1.24-alpine` with `git`, `make`
- **Production:** `alpine:latest` with `ca-certificates`, `tzdata`
- Non-root user: `appuser` (UID 1000)
- Exposes: port 8443 (TCP + UDP for HTTP/3)
- Health check: `https://localhost:8443/health`
- Entry: `/app/translator-server -config /app/config/config.json`

### 13.3 Docker Compose Stack

```yaml
services:
  postgres:          # PostgreSQL 16-alpine (port 5432)
  redis:             # Redis 7-alpine (port 6379, 512MB max)
  translator-api:    # API server (port 8443 + 8080)
  adminer:           # [Optional] DB management UI (port 8081)
  redis-commander:   # [Optional] Redis management UI (port 8082)
```

### 13.4 Cross-Compilation Targets
- Linux AMD64, Linux ARM64
- macOS AMD64, macOS ARM64
- Windows AMD64

### 13.5 CI/Validation
```bash
make ci-validate-all    # no-silent-skips-warn + demo-all-warn
make no-silent-skips    # bash scripts/no-silent-skips.sh
make demo-all           # bash scripts/demo-all.sh
```

---

## 14. Dependencies (Key Roles)

### Direct Dependencies (go.mod)

| Dependency | Role |
|-----------|------|
| `gin-gonic/gin` | HTTP REST API framework |
| `gorilla/websocket` | WebSocket real-time monitoring |
| `quic-go/quic-go` | HTTP/3 (QUIC) transport |
| `golang-jwt/jwt/v5` | JWT authentication |
| `google.golang.org/grpc` | gRPC service framework |
| `lib/pq` | PostgreSQL driver |
| `mattn/go-sqlite3` | SQLite driver |
| `redis/go-redis/v9` | Redis caching |
| `unidoc/unioffice` | DOCX document processing |
| `unidoc/unipdf/v3` | PDF document processing |
| `spf13/cobra` | CLI framework |
| `stretchr/testify` | Testing assertions & mocks |
| `google/uuid` | UUID generation |
| `golang.org/x/text` | Text processing & normalization |
| `golang.org/x/crypto` | TLS, crypto primitives |
| `golang.org/x/time` | Rate limiting |
| `gopkg.in/yaml.v3` | YAML config parsing |

### Replace Directives
```go
replace digital.vasic.challenges => ./Challenges
replace digital.vasic.containers => ./Containers
```

---

## 15. Configuration

### 15.1 Main Config: `config.json`

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8443,
    "enable_http3": true,
    "tls_cert_file": "certs/server.crt",
    "tls_key_file": "certs/server.key",
    "read_timeout": 30,
    "write_timeout": 30,
    "max_upload_size": 104857600  // 100MB
  },
  "security": {
    "enable_auth": true,
    "jwt_secret": "",
    "api_key_header": "X-API-Key",
    "rate_limit_rps": 10,
    "rate_limit_burst": 20,
    "cors_origins": ["*"]
  },
  "translation": {
    "default_provider": "openai",
    "cache_enabled": true,
    "cache_ttl": 3600,
    "max_concurrent": 5,
    "providers": {}
  },
  "preparation": {
    "enabled": true,
    "pass_count": 2,
    "providers": ["deepseek", "anthropic"],
    "analyze_content_type": true,
    "analyze_characters": true,
    "analyze_terminology": true,
    "analyze_culture": true,
    "analyze_chapters": true,
    "detail_level": "standard"
  },
  "distributed": {
    "enabled": false,
    "ssh_timeout": 30,
    "ssh_max_retries": 3,
    "health_check_interval": 30,
    "max_remote_instances": 20,
    "workers": {}
  },
  "logging": {
    "level": "info",
    "format": "json"
  }
}
```

### 15.2 Environment Variables

```bash
# Server
MONITOR_SERVER_PORT=8090
LOG_LEVEL=info          # debug, info, warn, error

# LLM API Keys
OPENAI_API_KEY=...
ANTHROPIC_API_KEY=...
DEEPSEEK_API_KEY=...
ZHIPU_API_KEY=...
QWEN_API_KEY=...
GEMINI_API_KEY=...

# SSH Workers
SSH_WORKER_HOST=localhost
SSH_WORKER_USER=milosvasic
SSH_WORKER_PASSWORD=...
SSH_WORKER_PORT=22
SSH_WORKER_REMOTE_DIR=/tmp/translate-ssh

# JWT
JWT_SECRET=...          # min 16 chars
```

### 15.3 Provider-Specific Configs
- `internal/working/config.openai.json`
- `internal/working/config.distributed.json`
- `internal/working/config.worker*.json`
- Various LLM provider configs in `internal/working/`

### 15.4 TLS Certificates
- `certs/server.crt` — TLS certificate (required for HTTP/3)
- `certs/server.key` — TLS private key (required for HTTP/3)

---

## 16. Existing QA/Testing References

### HelixQA References
No explicit references to "HelixQA" were found in any of the analyzed files (README.md, CLAUDE.md, AGENTS.md, CONSTITUTION.md, go.mod, Makefile).

### Testing Infrastructure Found
1. **Test suites** in `test/` and `tests/` directories
2. **Mock implementations** in `test/mocks/` using testify/mock
3. **Build tags** for integration and e2e tests
4. **Coverage tracking** at ~43.6% with `coverage.html` artifacts committed
5. **CI validation** via `make ci-validate-all` (no-silent-skips + demo-all)
6. **Challenge scripts** in `challenges/scripts/` for CI gates:
   - `no_suspend_calls_challenge.sh` — Static source tree scanner
   - `host_no_auto_suspend_challenge.sh` — Host state verification

### Testing Gaps Identified
- No explicit HelixQA integration or framework
- Coverage at 43.6% — significant room for improvement
- The `Documentation/CLAUDE.md` references a "legacy Python implementation" — testing for Python components may exist but is not authoritative
- Many `*_test.go` patterns exist but coverage is described as approximate

---

## 17. Architecture Summary

### Event-Driven Architecture
```
Translation CLI/Client
    │
    ├──► EventBus (pkg/events/events.go) — Central pub/sub
    │       ├──► WebSocket Hub (pkg/websocket/hub.go) — Dashboard clients
    │       ├──► Progress tracking
    │       └──► Error tracking
    │
    ├──► SSH Workers (pkg/distributed/)
    │       ├── Coordinator (work distribution)
    │       ├── SSH Pool (connection management)
    │       ├── Version Manager (sync enforcement)
    │       └── Fallback (graceful degradation)
    │
    └──► Storage (pkg/storage/)
            ├── PostgreSQL (persistent)
            ├── Redis (cache)
            └── SQLite (alternative)
```

### Translation Pipeline
```
Input File
    │
    ▼
Format Detection (pkg/format/)
    │
    ▼
Parsing (pkg/ebook/*_parser.go)
    │
    ▼
Preparation Phase (pkg/preparation/) ← Optional
    │
    ▼
Translation (pkg/translator/llm/) → LLM Provider
    │
    ▼
Verification (pkg/verification/) ← Optional
    │
    ▼
Output Generation (pkg/ebook/*_writer.go)
    │
    ▼
Translated File
```

---

## 18. Key Patterns & Conventions

### Go Conventions
- Module-local imports: `digital.vasic.translator/...`
- Error wrapping: `fmt.Errorf("...: %w", err)`
- Prefer `any` over `interface{}`
- Line length cap: 140 (`lll` linter)
- `cmd/` exempted from `funlen` and `gochecknoinits`
- `pkg/` exempted from `gochecknoinits`

### Import Cycle Prevention
- `pkg/translator/llm/` uses type alias to avoid cycles with `pkg/translator/`
- `TranslationConfig = translator.TranslationConfig`

### Factory Pattern
- `NewLLMTranslator()` validates provider+model and creates appropriate client
- Config-based instantiation with env var overrides

### Distributed Version Compatibility
- Worker changes must stay version-compatible
- Protocol/message changes require coordinated updates via `version_manager.go`

---

*End of Analysis*
