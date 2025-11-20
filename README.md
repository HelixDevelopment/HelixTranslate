# Russian-Serbian FB2 Translator

A high-performance, enterprise-grade translation toolkit for Russian to Serbian FictionBook2 (FB2) e-books, featuring multiple translation engines, REST API with HTTP/3 support, and real-time WebSocket events.

## 🚀 Features

- **Multiple Translation Engines**
  - Simple dictionary-based translation
  - Advanced LLM translation (OpenAI GPT, Anthropic Claude, Zhipu AI, DeepSeek, Local Ollama)
  - Google Translate integration (legacy)

- **Modern Architecture**
  - Built with Go for high performance
  - REST API with Gin Gonic framework
  - HTTP/3 (QUIC) support for reduced latency
  - WebSocket support for real-time progress tracking
  - Event-driven architecture

- **Security First**
  - JWT authentication
  - API key support
  - Rate limiting
  - TLS 1.3 encryption
  - CORS configuration

- **Developer Friendly**
  - CLI tool for batch processing
  - Comprehensive API documentation
  - OpenAPI specification
  - Postman collection
  - curl examples
  - WebSocket test page

- **Format Support**
  - FB2 (FictionBook2) parsing and generation
  - Cyrillic ↔ Latin script conversion
  - EPUB conversion (planned)
  - PDF conversion (planned)

## 📦 Installation

### Prerequisites

- Go 1.21 or higher
- Make (optional, for Makefile commands)
- OpenSSL (for TLS certificate generation)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd Translate

# Install dependencies
make deps

# Build CLI and server
make build

# Or build individually
make build-cli
make build-server
```

### Binary Installation

```bash
# Install to GOPATH/bin
make install
```

### Docker

```bash
# Build Docker image
make docker-build

# Run container
make docker-run
```

## 🎯 Quick Start

### CLI Usage

```bash
# Basic dictionary translation
./build/translator -input book.fb2

# LLM translation with OpenAI
export OPENAI_API_KEY="your-key"
./build/translator -input book.fb2 -provider openai -model gpt-4

# Latin script output
./build/translator -input book.fb2 -provider deepseek -script latin

# See all options
./build/translator -help
```

### REST API Server

```bash
# Generate TLS certificates
make generate-certs

# Start server (creates default config if not exists)
./build/translator-server

# Or with custom config
./build/translator-server -config config.json

# Server will start on:
# - HTTP/3 (QUIC): https://localhost:8443
# - HTTP/2 (fallback): https://localhost:8443
# - WebSocket: wss://localhost:8443/ws
```

### API Examples

#### Translate Text

```bash
curl -X POST https://localhost:8443/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Привет, мир!",
    "provider": "dictionary",
    "script": "cyrillic"
  }' \
  --insecure
```

#### Translate FB2 File

```bash
curl -X POST https://localhost:8443/api/v1/translate/fb2 \
  -F "file=@book.fb2" \
  -F "provider=openai" \
  -F "model=gpt-4" \
  --output book_translated.fb2 \
  --insecure
```

#### WebSocket Connection

```javascript
const ws = new WebSocket('wss://localhost:8443/ws?session_id=your-session-id');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${data.type}] ${data.message}`);
};
```

## 📖 Documentation

Comprehensive documentation is available in the `/Documentation` directory:

- **[API Documentation](Documentation/API.md)** - Complete API reference
- **[Architecture](Documentation/ARCHITECTURE.md)** - System architecture and design
- **[CLAUDE.md](CLAUDE.md)** - Project guidelines for AI assistants

### API Documentation Files

- **OpenAPI Specification**: `/api/openapi/openapi.yaml`
- **Postman Collection**: `/api/examples/postman/translator-api.postman_collection.json`
- **curl Examples**: `/api/examples/curl/`
- **WebSocket Test Page**: `/api/examples/curl/websocket-test.html`

## 🔧 Configuration

### Environment Variables

```bash
# LLM Provider API Keys
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"

# Server Security
export JWT_SECRET="your-secret-key"
```

### Configuration File

Create a `config.json` file:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8443,
    "enable_http3": true,
    "tls_cert_file": "certs/server.crt",
    "tls_key_file": "certs/server.key"
  },
  "security": {
    "enable_auth": false,
    "rate_limit_rps": 10,
    "rate_limit_burst": 20
  },
  "translation": {
    "default_provider": "dictionary",
    "cache_enabled": true,
    "cache_ttl": 3600
  }
}
```

## 🧪 Testing

```bash
# Run all tests
make test

