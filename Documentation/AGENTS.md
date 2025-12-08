# AGENTS.md - Universal Multi-Format Multi-Language Ebook Translation System

## Build/Lint/Test Commands

### Core Commands
- **Build all**: `make build` (builds gRPC, API, and CLI binaries)
- **Build specific**: `make build-grpc`, `make build-api`, `make build-cli`
- **Build for all platforms**: `make build-all`
- **Test all**: `make test` or `go test ./... -v`
- **Test with coverage**: `make test-coverage` (generates coverage.html)
- **Test single package**: `go test -v ./pkg/package`
- **Test specific function**: `go test -v -run TestFunctionName ./pkg/package`
- **Lint**: `golangci-lint run` (uses .golangci.yml configuration)
- **Format**: `make fmt` (go fmt)
- **Vet**: `make vet` (go vet)
- **Quick test cycle**: `make quick-test` (fmt + vet + test)
- **Pre-commit**: `make pre-commit`

### Running Services
- **Development**: `make dev` (starts gRPC on 50051 and API on 8080 in debug mode)
- **Full system**: `make run-system` (builds and runs both servers)
- **gRPC server only**: `make run-grpc`
- **API server only**: `make run-api`

### Docker
- **Build**: `make docker-build`
- **Run**: `make docker-run`
- **Full stack**: `docker-compose up -d` (includes PostgreSQL, Redis, monitoring)

## Project Architecture

### Module Information
- **Go version**: 1.25.2
- **Module**: `digital.vasic.translator`
- **Entry points**: Multiple commands in `cmd/` directory:
  - `cmd/cli/main.go` - Basic CLI tool
  - `cmd/server/main.go` - Main REST API server
  - `cmd/grpc-server/main.go` - gRPC server (port 50051)
  - `cmd/api-server/main.go` - API server (port 8080)
  - `cmd/monitor-server/main.go` - WebSocket monitoring server (port 8090)
  - `cmd/unified-translator/main.go` - **Primary unified CLI tool** with full provider support
  - `cmd/translate-ssh/main.go` - SSH translation worker
  - `cmd/preparation-translator/main.go` - Preparation phase translator
  - `cmd/markdown-translator/main.go` - Markdown workflow translator
  - `cmd/ebook-translator/main.go` - Specialized ebook translator
  - `cmd/deployment/main.go` - Deployment management tool

### Core Packages Structure
```
pkg/
├── ebook/          # Universal ebook parsing (FB2, EPUB, TXT, HTML, PDF, DOCX)
├── translator/     # Translation engines and LLM providers
│   └── llm/       # LLM providers (OpenAI, Anthropic, Zhipu, DeepSeek, Ollama, Qwen, Gemini, LlamaCpp)
├── format/        # Format detection and validation
├── language/      # Language detection
├── api/          # REST API handlers and WebSocket
├── distributed/  # Distributed processing coordination
├── security/     # JWT auth, rate limiting
├── events/       # Event bus system
├── storage/      # Database abstraction (PostgreSQL, Redis, SQLite)
├── verification/ # Translation quality verification
├── markdown/     # EPUB to Markdown conversion workflow
├── preparation/  # Pre-translation preparation phase
├── websocket/    # WebSocket hub and connections
├── deployment/   # Docker and SSH deployment
├── coordination/ # Multi-LLM coordination
├── batch/        # Batch processing
├── hardware/     # Hardware detection for optimization
├── script/       # Script conversion (Cyrillic/Latin)
├── progress/     # Progress tracking
├── report/       # Report generation
├── cache/        # Translation caching
├── logger/       # Logging utilities
├── fb2/          # FB2 format specific handling
├── grpc/         # gRPC service definitions
└── models/       # Data models and registry
```

### Configuration System
- **Config files**: JSON format (see `config.json` or files in `internal/working/`)
- **Environment variables**: Use for API keys and secrets
- **Internal config**: `internal/config/config.go` handles loading and validation
- **TLS certs**: Required for HTTPS/HTTP3 (certs stored in `certs/` directory)
- **Docker config**: Environment-based configuration in `docker-compose.yml`

## Code Style Guidelines

### Go Standards
- **Naming**: PascalCase for exported, camelCase for unexported
- **Imports**: Standard library → third-party → local packages (alphabetical)
- **Types**: Use `any` instead of `interface{}`, interfaces for behavior
- **Error handling**: Explicit returns, wrap with context: `fmt.Errorf("failed: %w", err)`
- **Comments**: Document exported functions/types, avoid obvious comments
- **Security**: Never hardcode API keys, use environment variables

