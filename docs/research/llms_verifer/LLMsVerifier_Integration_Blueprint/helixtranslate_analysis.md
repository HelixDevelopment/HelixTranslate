# HelixTranslate Repository - Comprehensive Analysis Report

## A. PROJECT OVERVIEW

**Project Name**: HelixTranslate (digital.vasic.translator)
**Language**: Go 1.25.2
**Module**: `digital.vasic.translator`
**Version**: 2.3.0 (VERSION file) / 3.0.0 (Makefile)
**Type**: Universal multi-format, multi-language ebook translation system

**Architecture Pattern**: Event-driven microservices-style architecture with:
- Central event bus (pub/sub) for decoupled component communication
- Factory pattern for LLM provider instantiation
- Strategy pattern for format-specific parsers
- Hub-and-spoke pattern for WebSocket connections
- Layered architecture: cmd/ -> pkg/ -> internal/

**Core Capabilities**:
- Multi-format ebook parsing (FB2, EPUB, TXT, HTML, PDF, DOCX)
- Multi-language translation using 8+ LLM providers
- Real-time WebSocket monitoring dashboard
- Distributed translation via SSH workers
- REST API (HTTP/3 + WebSocket) + gRPC interfaces
- Translation caching (PostgreSQL, Redis, SQLite)
- Quality verification with multi-pass polishing
- Preparation phase analysis (characters, terminology, culture)
- Serbian Cyrillic/Latin script conversion
- Markdown workflow (EPUB <-> Markdown)

---

## B. DIRECTORY STRUCTURE (Complete)

```
HelixTranslate/
├── .gitignore                          # Git ignore rules
├── .gitmodules                         # Submodule definitions (Challenges, Containers)
├── .golangci.yml                       # golangci-lint configuration
├── AGENTS.md                           # AI agent guidance (20KB)
├── CLAUDE.md                           # Claude Code guidance (11KB)
├── COMPREHENSIVE_COMPLETION_REPORT.md  # Completion status report
├── COMPREHENSIVE_PROJECT_COMPLETION_PLAN.md # Implementation plan
├── CONSTITUTION.md                     # Team constitution
├── CURRENT_PROGRESS_REPORT.md          # Current progress
├── Dockerfile                          # Multi-stage Docker build
├── DISTRIBUTED_WORK_IMPLEMENTATION_PLAN.md
├── DOCUMENTATION_COMPLETION_PLAN.md
├── Makefile                            # Build automation (v3.0.0)
├── README.md                           # WebSocket monitoring system docs
├── config.json                         # Main application configuration
├── go.mod                              # Go module dependencies
├── go.sum                              # Dependency checksums
│
├── Challenges/                         # SUBMODULE - Challenge system
│   └── (separate repo: vasic-digital/Challenges)
│
├── Containers/                         # SUBMODULE - Container definitions
│   └── (separate repo: vasic-digital/Containers)
│
├── Documentation/                      # ~60 documentation files
│   ├── AGENTS.md, API.md, ARCHITECTURE.md, CLI.md
│   ├── BATCH_TRANSLATION_COMMANDS.md
│   ├── *_IMPLEMENTATION*.md, *_GUIDE*.md
│   └── Various status/completion reports
│
├── cmd/                                # 17 entry point binaries
│   ├── api-server/                     # REST API server (:8080)
│   ├── challenge-runner/               # Challenge execution runner
│   ├── cli/                            # Basic CLI tool
│   ├── deployment/                     # Deployment management
│   ├── ebook-translator/               # Specialized ebook translator
│   ├── grpc-server/                    # gRPC service (:50051)
│   ├── markdown-translator/            # Markdown workflow tool
│   ├── monitor-server/                 # WebSocket monitoring (:8090)
│   ├── preparation-translator/         # Pre-translation analysis
│   ├── server/                         # Legacy REST API server
│   ├── ssh-translation/                # SSH translation (older)
│   ├── translate-ssh/                  # SSH worker binary
│   ├── translator/                     # Legacy translator CLI
│   └── unified-translator/            # PRIMARY unified CLI
│
├── internal/                           # Private packages
│   ├── cache/                          # Caching layer
│   ├── config/                         # Configuration loading
│   ├── scripts/                        # 50+ shell scripts
│   └── working/                        # Configs, test data, logs
│
├── pkg/                                # Public packages (~30 packages)
│   ├── api/                            # REST handlers, routing
│   ├── batch/                          # Batch processing
│   ├── coordination/                   # Multi-LLM coordination
│   ├── deployment/                     # Docker/SSH deployment
│   ├── distributed/                    # SSH worker system
│   ├── ebook/                          # Universal ebook parsing
│   ├── events/                         # Event bus system
│   ├── fb2/                            # FB2-specific XML handling
│   ├── format/                         # Format detection
│   ├── grpc/                           # gRPC service + proto
│   ├── hardware/                       # Hardware detection
│   ├── hash/                           # Codebase hashing
│   ├── language/                       # Language detection
│   ├── logger/                         # Structured logging
│   ├── markdown/                       # EPUB<->Markdown workflow
│   ├── models/                         # Model registry + downloader
│   ├── preparation/                    # Pre-translation analysis
│   ├── progress/                       # Progress tracking
│   ├── report/                         # Report generation
│   ├── script/                         # Cyrillic/Latin conversion
│   ├── security/                       # JWT auth, rate limiting
│   ├── sshworker/                      # SSH worker management
│   ├── storage/                        # DB abstraction (PG/Redis/SQLite)
│   ├── translator/                     # Translation engine core
│   │   └── llm/                        # 8 LLM provider implementations
│   ├── verification/                   # Quality verification
│   ├── version/                        # Version management
│   └── websocket/                      # WebSocket hub
│
├── test/                               # Cross-cutting test suites
│   ├── distributed/                    # Distributed system tests
│   ├── e2e/                            # End-to-end tests
│   ├── fixtures/                       # Test data
│   ├── integration/                    # Integration tests
│   ├── mocks/                          # Shared mocks
│   ├── performance/                    # Performance tests
│   ├── security/                       # Security tests
│   ├── stress/                         # Stress tests
│   ├── translator/                     # Translator tests
│   ├── unit/                           # Unit tests
│   └── utils/                          # Test helpers
│
└── tests/                              # Additional WebSocket tests
    └── websocket_monitoring_test.go
```

