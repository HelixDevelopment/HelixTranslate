# Universal Ebook Translator

## Project Overview

This is a high-performance, enterprise-grade universal ebook translation toolkit written in Go. It supports **any ebook format** and **any language pair**, featuring multiple translation engines, REST API with HTTP/3 support, and real-time WebSocket events.

### Key Features
- **Universal Format Support**: FB2, EPUB, TXT, HTML (auto-detected input), EPUB (default), TXT output
- **Universal Language Support**: Any language pair supported with automatic language detection and 18+ pre-configured languages
- **Multiple Translation Engines**: Dictionary (offline), OpenAI GPT, Anthropic Claude, Zhipu AI, DeepSeek, Ollama
- **Modern Architecture**: Built with Go, REST API using Gin Gonic, HTTP/3 (QUIC) support, WebSocket for real-time progress
- **Security & Performance**: JWT authentication, rate limiting, TLS 1.3 encryption, translation caching, concurrent processing

### Architecture
The project is structured into several key components:
- `cmd/cli` - Command-line interface for translating ebooks
- `cmd/server` - REST API server with HTTP/3 and WebSocket support
- `pkg/ebook` - Universal ebook parsing and writing (EPUB, FB2, TXT, HTML)
- `pkg/translator` - Translation engine abstraction with multiple providers
- `pkg/language` - Language detection and management
- `pkg/events` - Event-driven architecture for progress reporting
- `pkg/websocket` - Real-time progress tracking

## Building and Running

### Prerequisites
- Go 1.21 or higher
- Make (optional)
- OpenSSL (for TLS certificates)

### Building from Source
```bash
# Clone and build
make deps
make build

# Binaries will be in build/
# translator (CLI)  translator-server (REST API)
```

### Docker
```bash
# Build and run with Docker
make docker-build
make docker-run
```

### Running the CLI
```bash
# Translate any ebook to Serbian (auto-detect source language)
./build/translator -input book.epub

# Translate EPUB to German
./build/translator -input book.epub -locale de

# Translate with OpenAI GPT-4
export OPENAI_API_KEY="your-key"
./build/translator -input book.txt -locale es -provider openai -model gpt-4
```

### Running the REST API Server
```bash
# Generate TLS certificates for HTTP/3
make generate-certs

# Start server
./build/translator-server

# Server starts on:
# - HTTP/3 (QUIC): https://localhost:8443
# - HTTP/2 (fallback): https://localhost:8443
# - WebSocket: wss://localhost:8443/ws
```

### Available Make Commands
- `build` - Build CLI and server binaries
- `build-cli` - Build CLI binary only
- `build-server` - Build server binary only
- `test` - Run all tests
- `test-unit` - Run unit tests
- `test-integration` - Run integration tests
- `fmt` - Format code
- `lint` - Lint code
- `docker-build` - Build Docker image
- `docker-run` - Run Docker container
- `generate-certs` - Generate self-signed TLS certificates

## Development Conventions

### Coding Style
- Follow Go idioms and best practices
- Use context.Context for cancellation and timeouts
- Implement error wrapping with %w verb
- Use structured logging with appropriate levels
- Organize code by domain in separate packages

### Testing
- Unit tests in `test/unit/` directory
- Integration tests in `test/integration/` directory
- End-to-end tests in `test/e2e/` directory
- Performance tests in `test/performance/` directory
- Use `make test` to run all tests

### Project Structure
- `api/` - API clients and documentation
- `cmd/` - Main applications (CLI and server)
- `Documentation/` - Project documentation
- `internal/` - Internal packages
- `pkg/` - Public packages that could be used by other projects
- `scripts/` - Build and deployment scripts
- `test/` - Test files