### Testing Patterns
- **Table-driven tests**: Preferred for unit tests with struct-based test cases
- **Naming**: `TestFunctionName_Scenario`
- **Build tags**: Use `//go:build integration`, `//go:build e2e` for selective test execution
- **Test locations**: 
  - `test/unit/` - Unit tests for individual components
  - `test/integration/` - Cross-package integration tests
  - `test/e2e/` - End-to-end tests with real scenarios
  - `test/performance/` - Performance benchmarks
  - `test/stress/` - Load testing
  - `test/security/` - Security-focused tests
  - `test/distributed/` - Distributed system tests
- **Mocking**: Create mock implementations in test files using testify/mock (see `test/mocks/`)
- **Coverage**: Current overall coverage is approximately 43.6%, HTML reports available at `coverage.html`
- **Test utilities**: Helper functions in `test/utils/helpers.go` for creating test files and configurations
- **Fixtures**: Test data in `test/fixtures/ebooks/`, `test/fixtures/configs/`, `test/fixtures/translations/`

## Essential Commands

### Development Workflow
```bash
# Initial setup
make deps

# Development cycle
make build
make test
make fmt
make vet

# Run locally
make dev        # Development environment with both servers
make run-grpc    # gRPC server only
make run-api     # API server only
make run-system  # Full system

# Test specific packages
go test ./pkg/markdown -v
go test ./pkg/format -v
go test ./pkg/distributed -v
```

### Translation Operations
```bash
# CLI tools (build first)
make build-cli

# Basic translation
./build/unified-translator -input book.fb2 -output book_sr.epub

# With specific LLM provider
./build/unified-translator -i book.fb2 -provider openai -model gpt-4 -api-key YOUR_KEY

# Local llama.cpp translation
./build/unified-translator -i book.fb2 -provider llamacpp -llama-model ./model.gguf

# SSH worker translation
./build/unified-translator -i book.fb2 -provider ssh -ssh-host worker.local -ssh-user user -ssh-password pass

# With monitoring
./build/unified-translator -i book.fb2 -provider openai -monitoring -monitoring-port 8080

# Language detection
./build/unified-translator -input book.txt -detect-lang

# Markdown workflow
./build/markdown-translator -input book.epub -output book.md

# Preparation phase
./build/preparation-translator -input book.epub -output book_sr.epub

# SSH worker (standalone)
./build/translate-ssh -config config.worker.json
```

### Unified Translator CLI Flags
- **Input/Output**: `-input, -i`, `-output, -o`
- **Language**: `-source-lang`, `-target-lang`, `-script` (cyrillic/latin)
- **Provider**: `-provider` (openai, anthropic, zhipu, deepseek, qwen, gemini, ollama, llamacpp, ssh)
- **Provider Config**: `-model`, `-api-key`, `-base-url`, `-temperature`, `-max-tokens`, `-timeout`
- **SSH Config**: `-ssh-host`, `-ssh-user`, `-ssh-password`, `-ssh-port`, `-remote-dir`
- **LlamaCpp Config**: `-llama-binary`, `-llama-model`, `-context-size`
- **Execution**: `-workers`, `-chunk-size`, `-concurrency`, `-verify`, `-verbose`
- **Monitoring**: `-monitoring`, `-monitoring-port`

### API Operations
```bash
# Start API server
make run-api

# Development mode
make dev

# Docker deployment
make docker-build && make docker-run

# Test API (examples in api/examples/)
curl -X POST https://localhost:8443/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{"input_file": "book.fb2", "provider": "openai"}'

# WebSocket monitoring
./build/monitor-server
# Then visit http://localhost:8090/monitor
```

### Distributed Processing
```bash
# Docker compose for full stack
docker-compose up -d

# Deploy to workers (scripts in internal/scripts/)
./internal/scripts/deploy_worker.sh
./internal/scripts/deploy_system.sh

# Monitor distributed system
./scripts/monitor_production.sh
./scripts/monitor_translation.sh

# Check worker status
./internal/scripts/check_worker.sh
./internal/scripts/check_health.sh
```

## Key Patterns and Conventions