---

## C. KEY SOURCE FILES - Detailed Analysis

### C1. Entry Points

#### `cmd/unified-translator/main.go` (PRIMARY CLI)
- **Purpose**: Main unified CLI with full provider support
- **Key Types**: `UnifiedConfig`, `TranslationSession`, `GeneratedFile`, `TranslationStep`
- **Flags**: `-input/-i`, `-output/-o`, `-source-lang`, `-target-lang`, `-script`, `-provider`, `-model`, `-api-key`, `-base-url`, `-temperature`, `-max-tokens`, `-timeout`, SSH config, llama.cpp config, `-workers`, `-chunk-size`, `-concurrency`, `-verify`, `-monitoring`, `-monitoring-port`
- **Flow**: parseFlags -> initLogger -> initEventBus -> createSession -> startMonitoring (optional) -> executeTranslation -> generateReport
- **Providers supported**: openai, anthropic, zhipu, deepseek, qwen, gemini, ollama, llamacpp, ssh

#### `cmd/api-server/main.go` (REST API Server)
- **Purpose**: REST API server with gRPC backend connection
- **Key Types**: `APIServer`, `APIConfig`, `TranslationRequest`, `ProviderRequestConfig`, `WebSocketMessage`, `APIResponse`
- **Default ports**: gRPC :50051, HTTP :8080
- **Features**: WebSocket hub, gRPC client, JWT auth

#### `cmd/grpc-server/main.go` (gRPC Server)
- **Purpose**: gRPC translation service
- **Key Types**: `ServerConfig`
- **Default**: :50051 with reflection enabled
- **Flags**: `-address`, `-port`, `-max-connections`, `-reflection`, `-metrics`, `-log-level`

### C2. LLM Provider Layer

