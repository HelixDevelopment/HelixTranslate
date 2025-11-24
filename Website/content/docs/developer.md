---
title: "Complete Developer Guide"
description: "Comprehensive guide for developers working with Universal Ebook Translator"
date: "2024-01-15"
weight: 20
---

# Complete Developer Guide

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Development Environment Setup](#development-environment-setup)
3. [Code Organization](#code-organization)
4. [API Development](#api-development)
5. [Plugin System](#plugin-system)
6. [Testing](#testing)
7. [Contributing](#contributing)
8. [Debugging](#debugging)
9. [Performance Optimization](#performance-optimization)
10. [Security](#security)

## Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client Layer                           │
├─────────────┬─────────────┬─────────────┬───────────────┤
│    CLI      │  Web UI     │   API       │   SDK         │
└─────────────┴─────────────┴─────────────┴───────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                 Translation Core                           │
├─────────────┬─────────────┬─────────────┬───────────────┤
│   Parser    │ Translator  │ Formatter   │   Validator   │
└─────────────┴─────────────┴─────────────┴───────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│               Infrastructure Layer                        │
├─────────────┬─────────────┬─────────────┬───────────────┤
│   Storage   │  Queueing   │  Monitoring  │   Auth        │
└─────────────┴─────────────┴─────────────┴───────────────┘
```

### Core Components

#### 1. Translation Pipeline

```go
// pkg/translator/pipeline.go
type TranslationPipeline struct {
    Parser      Parser
    Translator  Translator
    Formatter   Formatter
    Validator   Validator
    QualityAss  QualityAssessor
}

func (tp *TranslationPipeline) Process(ctx context.Context, job TranslationJob) error {
    // 1. Parse input document
    doc, err := tp.Parser.Parse(job.InputFile, job.Format)
    if err != nil {
        return fmt.Errorf("parsing failed: %w", err)
    }
    
    // 2. Translate content
    translated, err := tp.Translator.Translate(ctx, doc, job.Options)
    if err != nil {
        return fmt.Errorf("translation failed: %w", err)
    }
    
    // 3. Format output
    output, err := tp.Formatter.Format(translated, job.OutputFormat)
    if err != nil {
        return fmt.Errorf("formatting failed: %w", err)
    }
    
    // 4. Validate result
    if err := tp.Validator.Validate(output); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    return nil
}
```

#### 2. LLM Provider Interface

```go
// pkg/translator/llm/interface.go
type LLMProvider interface {
    Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error)
    GetModel() string
    GetCapabilities() ProviderCapabilities
    EstimateTokens(text string) int
    SupportsLanguage(source, target string) bool
}

type TranslationRequest struct {
    Text          string            `json:"text"`
    SourceLang    string            `json:"source_lang"`
    TargetLang    string            `json:"target_lang"`
    Context       map[string]string `json:"context,omitempty"`
    MaxTokens     int               `json:"max_tokens,omitempty"`
    Temperature   float64           `json:"temperature,omitempty"`
}

type TranslationResponse struct {
    TranslatedText string  `json:"translated_text"`
    TokensUsed    int     `json:"tokens_used"`
    Quality       float64 `json:"quality"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
```

#### 3. File Format System

```go
// pkg/format/interface.go
type FormatHandler interface {
    Detect(file []byte) (Format, error)
    Parse(file []byte) (*Document, error)
    Generate(doc *Document) ([]byte, error)
    GetCapabilities() FormatCapabilities
}

type Document struct {
    Metadata Metadata    `json:"metadata"`
    Content  []Content   `json:"content"`
    Assets   []Asset     `json:"assets,omitempty"`
}

type Content struct {
    Type    string      `json:"type"`    // paragraph, heading, image, etc.
    Content interface{} `json:"content"`
    Style   CSSStyle   `json:"style,omitempty"`
}
```

## Development Environment Setup

### Prerequisites

```bash
# Go 1.25.2+
go version

# Git
git --version

# Docker (optional)
docker --version

# Database (PostgreSQL for development)
psql --version

# Redis (optional)
redis-cli --version
```

### Development Setup

```bash
# Clone repository
git clone https://github.com/digital-vasic/translator.git
cd translator

# Install dependencies
make deps

# Setup development environment
make dev-setup

# Run tests to verify setup
make test-unit

# Start development server
make dev-server
```

### Makefile Commands

```makefile
# Development
deps:
	go mod download
	go mod tidy

dev-setup:
	pre-commit install
	go generate ./...

dev-server:
	go run ./cmd/server --config config.dev.json

test-unit:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -race -tags=integration ./test/integration/...

test-e2e:
	go test -v -tags=e2e ./test/e2e/...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .

build:
	go build -o build/translator ./cmd/cli
	go build -o build/translator-server ./cmd/server
```

### IDE Configuration

#### VS Code

`.vscode/settings.json`:
```json
{
    "go.useLanguageServer": true,
    "go.testFlags": ["-v", "-race"],
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "gutter",
        "coveredHighlightColor": "rgba(64,128,64,0.5)",
        "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
    },
    "files.exclude": {
        "**/coverage.out": true,
        "**/coverage.html": true
    }
}
```

#### GoLand

- Enable Go modules
- Configure file watchers
- Set up test configurations
- Enable code coverage visualization

## Code Organization

### Directory Structure

```
translator/
├── cmd/                    # Main applications
│   ├── cli/               # Command-line interface
│   ├── server/             # HTTP API server
│   └── worker/             # Distributed worker
├── pkg/                    # Library packages
│   ├── api/                # HTTP handlers
│   ├── translator/          # Translation logic
│   │   ├── llm/           # LLM providers
│   │   └── cache/          # Translation cache
│   ├── format/             # File format handlers
│   ├── storage/            # Database/storage
│   ├── distributed/        # Distributed processing
│   ├── security/           # Authentication/authorization
│   └── monitoring/         # Metrics/logging
├── internal/               # Internal packages
├── test/                   # Test files
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── docs/                   # Documentation
├── scripts/                 # Utility scripts
└── configs/                # Configuration files
```

### Package Naming Conventions

```bash
# Format: domain/subdomain
pkg/translator/llm/openai.go
pkg/format/epub/parser.go
pkg/storage/postgres/connection.go

# Internal packages: internal/domain
internal/config/loader.go
internal/database/migrations/
```

### Design Patterns Used

#### 1. Repository Pattern

```go
// pkg/storage/repository.go
type TranslationRepository interface {
    Save(ctx context.Context, translation *Translation) error
    FindByID(ctx context.Context, id string) (*Translation, error)
    FindByUser(ctx context.Context, userID string) ([]*Translation, error)
}

type PostgresTranslationRepository struct {
    db *sql.DB
}

func NewPostgresTranslationRepository(db *sql.DB) TranslationRepository {
    return &PostgresTranslationRepository{db: db}
}
```

#### 2. Factory Pattern

```go
// pkg/translator/llm/factory.go
type ProviderFactory interface {
    CreateProvider(config ProviderConfig) (LLMProvider, error)
}

type OpenAIFactory struct{}
func (f *OpenAIFactory) CreateProvider(config ProviderConfig) (LLMProvider, error) {
    return NewOpenAIProvider(config.APIKey, config.Model)
}

func GetProviderFactory(provider string) (ProviderFactory, error) {
    switch provider {
    case "openai":
        return &OpenAIFactory{}, nil
    case "anthropic":
        return &AnthropicFactory{}, nil
    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}
```

#### 3. Observer Pattern

```go
// pkg/events/events.go
type EventBus interface {
    Subscribe(eventType string, handler EventHandler)
    Publish(event Event)
}

type EventHandler func(event Event)

type Event struct {
    Type      string      `json:"type"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}

// Usage
eventBus.Subscribe("translation.started", func(event Event) {
    // Handle translation start
});
```

## API Development

### REST API Architecture

```go
// pkg/api/handlers.go
type Server struct {
    router         *gin.Engine
    translationSvc TranslationService
    authService   AuthService
    eventBus      events.EventBus
}

func NewServer(
    translationSvc TranslationService,
    authService AuthService,
    eventBus events.EventBus,
) *Server {
    s := &Server{
        translationSvc: translationSvc,
        authService:   authService,
        eventBus:      eventBus,
    }
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    v1 := s.router.Group("/api/v1")
    
    // Translation endpoints
    translations := v1.Group("/translations")
    {
        translations.POST("/", s.CreateTranslation)
        translations.GET("/:id", s.GetTranslation)
        translations.DELETE("/:id", s.CancelTranslation)
        translations.GET("/:id/status", s.GetTranslationStatus)
    }
    
    // File handling
    files := v1.Group("/files")
    {
        files.POST("/upload", s.UploadFile)
        files.GET("/:id/download", s.DownloadFile)
    }
}
```

### Handler Implementation

```go
// CreateTranslation handles translation requests
func (s *Server) CreateTranslation(c *gin.Context) {
    var req CreateTranslationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
        return
    }
    
    // Create translation job
    job, err := s.translationSvc.CreateTranslation(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // Publish event
    s.eventBus.Publish(events.Event{
        Type: "translation.created",
        Data: job,
        Timestamp: time.Now(),
    })
    
    c.JSON(http.StatusAccepted, CreateTranslationResponse{Job: job})
}
```

### Middleware

```go
// pkg/api/middleware/auth.go
func AuthMiddleware(authService AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "authorization header required",
            })
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        user, err := authService.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "invalid token",
            })
            return
        }
        
        c.Set("user", user)
        c.Next()
    }
}

// Rate limiting middleware
func RateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded",
            })
            return
        }
        c.Next()
    }
}
```

### WebSocket Support

```go
// pkg/api/websocket.go
type WebSocketHub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func (h *WebSocketHub) HandleTranslationProgress(jobID string) {
    // Register client for job progress updates
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                progress := h.getTranslationProgress(jobID)
                h.broadcastProgress(jobID, progress)
            }
        }
    }()
}

