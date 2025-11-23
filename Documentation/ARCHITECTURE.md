# Architecture Documentation

## Overview

The Universal Multi-Format Multi-Language Ebook Translation System is a high-performance, scalable translation service built with Go, featuring:

- **CLI tool** - Command-line interface for batch translation
- **REST API** - HTTP/3 enabled REST API with WebSocket support
- **Universal format support** - FB2, EPUB, TXT, HTML, PDF, DOCX input/output
- **Multi-language translation** - 100+ languages with auto-detection
- **Multiple translation engines** - Dictionary, Google Translate, and LLM providers
- **Real-time events** - WebSocket-based progress tracking
- **Distributed processing** - Multi-LLM coordination with fallback
- **High security** - JWT authentication, rate limiting, TLS/QUIC

## Project Structure

```
digital.vasic.translator/
├── cmd/
│   ├── cli/          # CLI application
│   └── server/       # REST API server
├── pkg/              # Public packages
│   ├── fb2/          # FB2 XML parsing
│   ├── translator/   # Translation engines
│   │   ├── dictionary/
│   │   ├── google/
│   │   └── llm/      # OpenAI, Anthropic, Zhipu, DeepSeek, Ollama
│   ├── converter/    # EPUB/PDF conversion
│   ├── script/       # Cyrillic/Latin conversion
│   ├── api/          # REST API handlers
│   ├── websocket/    # WebSocket hub
│   ├── security/     # Auth, rate limiting
│   └── events/       # Event system
├── internal/         # Private packages
│   ├── config/       # Configuration management
│   ├── cache/        # Translation caching
│   └── metrics/      # Statistics
├── test/
│   ├── unit/
│   ├── integration/
│   ├── e2e/
│   ├── performance/
│   └── stress/
├── api/
│   ├── openapi/      # OpenAPI specification
│   └── examples/     # curl, http, postman
├── Documentation/    # All documentation
└── Legacy/           # Python implementation
```

## Core Components

### 1. Event System

Central event bus for system-wide event distribution.

**Features:**
- Publish-subscribe pattern
- Type-safe events
- Thread-safe operations
- WebSocket integration

**Event Types:**
- `translation_started`
- `translation_progress`
- `translation_completed`
- `translation_error`
- `conversion_started`
- `conversion_progress`
- `conversion_completed`
- `conversion_error`

### 2. FB2 Parser

Handles FictionBook2 XML parsing and manipulation.

**Capabilities:**
- Parse FB2 files
- Maintain XML structure
- Update metadata
- Preserve formatting
- Namespace handling

### 3. Translation Engines

#### Dictionary Translator
- Fast, offline translation
- Built-in Russian-Serbian dictionary
- No API dependencies
- Suitable for basic translations

#### LLM Translators
High-quality, context-aware translation using:

**OpenAI GPT:**
- Models: gpt-4, gpt-3.5-turbo
- Best for complex literature
- High accuracy

**Anthropic Claude:**
- Models: claude-3-sonnet, claude-3-opus
- Excellent context understanding
- Natural language processing

**Zhipu AI (GLM-4):**
- Cutting-edge Chinese AI
- Good multilingual support
- Cost-effective

**DeepSeek:**
- Excellent quality-to-cost ratio
- Fast processing
- Good for bulk translations

**Ollama (Local):**
- Free, offline operation
- Privacy-preserving
- Models: llama3:8b, llama2:13b

### 4. Script Converter

Converts Serbian text between Cyrillic and Latin scripts.

**Features:**
- Bidirectional conversion
- Multi-character mapping (Lj, Nj, Dž)
- Auto-detection
- Preserves punctuation

### 5. WebSocket Hub

Manages WebSocket connections for real-time updates.

**Architecture:**
- Hub-and-spoke pattern
- Client management
- Session filtering
- Automatic cleanup

### 6. Security Layer

#### Authentication
- JWT tokens (HS256)
- API keys
- Configurable TTL

#### Rate Limiting
- Per-IP limiting
- Token bucket algorithm
- Configurable RPS and burst

#### TLS/QUIC
- HTTP/3 support
- TLS 1.3
- Self-signed or CA certificates

### 7. Caching

