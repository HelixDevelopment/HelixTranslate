# AGENTS.md - Russian-Serbian FB2 Translation Project

## Project Overview

This is a comprehensive Russian to Serbian translation toolkit that supports multiple translation methods, format conversions, and both CLI and server interfaces. The project provides:

- **Multi-format support**: FB2, EPUB, TXT, HTML input/output
- **Multiple translation providers**: Dictionary replacement, Google Translate, LLMs (OpenAI, Anthropic, Zhipu AI, DeepSeek, Ollama)
- **Two implementation approaches**: Legacy Python tools + Modern Go-based system
- **API server**: RESTful API with WebSocket support, caching, and authentication
- **Batch processing**: Multi-pass translation with quality verification

## Essential Commands

### Go Project Commands (Primary System)
```bash
# Build and test
make build                    # Build CLI and server binaries
make test                     # Run unit and integration tests
make test-unit                # Run unit tests only
make test-integration         # Run integration tests only
make test-e2e                 # Run end-to-end tests
make test-performance         # Run performance benchmarks
make fmt                      # Format Go code
make lint                     # Lint Go code

# Running applications
make run-cli                  # Run CLI application
make run-server               # Run server application
./build/translator            # Run CLI binary directly
./build/translator-server     # Run server binary directly

# Docker
make docker-build             # Build Docker image
make docker-run               # Run Docker container
docker-compose up -d          # Start full stack (PostgreSQL + Redis + API)
docker-compose -f docker-compose.yml --profile admin up  # Include admin tools

# Development
make dev-server               # Start server with auto-reload
make generate-certs           # Generate TLS certificates
```

### Legacy Python Commands (Compatibility)
```bash
# LLM-powered translation (highest quality)
python3 llm_fb2_translator.py book_ru.fb2 --provider openai
python3 llm_fb2_translator.py book_ru.fb2 --provider anthropic
python3 llm_fb2_translator.py book_ru.fb2 --provider zhipu
python3 llm_fb2_translator.py book_ru.fb2 --provider deepseek
python3 llm_fb2_translator.py book_ru.fb2 --provider ollama

# Traditional translation methods
python3 simple_fb2_translate.py input_ru.fb2 output_sr.b2
python3 high_quality_fb2_translator.py input_ru.fb2 output_sr.b2
python3 fb2_translator.py input_ru.fb2

# Format conversion
python3 fb2_to_epub.py input_sr.b2 output_sr.epub
python3 fb2_to_pdf.py input_sr.b2 output_sr.pdf
```

## Code Organization and Structure

### Go Architecture (Modern System)
```
cmd/                        # Entry points
├── cli/main.go             # CLI application
├── server/main.go          # API server
├── preparation-translator/
└── markdown-translator/

pkg/                        # Core packages
├── translator/             # Translation engine
│   ├── llm/               # LLM providers (OpenAI, Anthropic, Zhipu, DeepSeek, Ollama)
│   └── dictionary/        # Dictionary-based translation
├── ebook/                 # E-book parsing (FB2, EPUB, TXT, HTML)
├── api/                   # HTTP handlers and routing
├── storage/               # Database abstraction (PostgreSQL, SQLite, Redis)
├── security/              # Authentication and rate limiting
├── websocket/             # WebSocket hub for real-time updates
├── verification/          # Multi-pass translation verification
├── preparation/           # Translation preparation phase
├── markdown/              # Markdown workflow
├── batch/                 # Batch processing
└── coordination/          # Multi-LLM coordination

internal/                   # Internal packages
├── config/                # Configuration management
└── cache/                 # Caching layer

test/                      # Tests
├── unit/                  # Unit tests
├── integration/           # Integration tests
├── e2e/                   # End-to-end tests
├── performance/           # Performance tests
└── stress/                # Stress tests
```

### Legacy Python Structure
```
Legacy/                     # Legacy Python tools
├── llm_fb2_translator.py   # LLM-powered translation
├── high_quality_fb2_translator.py  # Google Translate with caching
├── simple_fb2_translate.py  # Dictionary replacement
├── fb2_translator.py        # Template-based translation
└── fb2_to_epub.py, fb2_to_pdf.py  # Format conversion
```

## Configuration

### Environment Variables (Required for LLM providers)
```bash
# LLM Provider API Keys
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"

# Database (for Go server)
export POSTGRES_USER="translator"
export POSTGRES_PASSWORD="secure_password"
export POSTGRES_DB="translator"
export REDIS_PASSWORD="redis_secure_password"
```

### Configuration Files
- `config.json` - Main server configuration
- `config_*.json` - Provider-specific configurations (OpenAI, Anthropic, Zhipu, DeepSeek, Ollama)
- `.env` - Environment variables (should be in .gitignore)

## Code Style and Patterns

### Go Conventions
- **Module name**: `digital.vasic.translator`
- **Go version**: 1.25.2
- **Package structure**: Standard Go project layout
- **Naming**: PascalCase for exported, camelCase for unexported
- **Interfaces**: Define behavior, use composition over inheritance
- **Error handling**: Explicit error returns, use wrapped errors with context

### Testing Patterns
```go
// Unit test naming: Test[FunctionName][Scenario]
func TestTranslatorTranslateText_Success(t *testing.T) {
    // Arrange
    // Act
    // Assert
}

// Integration test tag: //go:build integration
// E2E test tag: //go:build e2e
// Performance test tag: //go:build performance
```