func (h *WebSocketHub) broadcastProgress(jobID string, progress Progress) {
    data, _ := json.Marshal(WebSocketMessage{
        Type: "progress",
        Data: progress,
    })
    
    for client := range h.clients {
        if client.jobID == jobID {
            select {
            case client.send <- data:
            default:
                close(client.send)
                delete(h.clients, client)
            }
        }
    }
}
```

## Plugin System

### Plugin Interface

```go
// pkg/plugins/interface.go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Initialize(config map[string]interface{}) error
    Shutdown() error
}

type TranslatorPlugin interface {
    Plugin
    Translate(ctx context.Context, text string, opts TranslationOptions) (string, error)
    SupportsLanguage(source, target string) bool
}

type FormatPlugin interface {
    Plugin
    Parse(data []byte) (*Document, error)
    Generate(doc *Document) ([]byte, error)
    Detect(data []byte) bool
}
```

### Plugin Manager

```go
// pkg/plugins/manager.go
type PluginManager struct {
    plugins map[string]Plugin
    mu      sync.RWMutex
}

func NewPluginManager() *PluginManager {
    return &PluginManager{
        plugins: make(map[string]Plugin),
    }
}

func (pm *PluginManager) LoadPlugin(path string) error {
    // Load plugin from file
    plug, err := plugin.Open(path)
    if err != nil {
        return fmt.Errorf("failed to load plugin: %w", err)
    }
    
    // Look for exported symbols
    sym, err := plug.Lookup("NewPlugin")
    if err != nil {
        return fmt.Errorf("plugin does not export NewPlugin: %w", err)
    }
    
    // Create plugin instance
    newPluginFunc, ok := sym.(func() Plugin)
    if !ok {
        return fmt.Errorf("unexpected type from plugin symbol")
    }
    
    plugin := newPluginFunc()
    pm.mu.Lock()
    pm.plugins[plugin.Name()] = plugin
    pm.mu.Unlock()
    
    return nil
}
```

### Example Plugin

```go
// plugins/custom_translator.go
package main