Thread-safe translation cache with TTL.

**Features:**
- SHA-256 key hashing
- Automatic expiration
- Periodic cleanup
- Statistics tracking

## API Server Architecture

### HTTP/3 (QUIC)

The server supports HTTP/3 over QUIC for:
- Lower latency
- Better multiplexing
- Connection migration
- Improved performance on lossy networks

### Middleware Stack

1. **CORS Middleware** - Cross-origin request handling
2. **Rate Limit Middleware** - Request throttling
3. **Auth Middleware** - JWT/API key validation (optional)
4. **Logging Middleware** - Request logging (Gin default)
5. **Recovery Middleware** - Panic recovery (Gin default)

### Request Flow

```
Client Request
    ↓
CORS Check
    ↓
Rate Limit
    ↓
Authentication (if enabled)
    ↓
Route Handler
    ↓
Business Logic
    ↓
Event Publishing (WebSocket)
    ↓
Response
```

## Data Flow

### Translation Flow

```
1. Client submits translation request
2. API validates request
3. Creates translator instance
4. Generates session ID
5. Publishes start event
6. Processes translation
   ├── Checks cache
   ├── Calls translation engine
   └── Caches result
7. Publishes progress events
8. Returns translated content
9. Publishes completion event
```

### WebSocket Flow

```
1. Client connects to /ws
2. WebSocket upgrade
3. Client registered in hub
4. Events published to event bus
5. Hub filters by session ID
6. Events sent to relevant clients
7. Client disconnects
8. Cleanup
```

## Scalability

### Horizontal Scaling

- Stateless API design
- External cache (Redis) integration ready
- Load balancer support
- Session affinity for WebSocket

### Vertical Scaling

- Concurrent request handling
- Go routines for parallel processing
- Efficient memory management
- HTTP/3 multiplexing

## Security Considerations

### Input Validation

- Request size limits
- File type validation
- XML bomb protection
- SQL injection prevention (if DB added)

### API Security

- HTTPS/HTTP3 only
- JWT token expiration
- API key rotation
- Rate limiting

### Data Privacy

- No persistent storage (by default)
- Secure credential handling
- Environment variable secrets
- Optional encryption at rest

## Performance

### Optimization Strategies

1. **Caching** - Reduce redundant translations
2. **Connection Pooling** - Reuse HTTP connections
3. **Goroutines** - Parallel processing
4. **HTTP/3** - Reduced latency
5. **Binary Encoding** - Efficient serialization

### Benchmarks

Target performance metrics:
- **API Latency**: < 100ms (dictionary), < 2s (LLM)
- **Throughput**: 100+ req/s (dictionary)
- **WebSocket**: 1000+ concurrent connections
- **Memory**: < 100MB baseline

## Monitoring

### Metrics

- Request count
- Response times
- Error rates
- Cache hit ratio
- Active WebSocket connections
- Translation statistics

### Logging

- Structured JSON logging
- Configurable log levels
- Request/response logging
- Error stack traces

## Deployment

### Docker

```bash
docker build -t translator:latest .
docker run -p 8443:8443 translator:latest
```

### Kubernetes

Ready for Kubernetes deployment with:
- Health checks
- Liveness probes
- Readiness probes
- Resource limits
- Auto-scaling

### Cloud Platforms

Compatible with:
- AWS ECS/EKS
- Google Cloud Run/GKE
- Azure Container Instances/AKS
- DigitalOcean App Platform

## Testing Strategy

### Unit Tests

- Package-level tests
- Mock dependencies
- Coverage > 80%

### Integration Tests

- API endpoint tests
- Database integration
- External service mocks

### E2E Tests

- Full workflow testing
- CLI integration
- WebSocket scenarios

### Performance Tests

- Load testing
- Stress testing
- Benchmark comparisons

## Future Enhancements

1. **Database Integration** - Persistent storage
2. **User Management** - Full auth system
3. **Admin Dashboard** - Web UI
4. **Batch Processing** - Queue system
5. **Multi-language Support** - Beyond RU-SR
6. **PDF/EPUB Direct** - Native format support
7. **Metrics Dashboard** - Prometheus/Grafana
8. **gRPC API** - Alternative protocol