#### `pkg/translator/llm/llm.go` (CORE LLM INTERFACE)
- **LLMClient interface**: `Translate(ctx, text, prompt) (string, error)` + `GetProviderName() string`
- **Factory**: `NewLLMTranslator(config)` -> switches on provider string
- **ValidModels map**: Maps each Provider to valid model names
- **Providers enum**: `ProviderOpenAI`, `ProviderAnthropic`, `ProviderZhipu`, `ProviderDeepSeek`, `ProviderQwen`, `ProviderGemini`, `ProviderOllama`, `ProviderLlamaCpp`, `ProviderMock`
- **Auto-retry**: `translateWithRetry()` splits text on size errors
- **Text splitting**: Splits at sentence/paragraph boundaries, max 20KB chunks
- **Size error detection**: Checks for "max_tokens", "token limit", "too large", etc.
- **Caching**: In-memory cache keyed by `text:context`

#### `pkg/translator/llm/openai.go`
- **Type**: `OpenAIClient` with `OpenAIRequest/Response` structs
- **Timeout**: 600 seconds (10 minutes for large book sections)
- **Max tokens**: 8192 (DeepSeek-compatible)
- **Default model**: gpt-4
- **Temperature**: 0.0-2.0 range validated
- **API**: POST /chat/completions

#### `pkg/translator/llm/anthropic.go`
- **Type**: `AnthropicClient`
- **API**: POST /messages with `x-api-key` and `anthropic-version: 2023-06-01`
- **Timeout**: 600 seconds
- **Max tokens**: 4096
- **Models**: claude-3-opus, claude-3-sonnet, claude-3-haiku

#### `pkg/translator/llm/deepseek.go`
- **Type**: `DeepSeekClient` (embeds `OpenAIClient`)
- **BaseURL**: https://api.deepseek.com/v1
- **Models**: deepseek-chat, deepseek-coder
- **Uses OpenAI-compatible API format**

#### `pkg/translator/llm/llamacpp.go` & `llamacpp_provider.go`
- **Local execution**: Spawns llama.cpp binary process
- **Configuration**: `-llama-binary`, `-llama-model`, `-context-size`
- **Models**: llama2, mistral, vicuna (custom models allowed with warning)

#### `pkg/translator/llm/ollama.go`
- **Local Ollama API**: http://localhost:11434
- **Models**: llama2, codellama, mistral, vicuna

#### `pkg/translator/llm/qwen.go`, `zhipu.go`, `gemini.go`
- Cloud API clients for respective providers

#### `pkg/translator/llm/mock.go`
- **MockLLMClient**: For testing, returns predictable responses

### C3. Configuration System

#### `internal/config/config.go`
- **Config struct** with sections:
  - `ServerConfig`: Host, Port, EnableHTTP3, TLS, Timeouts, MaxUploadSize
  - `SecurityConfig`: EnableAuth, JWTSecret, APIKeyHeader, RateLimitRPS/Burst, CORSOrigins
  - `TranslationConfig`: DefaultProvider, DefaultModel, CacheEnabled, CacheTTL, MaxConcurrent, Providers map
  - `ProviderConfig`: APIKey, BaseURL, Model, Options
  - `DistributedConfig`: Enabled, Workers map, SSHTimeout, MaxRetries, HealthCheckInterval
  - `WorkerConfig`: Name, Host, Port, User, KeyFile, Password, MaxCapacity, Tags
  - `LoggingConfig`: Level, Format, OutputFile
  - `PreparationConfig`: Enabled, PassCount, Providers, Analysis flags, DetailLevel