import "fmt"

type CustomTranslatorPlugin struct {
    config map[string]interface{}
}

func (p *CustomTranslatorPlugin) Name() string {
    return "CustomTranslator"
}

func (p *CustomTranslatorPlugin) Version() string {
    return "1.0.0"
}

func (p *CustomTranslatorPlugin) Description() string {
    return "Custom translation plugin example"
}

func (p *CustomTranslatorPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *CustomTranslatorPlugin) Shutdown() error {
    // Cleanup resources
    return nil
}

func (p *CustomTranslatorPlugin) Translate(ctx context.Context, text string, opts TranslationOptions) (string, error) {
    // Custom translation logic
    return fmt.Sprintf("[TRANSLATED: %s]", text), nil
}

func (p *CustomTranslatorPlugin) SupportsLanguage(source, target string) bool {
    return source == "en" && target == "test"
}

// Export plugin
func NewPlugin() Plugin {
    return &CustomTranslatorPlugin{}
}
```

## Testing

### Testing Strategy

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **End-to-End Tests**: Test complete workflows
4. **Performance Tests**: Test performance characteristics
5. **Security Tests**: Test security vulnerabilities

### Unit Testing

```go
// pkg/translator/translator_test.go
func TestTranslator_Translate(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "simple translation",
            input:    "Hello",
            expected: "Привет",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            translator := NewMockTranslator()
            result, err := translator.Translate(context.Background(), tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Mock Objects

```go
// test/mocks/translator.go
type MockTranslator struct {
    translations map[string]string
    t           *testing.T
}

func NewMockTranslator() *MockTranslator {
    return &MockTranslator{
        translations: map[string]string{
            "Hello": "Привет",
            "World": "Мир",
        },
    }
}

func (m *MockTranslator) Translate(ctx context.Context, text string) (string, error) {
    if result, exists := m.translations[text]; exists {
        return result, nil
    }
    return "", fmt.Errorf("translation not found for: %s", text)
}
```

### Integration Testing

```go
// test/integration/api_test.go
func TestAPI_CreateTranslation(t *testing.T) {
    // Setup test server
    server := setupTestServer(t)
    defer server.Close()
    
    // Create request
    req := CreateTranslationRequest{
        InputFile:    "test.fb2",
        SourceLang:    "ru",
        TargetLang:    "sr",
        Provider:      "mock",
    }
    
    reqBody, _ := json.Marshal(req)
    resp, err := http.Post(server.URL+"/api/v1/translations", "application/json", bytes.NewBuffer(reqBody))
    
    require.NoError(t, err)
    require.Equal(t, http.StatusAccepted, resp.StatusCode)
    
    var result CreateTranslationResponse
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)
    assert.NotEmpty(t, result.Job.ID)
}
```

### Test Utilities

```go
// test/utils.go
func setupTestServer(t *testing.T) *httptest.Server {
    router := gin.New()
    
    // Setup routes with mocked services
    translationSvc := &MockTranslationService{}
    authService := &MockAuthService{}
    eventBus := events.NewMemoryEventBus()
    
    server := api.NewServer(translationSvc, authService, eventBus)
    server.SetupRoutes(router)
    
    return httptest.NewServer(router)
}

func createTestFile(t *testing.T, content string) string {
    file, err := os.CreateTemp("", "test_*.fb2")
    require.NoError(t, err)
    
    _, err = file.WriteString(content)
    require.NoError(t, err)
    
    file.Close()
    return file.Name()
}
```

## Contributing

### Contribution Workflow

1. **Fork Repository**
   ```bash
   gh repo fork digital-vasic/translator
   cd translator
   ```

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/new-feature
   ```

3. **Make Changes**
   - Write code
   - Add tests
   - Update documentation

4. **Run Tests**
   ```bash
   make test-unit
   make test-integration
   make lint
   ```

5. **Commit Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

6. **Push and Create PR**
   ```bash
   git push origin feature/new-feature
   gh pr create --title "Add new feature" --body "Description of changes"
   ```

### Code Standards

#### 1. Code Style

```go
// Use meaningful variable names
func calculateTranslationCost(tokens int, pricePerToken float64) float64 {
    return float64(tokens) * pricePerToken
}

// Keep functions short and focused
func validateTranslationRequest(req *TranslationRequest) error {
    if req.SourceLang == "" {
        return errors.New("source language is required")
    }
    if req.TargetLang == "" {
        return errors.New("target language is required")
    }
    return nil
}

// Use descriptive function names
func isTranslationInProgress(status TranslationStatus) bool {
    return status == StatusQueued || status == StatusProcessing
}
```

#### 2. Error Handling

```go
// Always check for errors
func translateDocument(ctx context.Context, doc *Document) (*TranslatedDocument, error) {
    if doc == nil {
        return nil, errors.New("document cannot be nil")
    }
    
    // Use wrapped errors
    result, err := translateContent(ctx, doc.Content)
    if err != nil {
        return nil, fmt.Errorf("failed to translate content: %w", err)
    }
    
    return &TranslatedDocument{
        Content: result,
        Metadata: doc.Metadata,
    }, nil
}

// Define custom error types
type TranslationError struct {
    Code    string
    Message string
    Cause   error
}

func (e *TranslationError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

#### 3. Documentation

```go
// Package translator provides translation services for multiple LLM providers.
//
// Example usage:
//
//     t := translator.New(translator.Config{
//         Provider: "openai",
//         APIKey: "your-api-key",
//     })
//     result, err := t.Translate(ctx, "Hello", "en", "es")
//
package translator

// Translate translates text from source language to target language.
//
// ctx is the context for the translation operation.
// text is the source text to translate.
// sourceLang is the ISO code of the source language.
// targetLang is the ISO code of the target language.
//
// Returns the translated text and any error encountered.
func (t *Translator) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
    // Implementation...
}
```

### Pull Request Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests pass
- [ ] No breaking changes (or clearly documented)
```

## Debugging

### Debug Configuration

```json
{
  "logging": {
    "level": "debug",
    "format": "json",
    "output": "stdout",
    "fields": [
      "timestamp",
      "level",
      "component",
      "function",
      "line",
      "message"
    ]
  },
  "debug": {
    "pprof": {
      "enabled": true,
      "port": 6060
    },
    "trace": {
      "enabled": true,
      "sample_rate": 0.1
    }
  }
}
```

### Debug Tools

```go
// Debug endpoints for development
func (s *Server) setupDebugRoutes() {
    debug := s.router.Group("/debug")
    debug.Use(gin.Logger())
    debug.Use(gin.Recovery())
    
    // pprof endpoints
    debug.GET("/pprof/*", gin.WrapF(http.HandlerFunc(pprof.Index)))
    debug.GET("/pprof/cmdline", gin.WrapF(pprof.Cmdline))
    debug.GET("/pprof/profile", gin.WrapF(pprof.Profile))
    
    // Debug info
    debug.GET("/info", s.debugInfo)
    debug.GET/config", s.debugConfig)
    debug.GET/health", s.debugHealth)
}

func (s *Server) debugInfo(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "version":     s.version,
        "build_time":  s.buildTime,
        "git_commit":  s.gitCommit,
        "go_version": runtime.Version(),
        "goroutines": runtime.NumGoroutine(),
    })
}
```

### Common Debugging Scenarios

#### 1. Performance Issues

```go
// Add timing metrics
func (t *Translator) Translate(ctx context.Context, text string) (string, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        metrics.RecordTranslationTime(duration)
        log.Debug("translation completed", 
            map[string]interface{}{
                "duration_ms": duration.Milliseconds(),
                "text_length": len(text),
            })
    }()
    
    // Translation logic...
}
```

#### 2. Memory Leaks

```go
// Use runtime/pprof for memory profiling
func (s *Server) startMemoryProfiling() {
    go func() {
        for {
            time.Sleep(30 * time.Second)
            
            var memStats runtime.MemStats
            runtime.ReadMemStats(&memStats)
            
            log.Info("memory stats",
                map[string]interface{}{
                    "alloc_mb":     memStats.Alloc / 1024 / 1024,
                    "total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
                    "sys_mb":       memStats.Sys / 1024 / 1024,
                    "num_gc":       memStats.NumGC,
                })
        }
    }()
}
```

#### 3. Concurrency Issues

```go
// Use race detector in tests
//go test -race ./...

