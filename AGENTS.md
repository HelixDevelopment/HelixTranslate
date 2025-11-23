# AGENTS.md - Universal Multi-Format Multi-Language Ebook Translation System

## Build/Lint/Test Commands
- **Build**: `make build` or `go build ./cmd/cli`
- **Test all**: `make test-unit test-integration test-e2e`
- **Test single**: `go test -v -run TestFunctionName ./pkg/package`
- **Lint**: `make lint` (golangci-lint)
- **Format**: `make fmt` (go fmt)
- **Coverage**: `make test-coverage` (generates coverage.html)
- **Docker**: `make docker-build && make docker-run`
- **Distributed deployment**: `make deploy` (requires deployment-plan.json)
- **Monitor**: `make monitor` for production health checks

## Project Architecture

### Module Information
- **Go version**: 1.25.2
- **Module**: `digital.vasic.translator`
- **Entry points**: `cmd/cli/main.go` (CLI tool), `cmd/server/main.go` (REST API)

### Core Packages Structure
```
pkg/
├── ebook/          # Universal ebook parsing (FB2, EPUB, TXT, HTML)
├── translator/     # Translation engines (dictionary, Google, LLM)
│   └── llm/       # LLM providers (OpenAI, Anthropic, Zhipu, DeepSeek, Ollama)
├── format/        # Format detection and validation
├── language/      # Language detection
├── api/          # REST API handlers and WebSocket
├── distributed/  # Distributed processing coordination
├── security/     # JWT auth, rate limiting
├── events/       # Event bus system
├── storage/      # Database abstraction (PostgreSQL, Redis, SQLite)
└── verification/ # Translation quality verification
```

### Configuration System
- **Config files**: JSON format (see `config.json` for template)
- **Environment variables**: Use for API keys and secrets
- **Internal config**: `internal/config/config.go` handles loading and validation
- **TLS certs**: Required for HTTPS/HTTP3, generate with `make generate-certs`

## Code Style Guidelines

### Go Standards
- **Naming**: PascalCase for exported, camelCase for unexported
- **Imports**: Standard library → third-party → local packages (alphabetical)
- **Types**: Use `any` instead of `interface{}`, interfaces for behavior
- **Error handling**: Explicit returns, wrap with context: `fmt.Errorf("failed: %w", err)`
- **Comments**: Document exported functions/types, avoid obvious comments
- **Security**: Never hardcode API keys, use environment variables

### Testing Patterns
- **Table-driven tests**: Preferred for unit tests
- **Naming**: `TestFunctionName_Scenario`
- **Build tags**: Use `//go:build integration`, `//go:build e2e`
- **Test locations**: `test/unit/`, `test/integration/`, `test/e2e/`, `test/performance/`, `test/stress/`
- **Mocking**: Create mock implementations in test files

## Essential Commands

### Development Workflow
```bash
# Initial setup
make deps
make generate-certs

# Development cycle
make build
make test-unit
make lint
make fmt

# Run locally
make run-cli    # CLI tool
make run-server  # API server
```

### Translation Operations
```bash
# Basic translation
./build/translator -input book.fb2 -output book_sr.epub

# With specific LLM provider
./build/translator -input book.fb2 -provider openai -model gpt-4

# Language detection
./build/translator -input book.txt -detect-lang
```

### API Operations
```bash
# Start API server
./build/translator-server -config config.json

# Test API (examples in api/examples/)
curl -X POST https://localhost:8443/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{"input_file": "book.fb2", "provider": "openai"}'
```

### Distributed Processing
```bash
# Deploy to workers
make deploy

# Monitor distributed system
make monitor-continuous

# Check worker status
./scripts/check_worker.sh
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