- **Environment variable loading**:
  - `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `DEEPSEEK_API_KEY`, `ZHIPU_API_KEY`
  - `QWEN_API_KEY`, `GEMINI_API_KEY`
  - `SSH_WORKER_{HOST,USER,PASSWORD,PORT,REMOTE_DIR}`
  - `MONITOR_SERVER_PORT`, `LOG_LEVEL`

- **Default config**: Port 8443, HTTP3 enabled, auth enabled, 10 RPS rate limit

### C4. Translation Engine

#### `pkg/translator/translator.go`
- **Translator interface**: `Translate()`, `TranslateWithProgress()`, `GetStats()`, `GetName()`
- **TranslationResult**: OriginalText, TranslatedText, Provider, Cached, Error
- **TranslationStats**: Total, Translated, Cached, Errors
- **BaseTranslator**: Config, stats, in-memory cache
- **EmitProgress/EmitError**: Event bus helpers

#### `pkg/translator/universal.go`
- **UniversalTranslator**: Orchestrates full book translation
- **Flow**: Detect language -> Translate metadata -> Translate chapters (with progress events)
- **Per-chapter progress**: Events with chapter number, progress percentage

### C5. Ebook Processing

#### `pkg/ebook/parser.go`
- **Book struct**: Metadata + Chapters[] + Format + Language
- **Metadata**: Title, Authors, Description, Publisher, Language, ISBN, Date, Cover
- **Chapter**: Title + Sections[]
- **Section**: Title, Content, Subsections (recursive)
- **UniversalParser**: Format auto-detection -> delegates to format-specific parser

#### `pkg/format/detector.go`
- **Formats**: FB2, EPUB, PDF, MOBI, AZW, AZW3, TXT, HTML, DOCX, RTF
- **Detection**: Magic bytes + file extension + content-based disambiguation
- **EPUB/DOCX disambiguation**: ZIP internal structure inspection

### C6. Events & WebSocket

#### `pkg/events/events.go` (assumed structure)
- **EventBus**: Thread-safe pub/sub
- **Event types**: TranslationStarted, Progress, Completed, Error; ConversionStarted, Progress, Completed, Error
- **Event**: Type, Message, Data map, SessionID, Timestamp

#### `pkg/websocket/hub.go`
- **Hub**: Manages WebSocket clients, subscribes to event bus
- **Client**: ID, SessionID, Conn, Send channel
- **Features**: Session filtering, automatic cleanup, broadcast
- **Integration**: All events auto-forwarded to dashboard clients

### C7. Distributed System

#### `pkg/distributed/coordinator.go`
- **DistributedCoordinator**: Manages remote LLM instances across SSH workers
- **RemoteLLMInstance**: ID, WorkerID, Provider, Model, Priority, Available
- **Discovery**: `DiscoverRemoteInstances()` queries paired workers
- **Components**: SSHPool, PairingManager, FallbackManager, VersionManager

#### `pkg/distributed/ssh_pool.go`
- **SSHPool**: Pooled SSH connections with cleanup
- **SSHConnection**: Wraps `*ssh.Client` with metadata
- **WorkerConfig**: SSH params, tags, capacity
- **Features**: Auto-cleanup idle connections (30 min), retry logic

### C8. gRPC Layer

#### `pkg/grpc/translator.proto`
- **Service**: `TranslationService` with 7 RPCs
  - `StartTranslation`, `GetTranslationStatus`, `ListTranslations`
  - `CancelTranslation`, `StreamTranslationProgress`
  - `GetProviders`, `SubscribeEvents`
- **Messages**: TranslationRequest, ProviderConfig, TranslationOptions
  - TranslationResponse, TranslationStatusResponse
  - TranslationProgressEvent, GeneratedFile, TranslationStep
  - ProvidersResponse, ProviderInfo, ProviderStatus, SystemEvent

### C9. API Handlers

#### `pkg/api/handler.go`
- **Handler struct**: Config, EventBus, Cache, AuthService, WSHub, DistributedManager
- **Routes**:
  - `GET /health`, `/`
  - `GET /ws` (WebSocket)
  - `POST /api/v1/translate` (text translation)
  - `POST /api/v1/translate/fb2` (FB2 translation)
  - `POST /api/v1/translate/batch` (batch translation)
  - `POST /api/v1/translate/ebook` (ebook translation)
  - `POST /api/v1/convert/script` (script conversion)
  - `GET /api/v1/status/:session_id`
  - `GET /api/v1/version`, `/providers`, `/stats`, `/languages`
  - `POST /api/v1/translate/validate`
  - `POST /api/v1/translate/cancel/:session_id`
  - `POST /api/v1/preparation/analyze`
  - `GET /api/v1/preparation/result/:session_id`

---

## D. CONFIGURATION SYSTEM

### Configuration Files Hierarchy
1. **Main config**: `config.json` (project root)
2. **Example configs**: `internal/working/config_*.json`
3. **Provider configs**: `config_anthropic.json`, `config_deepseek.json`, etc.
4. **Worker configs**: `config.worker.json`, `config.worker.llamacpp.json`
5. **Distributed configs**: `config.distributed.json`

### Configuration Sections
```json
{
  "server": { "host", "port", "enable_http3", "tls_cert_file", "tls_key_file", "timeouts", "max_upload_size" },
  "security": { "enable_auth", "jwt_secret", "api_key_header", "rate_limit_rps", "rate_limit_burst", "cors_origins" },
  "translation": { "default_provider", "default_model", "cache_enabled", "cache_ttl", "max_concurrent", "providers": {} },
  "preparation": { "enabled", "pass_count", "providers", "analyze_*" flags, "detail_level" },
  "distributed": { "enabled", "workers": {}, "ssh_timeout", "ssh_max_retries", "health_check_interval" },
  "logging": { "level", "format", "output_file" }
}
```

### Environment Variables (API Keys ONLY from env)
- **OpenAI**: `OPENAI_API_KEY`
- **Anthropic**: `ANTHROPIC_API_KEY`
- **DeepSeek**: `DEEPSEEK_API_KEY`
- **Zhipu**: `ZHIPU_API_KEY`
- **Qwen**: `QWEN_API_KEY`
- **Gemini**: `GEMINI_API_KEY`
- **SSH Workers**: `SSH_WORKER_HOST`, `SSH_WORKER_USER`, `SSH_WORKER_PASSWORD`, `SSH_WORKER_PORT`, `SSH_WORKER_REMOTE_DIR`
- **Server**: `MONITOR_SERVER_PORT`, `LOG_LEVEL`

### Security: API keys NEVER in config files (excluded by .gitignore)
- `.env*`, `config_with_keys.json`, `api_keys.json`, `secrets.*`, `**/qwen_credentials.json`

---

## E. MODEL/PROVIDER MANAGEMENT

### LLM Provider Factory
```
NewLLMTranslator(config) -> LLMTranslator
  -> validates provider string
  -> validates model against ValidModels[provider]
  -> creates provider-specific client via switch
  -> returns LLMTranslator with embedded LLMClient