// Debug with trace
func (s *Server) enableTracing() {
    go func() {
        trace.Start(os.Stdout)
        defer trace.Stop()
        
        time.Sleep(60 * time.Second) // Trace for 1 minute
    }()
}
```

## Performance Optimization

### Profiling

```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Blocking profiling
go tool pprof http://localhost:6060/debug/pprof/block
```

### Optimization Strategies

#### 1. Caching

```go
// pkg/translator/cache.go
type TranslationCache struct {
    redis  *redis.Client
    local  *lru.Cache
    ttl    time.Duration
}

func (c *TranslationCache) Get(key string) (string, bool) {
    // Check local cache first
    if value, exists := c.local.Get(key); exists {
        return value.(string), true
    }
    
    // Check Redis cache
    value, err := c.redis.Get(key).Result()
    if err == nil {
        // Cache locally for next time
        c.local.Add(key, value)
        return value, true
    }
    
    return "", false
}
```

#### 2. Connection Pooling

```go
// pkg/transport/http.go
type HTTPClient struct {
    client *http.Client
    pool   *pool.ConnectionPool
}

func NewHTTPClient(maxConnections int) *HTTPClient {
    return &HTTPClient{
        client: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        maxConnections,
                MaxIdleConnsPerHost: maxConnections,
                IdleConnTimeout:      90 * time.Second,
            },
            Timeout: 30 * time.Second,
        },
        pool: pool.NewConnectionPool(maxConnections),
    }
}
```

#### 3. Batch Processing

```go
// pkg/batch/processor.go
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, items []TranslationItem) error {
    // Process items in parallel
    sem := make(chan struct{}, bp.maxConcurrency)
    
    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        
        go func(item TranslationItem) {
            defer wg.Done()
            
            sem <- struct{}{}
            defer func() { <-sem }()
            
            if err := bp.processItem(ctx, item); err != nil {
                log.Error("failed to process item", map[string]interface{}{
                    "item_id": item.ID,
                    "error":   err,
                })
            }
        }(item)
    }
    
    wg.Wait()
    return nil
}
```

## Security

### Input Validation

```go
// pkg/validation/validator.go
type InputValidator struct {
    maxFileSize    int64
    allowedTypes   []string
    maxTextLength int
}