### Event-Driven Architecture
- **Event bus**: Central `pkg/events/events.go` for system-wide communication
- **Event types**: `translation_started`, `translation_progress`, `translation_completed`
- **WebSocket integration**: Real-time progress updates via `pkg/websocket/`

### Translation Pipeline
1. **Format detection** (`pkg/format/detector.go`)
2. **Parsing** (`pkg/ebook/parser.go` → format-specific parsers)
3. **Translation** (`pkg/translator/translator.go`)
4. **Output generation** (format-specific writers)

### LLM Provider Pattern
All LLM providers implement `pkg/translator/llm/llm.go:LLMClient` interface:
```go
type LLMClient interface {
    Translate(ctx context.Context, text string, prompt string) (string, error)
    GetProviderName() string
}
```

**Supported Providers**:
- **API-based**: OpenAI, Anthropic, Zhipu, DeepSeek, Qwen, Gemini
- **Self-hosted**: Ollama, LlamaCpp (local binary execution)
- **Distributed**: SSH workers for remote processing
- **Factory Pattern**: `NewLLMTranslator()` validates provider/model and creates appropriate client

**Provider Configuration**:
- JSON config in `config.json` under `translation.providers` section
- Provider-specific config files (e.g., `config_openai.json`)
- Environment variable overrides for API keys

### Configuration Pattern
- Struct-based config in `internal/config/config.go`
- JSON file loading with environment variable overrides
- Validation in config initialization
- Provider-specific config files (e.g., `config_openai.json`)

## Critical Implementation Notes

### Translation Quality
- **Multi-pass verification**: `pkg/verification/multipass.go`
- **Reference translations**: Cache high-quality translations
- **Script support**: Serbian Cyrillic ↔ Latin conversion in `pkg/script/`

### Distributed System
- **SSH-based deployment**: `pkg/distributed/ssh_pool.go`
- **Worker coordination**: `pkg/distributed/coordinator.go`
- **Fallback mechanisms**: `pkg/distributed/fallback.go`
- **Version management**: `pkg/distributed/version_manager.go`

### Security Requirements
- **TLS required**: All API communication uses HTTPS/HTTP3
- **JWT authentication**: Configurable in `security.enable_auth`
- **Rate limiting**: Configurable RPS and burst limits
- **API key security**: Never commit, use environment variables

### Performance Considerations
- **Translation caching**: Redis/SQLite-based caching system
- **Concurrent processing**: Configurable `translation.max_concurrent`
- **Memory management**: Streaming for large files
- **Connection pooling**: Database and HTTP client pooling

## File Organization

### Input/Output Patterns
- **Input formats**: FB2, EPUB, TXT, HTML, PDF, DOCX
- **Output formats**: Same as input, plus format conversion
- **Naming**: `original_name_sr.epub` for Serbian translations
- **Temporary files**: Use `os.TempDir()` for intermediate files

### Testing Data
- **Test files**: `test/` directory with sample ebooks
- **Mock data**: Create test-specific data in test functions
- **E2E tests**: Use real translation providers with API keys

## Gotchas and Non-Obvious Patterns

### FB2 XML Handling
- **Namespaces**: Must register FB2 namespace before parsing
- **Encoding**: Always use UTF-8
- **Structure preservation**: Maintain XML hierarchy during translation

### LLM Provider Quirks
- **Rate limits**: Each provider has different limits
- **Context windows**: Split large texts appropriately
- **API key formats**: Vary by provider (check documentation)
- **Retry logic**: Implement exponential backoff

### Distributed System Complexity
- **SSH keys**: Required for worker deployment
- **Network discovery**: Workers must be reachable from coordinator
- **Version synchronization**: All workers must run same version
- **Health monitoring**: Continuous health checks required

### Performance Tuning
- **Batch size**: Adjust based on document size and provider limits
- **Memory usage**: Monitor with large ebook translations
- **Concurrent requests**: Balance between speed and rate limits
- **Cache hit rates**: Monitor cache effectiveness

## Event-Driven Architecture

### EventBus System
- **Core**: `pkg/events/events.go` - Central pub/sub system with thread-safe handler management
- **Event Types**: Predefined constants for translation lifecycle:
  - `EventTranslationStarted`
  - `EventTranslationProgress` 
  - `EventTranslationCompleted`
  - `EventTranslationError`
  - `EventConversionStarted`
  - `EventConversionProgress`
  - `EventConversionCompleted`
  - `EventConversionError`