```

### Supported Providers (9 total)
| Provider | Type | API Style | Models |
|----------|------|-----------|--------|
| OpenAI | Cloud | Native | gpt-3.5-turbo, gpt-4, gpt-4-turbo, gpt-4o |
| Anthropic | Cloud | Native | claude-3-opus, claude-3-sonnet, claude-3-haiku |
| DeepSeek | Cloud | OpenAI-compatible | deepseek-chat, deepseek-coder |
| Zhipu | Cloud | Native | glm-4, glm-3-turbo |
| Qwen | Cloud | Native | qwen-max, qwen-plus, qwen-turbo |
| Gemini | Cloud | Native | gemini-pro, gemini-pro-vision |
| Ollama | Local | Ollama API | llama2, codellama, mistral, vicuna |
| LlamaCpp | Local | Process spawn | llama2, mistral, vicuna |
| Mock | Test | N/A | mock |

### Local Model Registry (`pkg/models/registry.go`)
Pre-configured GGUF models for local execution:
- **hunyuan-mt-7b** (Q4/Q8): Translation-specialized, 33 languages, Apache-2.0
- **aya-23-8b** (Q4): 23 languages, strong multilingual
- **qwen2.5-7b-instruct** (Q4): 32K context, Russian/Serbian support
- **mistral-7b-instruct** (Q4): General-purpose + translation
- **qwen2.5-14b-instruct** (Q4): Higher quality for 16GB+ RAM systems

Each model has: Parameters, RAM requirements, QuantType, SourceURL, Languages, Quality rating

---

## F. EXISTING TEST STRUCTURE

### Test Organization
**Two locations for tests**:
1. **Alongside source**: `*_test.go` in each `pkg/*` directory (standard Go pattern)
2. **Cross-cutting suites**: `test/` directory

### `test/` Directory Structure
```
test/
├── unit/              # 8 test files (batch, coordination, ebook, fb2, format, language, qwen, script, verification)
├── integration/       # 4 test files (batch_api, cross_package, debug, ssh_translation)
├── e2e/               # translation_quality_e2e_test.go
├── performance/       # translation_performance_test.go
├── stress/            # translation_stress_test.go (.gitkeep)
├── security/          # 3 test files (authentication, input_validation*2)
├── distributed/       # 6 test files (fallback, integration, manager, performance, security, stress)
├── translator/        # LLM provider tests
├── mocks/             # provider.go (shared mocks)
├── fixtures/          # Test ebooks, configs, expected translations
└── utils/             # helpers.go, ports.go, ssh_test_server.go, test_infrastructure.go
```

### Test Statistics (from COMPREHENSIVE_COMPLETION_REPORT.md)
- **Current coverage**: 43.6% overall
- **Critical gaps**: pkg/api (32.8%), pkg/distributed (45.2%), version_manager (26.2%)
- **61 disabled tests** requiring attention
- **Build tags**: `//go:build integration`, `//go:build e2e`

### Test Naming Conventions
- Table-driven tests preferred
- Naming: `TestFunctionName_Scenario`
- Coverage artifacts: `coverage*.out`, `coverage.html`

---

## G. DOCUMENTATION ANALYSIS

### Core Documentation Files (~60 files in Documentation/)

| File | Purpose | Size |
|------|---------|------|
| `CLAUDE.md` | Claude Code guidance - authoritative for Go codebase | 11KB |
| `AGENTS.md` | AI agent project overview - comprehensive reference | 20KB |
| `ARCHITECTURE.md` | System architecture, components, request flows | 8KB |
| `API.md` | REST API documentation | 15KB |
| `API_COMPREHENSIVE_GUIDE.md` | Full API guide with examples | 17KB |
| `CLI.md` | CLI usage documentation | 6KB |
| `CONFIGURATION_REFERENCE.md` | All config options documented | - |
| `BATCH_TRANSLATION_COMMANDS.md` | Batch translation recipes | 9KB |
| `AUTOMATED_DEPLOYMENT.md` | Deployment automation | 16KB |
| `SECURITY_HARDENING.md` | Security best practices | - |
| `TESTING_GUIDE.md` | Test writing guidance | - |
| Various `*_GUIDE.md` | Domain-specific guides | - |
| Various `*_REPORT.md` | Status/completion reports | - |

### CLAUDE.md Key Points
- Module: `digital.vasic.translator`, Go 1.25.2
- VERSION file (2.3.0) is authoritative over Makefile (3.0.0)
- All commands via Makefile
- Entry points in `cmd/` - unified-translator is primary
- API keys from environment variables ONLY
- Distributed changes must be version-compatible
- **Definition of Done**: Real system run with pasted terminal output required

### AGENTS.md Key Points
- Technology stack fully documented
- Build/test commands documented
- Complete directory structure
- Code conventions (error wrapping, `any` over `interface{}`, 140 char line limit)

---

## H. SUBMODULES

### 1. Challenges (`./Challenges`)
- **Repo**: `git@github.com:vasic-digital/Challenges.git`
- **Commit**: `3937f06e9defbc3f466e9d859cf0e34079a5fc6a`
- **Purpose**: Challenge/test system - likely contains translation challenges, test suites, benchmarking tasks
- **Own submodules**: References `Containers` submodule
- **Files**: AGENTS.md (30KB), ARCHITECTURE.md, CLAUDE.md (27KB), CONSTITUTION.md, Makefile
- **Likely role**: Contains evaluation challenges for measuring translation quality

### 2. Containers (`./Containers`)
- **Repo**: `git@github.com:vasic-digital/Containers.git`
- **Commit**: `f572d2615307c696d53e56acc9fdf93f2bb6120e`
- **Purpose**: Container definitions - Dockerfiles, compose files, deployment configurations
- **Used by**: Both main project and Challenges submodule

---

## I. DEPENDENCIES

### Direct Dependencies (from go.mod)
```
github.com/gin-gonic/gin v1.11.0           # Web framework
github.com/golang-jwt/jwt/v5 v5.3.0        # JWT authentication
github.com/google/uuid v1.6.0              # UUID generation
github.com/gorilla/websocket v1.5.3        # WebSocket support
github.com/lib/pq v1.10.9                  # PostgreSQL driver
github.com/mattn/go-sqlite3 v1.14.24       # SQLite driver
github.com/quic-go/quic-go v0.56.0         # HTTP3/QUIC support
github.com/redis/go-redis/v9 v9.7.0        # Redis client
github.com/spf13/cobra v1.10.1             # CLI framework
github.com/stretchr/testify v1.11.1        # Testing framework
github.com/unidoc/unioffice v1.39.0        # DOCX processing
github.com/unidoc/unipdf/v3 v3.69.0        # PDF processing
golang.org/x/crypto v0.48.0                # TLS, SSH
golang.org/x/net v0.49.0                   # Extended networking
golang.org/x/text v0.34.0                  # Text processing
golang.org/x/time v0.14.0                  # Rate limiting
google.golang.org/grpc v1.77.0             # gRPC framework
google.golang.org/protobuf v1.36.10        # Protocol Buffers
gopkg.in/yaml.v3 v3.0.1                    # YAML parsing
```

### Local Modules (replace directives)
- `digital.vasic.challenges => ./Challenges`
- `digital.vasic.containers => ./Containers`

---

## J. API SURFACE

### REST API Endpoints (`pkg/api/handler.go`)

**Health & Info**:
- `GET /health` - Health check
- `GET /` - API info
- `GET /api/v1/version` - Version info
- `GET /api/v1/stats` - Statistics

**Translation**:
- `POST /api/v1/translate` - Text translation
- `POST /api/v1/translate/fb2` - FB2 file translation
- `POST /api/v1/translate/ebook` - Ebook translation
- `POST /api/v1/translate/batch` - Batch translation
- `POST /api/v1/translate/validate` - Validate request
- `POST /api/v1/translate/cancel/:session_id` - Cancel translation

**Preparation**:
- `POST /api/v1/preparation/analyze` - Content analysis
- `GET /api/v1/preparation/result/:session_id` - Get analysis result

**Utility**:
- `POST /api/v1/convert/script` - Cyrillic/Latin conversion
- `GET /api/v1/status/:session_id` - Translation status
- `GET /api/v1/providers` - List providers/models
- `GET /api/v1/languages` - List languages

**WebSocket**:
- `GET /ws` - Real-time event streaming

### gRPC Services (`pkg/grpc/translator.proto`)
- `TranslationService.StartTranslation`
- `TranslationService.GetTranslationStatus`
- `TranslationService.ListTranslations`
- `TranslationService.CancelTranslation`
- `TranslationService.StreamTranslationProgress` (server streaming)
- `TranslationService.GetProviders`
- `TranslationService.SubscribeEvents` (server streaming)

### CLI Commands (`cmd/unified-translator`)
Full flag-based CLI with:
- Input/output file specification
- Provider selection and configuration
- SSH worker parameters
- llama.cpp local parameters
- Monitoring toggle
- Concurrency/worker tuning

---

## K. GAPS/ISSUES

### Critical (from completion reports)
1. **Test Coverage Crisis**: 43.6% overall, need 100%
   - pkg/api: 32.8%
   - pkg/distributed: 45.2%
   - version_manager: 26.2%
2. **61 Disabled Tests**: Security, integration, performance suites
3. **Broken Test Infrastructure**: Hardcoded ports, race conditions, missing mocks
4. **SSH Worker Tests**: Non-existent test servers, incomplete auth mocking

### Incomplete Implementations
1. **Distributed System Security**: HTTP3/QUIC pairing not implemented, SSH key management unclear
2. **Version Management**: Rollback incomplete, update verification missing
3. **Performance Management**: Dynamic scaling not implemented
4. **API Documentation**: OpenAPI spec incomplete, WebSocket events undocumented

### Architecture Debt
1. **Multiple entry points**: Many cmd/ binaries with overlapping functionality
2. **Demo files at root**: Various `demo-*.go` and prebuilt binaries cluttering repo
3. **Coverage files committed**: `coverage*.out`, `coverage.html` in git
4. **Internal working directory**: Contains test data, logs, configs mixed together

### Documentation Gaps
1. Placeholder URLs in website content
2. Video course content missing
3. Advanced configuration examples incomplete
4. Distributed setup guide incomplete

---

## L. INTEGRATION POINTS - Where LLMsVerifier Would Plug In

### L1. LLM Provider Layer Integration (RECOMMENDED)
**Location**: `pkg/translator/llm/`
**Approach**: Add a new provider `llmsverifier` or create a verification wrapper

```
New Provider: pkg/translator/llm/llmsverifier.go
  -> Implements LLMClient interface
  -> Wraps any existing provider
  -> Sends translations to LLMsVerifier for quality check
  -> Returns verified/improved translation
```

**Integration Points**:
- `llm.go`: Add `ProviderLLMsVerifier` to Provider enum
- `ValidModels`: Add verifier-supported models
- Factory switch: Add case for new provider
- Config: Add `LLMsVerifier` section to `TranslationConfig`

### L2. Verification Package Integration
**Location**: `pkg/verification/`
**Existing components**: `verifier.go`, `multipass.go`, `polisher.go`, `reporter.go`
**Approach**: Add LLMsVerifier as a verification backend

```
pkg/verification/llmsverifier_backend.go
  -> Implements verification backend interface
  -> Integrates with existing multi-pass system
  -> Can be used as additional verification pass
```

### L3. Event-Driven Integration (Loosely Coupled)
**Location**: `pkg/events/events.go`
**Approach**: Subscribe to translation completion events, trigger verification

```
EventBus subscriber:
  On EventTranslationCompleted ->
    Extract translated text
    Send to LLMsVerifier API
    Emit EventTranslationVerified with results
```

### L4. API Handler Integration
**Location**: `pkg/api/handler.go`
**Approach**: Add verification endpoints

```
New endpoints:
  POST /api/v1/verify - Submit translation for verification
  GET /api/v1/verify/:id - Get verification results
  POST /api/v1/translate-with-verification - Combined endpoint
```

### L5. Configuration Integration
**Location**: `internal/config/config.go`
**Approach**: Add LLMsVerifier configuration section

```json
{
  "llmsverifier": {
    "enabled": true,
    "api_key": "${LLMSVERIFIER_API_KEY}",
    "base_url": "https://api.llmsverifier.com",
    "model": "verifier-v1",
    "verification_level": "strict",
    "auto_correct": true
  }
}
```

### L6. gRPC Integration
**Location**: `pkg/grpc/translator.proto`
**Approach**: Add verification RPCs

```protobuf
rpc VerifyTranslation(VerifyRequest) returns (VerifyResponse);
rpc StreamVerificationProgress(StreamRequest) returns (stream VerifyEvent);
```

### Recommended Integration Strategy
**Phase 1**: Add LLMsVerifier as a new LLM provider in `pkg/translator/llm/` - minimal changes, uses existing factory pattern
**Phase 2**: Integrate with `pkg/verification/` for multi-pass quality checks
**Phase 3**: Add REST/gRPC API endpoints for standalone verification
**Phase 4**: Add configuration and event-driven integration

---

## Summary Statistics
- **Total files in repo**: ~936
- **Go source packages**: ~30 in pkg/ + internal/
- **Entry point binaries**: 17 in cmd/
- **LLM providers**: 9 (8 real + 1 mock)
- **Test files**: 100+ across test/ and pkg/*_test.go
- **Documentation files**: 60+ in Documentation/ + root
- **Supported formats**: 10 (FB2, EPUB, TXT, HTML, PDF, DOCX, MOBI, AZW, AZW3, RTF)
- **Submodules**: 2 (Challenges, Containers)
- **Direct Go dependencies**: 19 + 2 local replace
