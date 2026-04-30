# LLMsVerifier - API Surface Quick Reference

## Package Hierarchy

```
digital.vasic.llmsverifier/
├── llmverifier/       # Core verification engine
├── providers/          # 25+ LLM provider adapters
├── verification/       # Verification pipeline
├── scoring/            # Scoring & ranking system
├── api/                # REST API server
├── database/           # SQLite + SQLCipher storage
├── config/             # Configuration management
├── client/             # HTTP client for LLM APIs
├── auth/               # JWT + RBAC + LDAP auth
├── capabilities/       # Dynamic capability detection
├── ai/                 # AI-powered recommendations
├── analytics/          # Predictive analytics
├── enhanced/           # Enterprise features
├── tui/                # Terminal UI
├── web/                # Web application
├── sdk/                # Language SDKs
├── messaging/          # Message broker factory
├── monitoring/         # Metrics & monitoring
├── scheduler/          # Job scheduler
├── notifications/      # Notification system
├── security/           # Security features
├── failover/           # Failover mechanisms
├── performance/        # Performance testing
├── multimodal/         # Multimodal support
├── events/             # Event system
└── logging/            # Logging system
```

## Top 50 Most Important Functions

### Verification
| Function | Package | Description |
|----------|---------|-------------|
| `llmverifier.New(cfg)` | llmverifier | Create Verifier instance |
| `llmverifier.LoadConfig(path)` | llmverifier | Load config from YAML |
| `Verifier.Verify()` | llmverifier | Run full verification |
| `Verifier.GenerateMarkdownReport(results, dir)` | llmverifier | Generate MD report |
| `Verifier.GenerateJSONReport(results, dir)` | llmverifier | Generate JSON report |
| `verification.NewVerifier(db)` | verification | Create verification service |
| `Verifier.Verify(ctx, req)` | verification | Verify single model |
| `NewCodeVerificationService(httpClient, logger)` | verification | Code verification service |
| `VerifyModelCodeVisibility(ctx, modelID, providerID, client)` | verification | "Do you see my code?" test |
| `NewCodingCapabilityVerificationService(httpClient, logger)` | verification | Coding capability service |
| `VerifyModelCodingCapabilities(ctx, modelID, providerID, client)` | verification | Full coding test |

### Providers
| Function | Package | Description |
|----------|---------|-------------|
| `providers.NewProviderRegistry()` | providers | Create provider registry |
| `ProviderRegistry.GetConfig(name)` | providers | Get provider config |
| `NewModelProviderService(configPath, logger)` | providers | 3-tier discovery service |
| `ModelProviderService.RegisterProvider(id, baseURL, apiKey)` | providers | Register provider |
| `ModelProviderService.GetAllModelsWithVerification(ctx)` | providers | Get all verified models |
| `NewVerifiedConfigGenerator(service, logger, outputDir)` | providers | Generate verified configs |
| `VerifiedConfigGenerator.GenerateVerifiedConfig()` | providers | Create verified config |
| `NewErrorClassifier(provider)` | providers | Classify API errors |

### Scoring
| Function | Package | Description |
|----------|---------|-------------|
| `scoring.NewScoringEngine(db, client, logger)` | scoring | Create scoring engine |
| `ScoringEngine.CalculateComprehensiveScore(ctx, modelID, config)` | scoring | Calculate full score |
| `scoring.DefaultScoringConfig()` | scoring | Default scoring weights |

### API
| Function | Package | Description |
|----------|---------|-------------|
| `api.NewServer(cfg, db)` | api | Create API server |
| `Server.Start(port)` | api | Start HTTP server |
| `Server.Stop()` | api | Stop server |

### Database
| Function | Package | Description |
|----------|---------|-------------|
| `database.New(path, encryptionKey)` | database | Open encrypted SQLite |
| `Database.CreateProvider(p)` | database | Create provider |
| `Database.GetProvider(id)` | database | Get provider by ID |
| `Database.ListProviders(filters)` | database | List providers |
| `Database.CreateModel(m)` | database | Create model |
| `Database.GetModel(id)` | database | Get model by ID |
| `Database.ListModels(filters)` | database | List models with filters |
| `Database.CreateVerificationResult(r)` | database | Save verification |
| `Database.Ping()` | database | Health check |

### Config
| Function | Package | Description |
|----------|---------|-------------|
| `config.LoadFromFile(path)` | config | Load config (YAML/JSON/TOML) |
| `config.SaveToFile(cfg, path)` | config | Save config to file |