func (v *InputValidator) ValidateTranslationRequest(req *TranslationRequest) error {
    // Validate file size
    if req.FileSize > v.maxFileSize {
        return ErrFileTooLarge
    }
    
    // Validate file type
    if !v.isAllowedType(req.FileType) {
        return ErrUnsupportedFileType
    }
    
    // Validate text length
    if len(req.Text) > v.maxTextLength {
        return ErrTextTooLong
    }
    
    // Sanitize input
    req.Text = v.sanitizeText(req.Text)
    
    return nil
}
```

### Authentication & Authorization

```go
// pkg/security/auth.go
type AuthService struct {
    jwtSecret []byte
    users     map[string]*User
}

func (a *AuthService) GenerateToken(user *User) (string, error) {
    claims := jwt.MapClaims{
        "sub":  user.ID,
        "name": user.Name,
        "role": user.Role,
        "exp":  time.Now().Add(time.Hour * 24).Unix(),
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(a.jwtSecret)
}

func (a *AuthService) ValidateToken(tokenString string) (*User, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return a.jwtSecret, nil
    })
    
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, ErrInvalidClaims
    }
    
    user, exists := a.users[claims["sub"].(string)]
    if !exists {
        return nil, ErrUserNotFound
    }
    
    return user, nil
}
```

### Rate Limiting

```go
// pkg/security/ratelimit.go
type RateLimiter struct {
    tokens map[string]*rate.Limiter
    mu     sync.RWMutex
    rate   rate.Limit
    burst  int
}

func (rl *RateLimiter) Allow(userID string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if _, exists := rl.tokens[userID]; !exists {
        rl.tokens[userID] = rate.NewLimiter(rl.rate, rl.burst)
    }
    
    return rl.tokens[userID].Allow()
}
```

### Security Headers

```go
// pkg/api/middleware/security.go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    }
}
```

---

## Getting Help

### Developer Resources

- **Documentation**: https://docs.translator.digital/developer
- **API Reference**: https://api.translator.digital/docs
- **Examples**: https://github.com/digital-vasic/translator/examples
- **Discord**: https://discord.gg/translator-dev
- **Stack Overflow**: [translator tag](https://stackoverflow.com/questions/tagged/translator)

### Reporting Issues

1. **Bug Reports**: Use GitHub issues with bug report template
2. **Security Issues**: Email security@translator.digital
3. **Feature Requests**: Use GitHub discussions

### Contributing Guidelines

- Follow all coding standards
- Add tests for new features
- Update documentation
- Ensure all tests pass
- Keep pull requests focused

For more information, see our [Contributing Guide](https://github.com/digital-vasic/translator/blob/main/CONTRIBUTING.md).