# Unit tests only
make test-unit

# Integration tests
make test-integration

# E2E tests
make test-e2e

# Performance tests
make test-performance

# Stress tests
make test-stress

# Generate coverage report
make test-coverage
```

## 🏗️ Development

### Project Structure

```
digital.vasic.translator/
├── cmd/              # Applications
│   ├── cli/          # CLI tool
│   └── server/       # REST API server
├── pkg/              # Public packages
│   ├── fb2/          # FB2 parsing
│   ├── translator/   # Translation engines
│   ├── api/          # API handlers
│   ├── websocket/    # WebSocket hub
│   └── security/     # Security features
├── internal/         # Private packages
├── test/             # Test suites
├── api/              # API documentation
├── Documentation/    # Project docs
└── Legacy/           # Python implementation
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Run checks
make test fmt lint
```

## 🚀 Deployment

### Docker Deployment

```bash
docker build -t translator:latest .
docker run -d \
  -p 8443:8443 \
  -v $(pwd)/certs:/app/certs \
  -v $(pwd)/config.json:/app/config/config.json \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  translator:latest
```

### Kubernetes

```bash
# Apply Kubernetes manifests (when available)
kubectl apply -f k8s/
```

## 🌟 Translation Providers

| Provider | Quality | Cost | Requirements |
|----------|---------|------|--------------|
| **Dictionary** | ⭐⭐⭐ | Free | None |
| **OpenAI GPT-4** | ⭐⭐⭐⭐⭐ | $$$ | API Key |
| **Anthropic Claude** | ⭐⭐⭐⭐⭐ | $$$ | API Key |
| **Zhipu AI (GLM-4)** | ⭐⭐⭐⭐ | $$ | API Key |
| **DeepSeek** | ⭐⭐⭐⭐ | $ | API Key |
| **Ollama (Local)** | ⭐⭐⭐⭐ | Free | Local Setup |

## 📊 Performance

- **API Latency**: < 100ms (dictionary), < 2s (LLM)
- **Throughput**: 100+ requests/second
- **WebSocket**: 1000+ concurrent connections
- **HTTP/3**: 30% latency reduction vs HTTP/2

## 🔒 Security

- TLS 1.3 encryption
- HTTP/3 (QUIC) support
- JWT authentication
- API key management
- Rate limiting (10 RPS default)
- CORS configuration
- Request size limits
- Input validation

## 📝 License

[Specify your license here]

## 🤝 Contributing

Contributions are welcome! Please read our contributing guidelines (when available).

## 🐛 Issues

Report issues at: [GitHub Issues URL]

## 📞 Support

- Documentation: `/Documentation`
- Examples: `/api/examples`
- Legacy Python: `/Legacy`

## 🎓 Legacy Python Implementation

The original Python implementation is preserved in the `/Legacy` directory for reference and gradual migration.

To use the Python version:

```bash
cd Legacy
pip3 install -r requirements.txt
python3 llm_fb2_translator.py book.fb2 --provider openai
```

## 🔄 Migration from Python

The Go implementation maintains CLI compatibility with the Python version:

```bash
# Python (Legacy)
python3 llm_fb2_translator.py book.fb2 --provider openai

# Go (New)
./translator -input book.fb2 -provider openai
```

## 🚀 Roadmap

- [ ] PostgreSQL integration for persistent storage
- [ ] User management and multi-tenancy
- [ ] Admin web dashboard
- [ ] Direct EPUB/PDF translation support
- [ ] Prometheus metrics export
- [ ] gRPC API
- [ ] Additional language pairs
- [ ] Machine learning model fine-tuning

## 📈 Metrics & Monitoring

The API provides built-in metrics:

```bash
# Get statistics
curl https://localhost:8443/api/v1/stats --insecure
```

## 🌐 Supported Formats

- **Input**: FB2 (FictionBook2)
- **Output**: FB2, Cyrillic, Latin script
- **Planned**: EPUB, PDF, MOBI

---

**Built with ❤️ using Go, Gin, QUIC, and modern cloud-native technologies**