### Auth
| Function | Package | Description |
|----------|---------|-------------|
| `auth.NewAuthManager(jwtSecret, ldapConfig)` | auth | Create auth manager |
| `AuthManager.RegisterClient(name, desc, perms, rateLimit)` | auth | Register API client |
| `AuthManager.GenerateJWT(client)` | auth | Generate JWT token |
| `AuthManager.ValidateJWT(token)` | auth | Validate JWT |

### Client
| Function | Package | Description |
|----------|---------|-------------|
| `client.New(baseURL)` | client | Create API client |
| `Client.Login(username, password)` | client | Authenticate |
| `Client.GetModels()` | client | List models via API |

### Capabilities
| Function | Package | Description |
|----------|---------|-------------|
| `capabilities.NewDetector()` | capabilities | Create capability detector |
| `Detector.DetectProviderCapabilities(ctx, provider, apiKey)` | capabilities | Detect capabilities |

## Configuration Schema

### Minimal Config
```yaml
global:
  api_key: "${API_KEY}"
database:
  path: "./verifier.db"
api:
  port: "8080"
  jwt_secret: "secret"
```

### Full Config
See main analysis document Section H.

## API Endpoints

```
GET  /api/health              -> {"status": "healthy", "timestamp": 1234}
GET  /api/models?provider_id=1&status=verified&min_score=7.0&search=gpt&limit=50
GET  /api/models/{id}
POST /api/models/{id}/verify
GET  /api/providers
POST /api/providers
```

## Provider Support Matrix

| Provider | File | Streaming | Vision | Function Calling | Code | Embeddings |
|----------|------|-----------|--------|-----------------|------|------------|
| OpenAI | openai.go | SSE | Yes | Yes | Yes | Yes |
| Anthropic | anthropic.go | SSE | Yes | Yes | Yes | Yes |
| Groq | groq.go | SSE | Yes | Yes | Yes | No |
| DeepSeek | deepseek.go | SSE | No | Yes | Yes | Yes |
| Cohere | cohere.go | SSE | No | Yes | Yes | Yes |
| Mistral | mistral.go | SSE | No | Yes | Yes | Yes |
| xAI | xai.go | SSE | No | Yes | Yes | No |
| Replicate | replicate.go | No | Yes | No | Yes | No |
| Cerebras | cerebras.go | SSE | No | Yes | Yes | No |
| Cloudflare | cloudflare.go | SSE | No | No | Yes | Yes |
| SiliconFlow | siliconflow.go | SSE | Yes | Yes | Yes | Yes |
| Together AI | togetherai.go | SSE | Yes | Yes | Yes | Yes |
| Hyperbolic | hyperbolic.go | SSE | No | Yes | Yes | Yes |
| SambaNova | sambanova.go | SSE | No | Yes | Yes | No |
| Moonshot/Kimi | kimi.go | SSE | No | Yes | Yes | No |
| Qwen/DashScope | qwen.go | SSE | Yes | Yes | Yes | Yes |
| 10+ more | ... | ... | ... | ... | ... | ... |

## Scoring Weights

```go
DefaultScoreWeights() = {
    ResponseSpeed:      0.25  // Latency
    ModelEfficiency:    0.20  // Throughput, context window
    CostEffectiveness:  0.25  // Price per token
    Capability:         0.20  // Feature breadth
    Recency:            0.10  // Last update date
}
```

Score range: 0.0 - 10.0

## Database Schema (Key Tables)

```sql
providers          -- Provider configurations
models             -- Model metadata and capabilities
verification_results -- Verification outcomes
api_keys           -- API key storage
config_exports     -- Exported configurations
events             -- System events
issues             -- Detected issues
schedules          -- Verification schedules
pricing            -- Model pricing data
users              -- User accounts
sessions           -- Active sessions
migrations         -- Schema migrations
verification_scores -- Calculated scores
```

## Integration for HelixTranslate

```go
// 1. Import
go get digital.vasic.llmsverifier

// 2. Load config
cfg, _ := config.LoadFromFile("config.yaml")

// 3. Create verifier
v := llmverifier.New(cfg)

// 4. Verify
results, _ := v.Verify()

// 5. Score
db, _ := database.New(cfg.Database.Path, "")
engine := scoring.NewScoringEngine(db, modelsDevClient, logger)
score, _ := engine.CalculateComprehensiveScore(ctx, "model-id", scoring.DefaultScoringConfig())

// 6. Report
v.GenerateMarkdownReport(results, "./reports")
```