- **Subscription**: 
  - `Subscribe(eventType, handler)` - Type-specific subscriptions
  - `SubscribeAll(handler)` - Global event listeners
- **WebSocket Integration**: `pkg/websocket/hub.go` automatically subscribes to all events and broadcasts to clients
- **Session Tracking**: Events filtered by session ID for per-client monitoring

### Usage Patterns
```go
// Emit events with session tracking
EmitProgress(eventBus, sessionID, "Translating chapter 5/10", map[string]interface{}{
    "current": 5, "total": 10, "progress": 0.5,
})

// Subscribe to events in your component
eventBus.Subscribe(EventTranslationProgress, func(event Event) {
    log.Printf("Progress: %s", event.Message)
})
```

## Advanced Features

### Preparation Phase System
- **Purpose**: Pre-translation analysis to improve quality
- **Binary**: `./build/preparation-translator` (build with `go build ./cmd/preparation-translator`)
- **Features**: Content analysis, character patterns, cultural references
- **Usage**: 
  ```bash
  ./build/preparation-translator -input book.epub -output book_sr.epub -analysis book_analysis.json
  ```

### Multi-Pass Translation
- **Verification**: `pkg/verification/multipass.go` implements quality checks
- **Polishing**: Automatic post-translation improvements
- **Reference caching**: Stores high-quality translations for reuse

### Markdown Workflow
- **EPUB to Markdown**: `pkg/markdown/epub_to_markdown.go`
- **Markdown to EPUB**: `pkg/markdown/markdown_to_epub.go`
- **Benefits**: Easier manual editing and review
- **Binary**: `./build/markdown-translator`

### Batch Processing
- **Directory support**: Process multiple files automatically
- **Scripts**: `scripts/batch_translate_*.sh` for different providers
- **Monitoring**: `scripts/monitor_translation.sh` for progress tracking

## Troubleshooting

### Common Issues
1. **TLS Certificate Errors**: Run `make generate-certs` before starting server
2. **API Rate Limits**: Check provider-specific limits in config
3. **Memory Leaks**: Monitor large file translations with `top` or `htop`
4. **SSH Connection Failures**: Verify key-based auth for distributed deployment
5. **FB2 Parsing Errors**: Ensure UTF-8 encoding and valid XML structure

### Debug Mode
- **Logging**: Set `logging.level` to `debug` in config
- **Verbose output**: Use `-v` flag with CLI tools
- **Event monitoring**: WebSocket events show real-time progress
- **Debug scripts**: Various `debug_*.sh` scripts for troubleshooting specific issues
- **Test utilities**: `test_*` scripts for isolated component testing

### Development Patterns
- **Code style**: Follow Go standards, use golangci-lint configuration
- **Error handling**: Explicit returns with context wrapping
- **Testing**: Table-driven tests with sub-tests, comprehensive mocking
- **Events**: Use event bus for component communication and progress tracking
- **Sessions**: Unique session IDs for tracking translation operations
- **Configuration**: JSON-based with environment variable overrides

### Performance Monitoring
- **Built-in metrics**: Available via `/api/v1/metrics` endpoint
- **Health checks**: `/api/v1/health` endpoint
- **Prometheus integration**: Metrics export available

## Version Information
- **Current version**: 2.3.0 (see `VERSION` file)
- **System version**: 3.0.0 (in Makefile)
- **API versioning**: Follows semantic versioning
- **Backward compatibility**: Maintained within major versions
- **Version tracking**: Distributed system requires version synchronization across workers

## Quick Reference

### Essential Files
- **VERSION**: Current application version (2.3.0)
- **go.mod**: Module definition and dependencies
- **Makefile**: Build, test, and development commands
- **docker-compose.yml**: Full production stack deployment
- **.golangci.yml**: Linting configuration (if golangci-lint is installed)
- **internal/config/config.go**: Configuration structure and loading
- **pkg/events/events.go**: Event system for real-time updates
- **pkg/websocket/hub.go**: WebSocket hub for monitoring

### Important Scripts
- **internal/scripts/**: Production and deployment scripts
- **scripts/**: Utility and demonstration scripts
- **test/fixtures/**: Test data and sample files

### Monitoring and Debugging
- **WebSocket Monitoring**: Use `cmd/monitor-server` for real-time progress
- **Event System**: All components emit events via `pkg/events`
- **Health Checks**: Built-in health check endpoints
- **Logging**: Structured logging with configurable levels