### Dependency Injection
- Use constructor injection for dependencies
- Define interfaces for external services
- Use dependency injection container for complex graphs

## Translation Pipeline

### Multi-Pass Translation System
1. **Preparation Phase**: Content analysis and preparation
2. **Translation Phase**: Core translation using selected provider
3. **Verification Phase**: Quality assessment and polishing
4. **Final Polish**: Final output generation

### Provider Quality Hierarchy
1. **LLM (GPT-4/Claude/Zhipu/DeepSeek)**: Professional literary quality
2. **Google Translate**: Basic translation with context awareness
3. **Dictionary**: Fast, word-for-word replacement

## Key Architectural Patterns

### Translation Interface
```go
type Translator interface {
    Translate(ctx context.Context, text string, options TranslationOptions) (*TranslationResult, error)
    TranslateBatch(ctx context.Context, texts []string, options TranslationOptions) ([]*TranslationResult, error)
    GetProviderInfo() ProviderInfo
}
```

### Event System
- Uses event-driven architecture for real-time updates
- WebSocket hub for progress notifications
- Event sourcing for translation history

### Caching Strategy
- **Redis**: Distributed caching for translation results
- **SQLite**: Local fallback and development
- **PostgreSQL**: Persistent storage and analytics

### Storage Abstraction
```go
type Storage interface {
    StoreTranslation(ctx context.Context, translation *Translation) error
    GetTranslation(ctx context.Context, id string) (*Translation, error)
    // ... other methods
}
```

## Critical Implementation Details

### FB2 Processing
- **Namespace**: Always register `http://www.gribuser.ru/xml/fictionbook/2.0`
- **Encoding**: UTF-8 for all file operations
- **Structure**: Preserve XML hierarchy and formatting
- **Metadata**: Update language field to 'sr' for Serbian

### Security Requirements
- **API Keys**: NEVER hardcode in source code - use environment variables
- **Authentication**: JWT-based auth with configurable providers
- **TLS**: Required for all API communications
- **Rate Limiting**: Per-client rate limiting with Redis backend

### Error Handling
- Use wrapped errors with context: `fmt.Errorf("translation failed: %w", err)`
- Implement retry logic with exponential backoff
- Graceful degradation when providers are unavailable
- Comprehensive logging with structured format

### Performance Considerations
- **Concurrent Processing**: Use goroutines with worker pools
- **Memory Management**: Stream processing for large files
- **Caching**: Multi-level caching (in-memory → Redis → database)
- **Batch Operations**: Minimize API calls through batching

## Testing Approach

### Test Categories
1. **Unit Tests**: Fast, isolated component tests
2. **Integration Tests**: Database and external service integration
3. **E2E Tests**: Full workflow testing with real files
4. **Performance Tests**: Benchmarking and load testing
5. **Stress Tests**: High-load scenarios and resource limits

### Test Data
- Sample FB2 files in test fixtures
- Mock LLM responses for consistent testing
- Database migrations for test isolation

### Coverage Requirements
- Minimum 80% code coverage for new features
- 100% coverage for critical security paths
- Integration tests for all external APIs

## Development Workflow

### Before Making Changes
1. Run existing tests: `make test`
2. Check code formatting: `make fmt`
3. Run linting: `make lint`
4. Create feature branch from main

### After Making Changes
1. Run full test suite: `make test test-integration test-e2e`
2. Check performance impact: `make test-performance`
3. Update documentation if needed
4. Ensure backward compatibility

### API Changes
1. Update OpenAPI spec in `api/openapi/openapi.yaml`
2. Update Postman collection in `api/examples/postman/`
3. Add integration tests for new endpoints
4. Update API documentation in `Documentation/API.md`

## Deployment and Operations

### Docker Deployment
- **Multi-stage builds**: Optimize image size
- **Health checks**: `/health` endpoint for container orchestration
- **Secrets management**: Environment variables for sensitive data
- **Volume mounts**: Persistent data and certificates

### Monitoring
- **Health checks**: HTTP/3, database, and Redis connectivity
- **Metrics**: Translation statistics, error rates, performance
- **Logging**: Structured JSON logging with correlation IDs
- **Alerting**: Failed translations, provider outages

### Scaling
- **Horizontal scaling**: Stateless API server design
- **Database**: PostgreSQL with connection pooling
- **Caching**: Redis cluster for distributed caching
- **Load balancing**: HTTP/3 support with QUIC

## Common Gotchas

### LLM Provider Issues
- **Rate Limits**: Implement exponential backoff and retries
- **Token Limits**: Split large texts into chunks (max 20KB per chunk)
- **Context Window**: Track token usage and manage context efficiently
- **Cost Management**: Monitor API usage and implement limits

### FB2 Processing
- **XML Namespaces**: Must be registered before parsing
- **Text Encoding**: Always UTF-8, never assume locale
- **Structure Preservation**: Maintain original hierarchy and formatting
- **Metadata Updates**: Update language but preserve other metadata

### Performance Pitfalls
- **Memory Leaks**: Proper cleanup of goroutines and resources
- **Database Connections**: Use connection pooling and proper timeout handling
- **Large Files**: Stream processing to avoid memory exhaustion
- **Caching**: Cache invalidation strategy is critical

This AGENTS.md should be updated whenever:
- New translation providers are added
- Major architectural changes are made
- New testing patterns are introduced
- Deployment or scaling requirements change