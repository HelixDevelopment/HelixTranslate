# Multi-LLM Priority System

## Overview
The multi-LLM coordinator now implements a priority-based load distribution system that ensures API-key providers (which you pay for) handle significantly more translation work than OAuth or free providers.

## Priority Levels

### Priority 10: API Key Providers (High Priority)
**3 instances per provider** - 75% of total workload

Providers in this category:
- ✅ **OpenAI** (GPT-4, etc.) - `OPENAI_API_KEY`
- ✅ **Anthropic** (Claude) - `ANTHROPIC_API_KEY`
- ✅ **DeepSeek** - `DEEPSEEK_API_KEY`
- ✅ **Zhipu AI** (GLM-4) - `ZHIPU_API_KEY`
- ✅ **Qwen** (with API key) - `QWEN_API_KEY`

### Priority 5: OAuth Providers (Medium Priority)
**2 instances per provider** - 25% of total workload

Providers in this category:
- ✅ **Qwen** (with OAuth credentials, no API key)

### Priority 1: Free/Local Providers (Low Priority)
**1 instance per provider** - ~10-15% of total workload

Providers in this category:
- ✅ **Ollama** (local models) - `OLLAMA_ENABLED=true`

## Workload Distribution Examples

### Example 1: Two API Key Providers
```bash
export DEEPSEEK_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"
```

**Result:**
- DeepSeek: 3 instances (50%)
- Zhipu: 3 instances (50%)
- **Total: 6 instances**

### Example 2: Two API Keys + OAuth
```bash
export DEEPSEEK_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"
# Qwen OAuth credentials detected automatically
```

**Result:**
- DeepSeek: 3 instances (37.5%)
- Zhipu: 3 instances (37.5%)
- Qwen (OAuth): 2 instances (25%)
- **Total: 8 instances**

### Example 3: Mixed Environment
```bash
export OPENAI_API_KEY="your-key"
export DEEPSEEK_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"
export QWEN_API_KEY="your-key"
export OLLAMA_ENABLED="true"
```

**Result:**
- OpenAI: 3 instances (23%)
- DeepSeek: 3 instances (23%)
- Zhipu: 3 instances (23%)
- Qwen (API key): 3 instances (23%)
- Ollama: 1 instance (8%)
- **Total: 13 instances**

**API key providers handle 92% of work, free provider handles 8%**

## How It Works

### 1. Provider Discovery
The coordinator automatically detects available providers by checking:
- Environment variables for API keys
- OAuth credential files for Qwen (`~/.qwen/oauth_creds.json`, `~/.translator/qwen_credentials.json`)
- Ollama availability flag

### 2. Priority Assignment
Each provider is assigned a priority based on authentication method:

```go
// API key provider detected
priority = 10  // High priority

// OAuth provider detected (Qwen without API key)
priority = 5   // Medium priority

// Free/local provider detected (Ollama)
priority = 1   // Low priority
```

### 3. Instance Creation
The coordinator creates multiple instances based on priority:

```go
func getInstanceCount(priority int) int {
    switch {
    case priority >= 10:
        return 3  // API key providers
    case priority >= 5:
        return 2  // OAuth providers
    default:
        return 1  // Free/local providers
    }
}
```

### 4. Round-Robin Distribution
Instances are selected using round-robin, naturally distributing work proportionally:
- More instances = more work
- API key providers (3 instances) get 3x work of free providers (1 instance)

## Benefits

### 1. Cost Efficiency
- **Maximize ROI**: Get the most out of your paid API subscriptions
- **Reduce Waste**: Avoid underutilizing expensive API keys
- **Smart Fallback**: OAuth/free providers provide backup capacity

### 2. Performance
- **Better Quality**: API-key providers typically offer better models
- **Faster Response**: Paid providers often have better SLAs
- **Load Balancing**: Work distributed across multiple high-priority instances

### 3. Reliability
- **Automatic Retry**: Failed API key provider → try another API key provider
- **Graceful Degradation**: If API key providers fail → fall back to OAuth/free
- **No Manual Configuration**: Priorities assigned automatically

## Implementation Details

### Code Location
- **Priority Assignment**: `pkg/coordination/multi_llm.go:discoverProviders()`
- **Instance Creation**: `pkg/coordination/multi_llm.go:initializeLLMInstances()`
- **Round-Robin Selection**: `pkg/coordination/multi_llm.go:getNextInstance()`

### Key Changes

**1. Added Priority Field to LLMInstance:**
```go
type LLMInstance struct {
    ID         string
    Translator translator.Translator
    Provider   string
    Model      string
    Priority   int       // Higher priority = more instances
    Available  bool
    LastUsed   time.Time
    mu         sync.Mutex
}
```

**2. Smart Instance Count Calculation:**
```go
// API key (priority 10) gets 3 instances
// OAuth (priority 5) gets 2 instances
// Free (priority 1) gets 1 instance
instanceCount := getInstanceCount(priority)
```

**3. Automatic Qwen OAuth Detection:**
```go
// Check for OAuth credentials if no API key
qwenOAuthPaths := []string{
    homeDir + "/.translator/qwen_credentials.json",
    homeDir + "/.qwen/oauth_creds.json",
}
for _, path := range qwenOAuthPaths {
    if _, err := os.Stat(path); err == nil {
        providers["qwen"] = map[string]interface{}{
            "api_key":  "", // OAuth will be used
            "model":    getEnvOrDefault("QWEN_MODEL", "qwen-plus"),
            "priority": 5, // OAuth = medium priority
        }
        break
    }
}
```

## Verification

### Check Instance Distribution
```bash
export DEEPSEEK_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"

./build/translator -input book.epub -provider multi-llm -locale sr | grep "instances"
```

**Expected Output:**
```
[multi_llm_init] Initializing 8 LLM instances across 3 providers (prioritizing API key providers)
Using translator: multi-llm-coordinator (8 instances)
```

### Monitor Workload Distribution
Watch the translation attempts in real-time:
```bash
./build/translator -input book.epub -provider multi-llm -locale sr 2>&1 | grep "translation_attempt"
```

**Expected Pattern:**
```
[translation_attempt] Attempting translation with deepseek-1 (Attempt 1)
[translation_attempt] Attempting translation with deepseek-2 (Attempt 1)
[translation_attempt] Attempting translation with zhipu-3 (Attempt 1)
[translation_attempt] Attempting translation with deepseek-4 (Attempt 1)
[translation_attempt] Attempting translation with zhipu-5 (Attempt 1)
[translation_attempt] Attempting translation with qwen-6 (Attempt 1)
[translation_attempt] Attempting translation with deepseek-1 (Attempt 1)
```

Notice: API key providers (deepseek, zhipu) appear more frequently than OAuth providers (qwen).

## Best Practices

### 1. Always Provide API Keys
For best performance and cost efficiency, always provide API keys:
```bash
export DEEPSEEK_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export QWEN_API_KEY="your-key"
```

### 2. Use OAuth as Backup
OAuth providers serve as reliable backup:
- Set up Qwen OAuth for additional capacity
- No API key needed - uses existing Qwen Code auth
- Automatically gets medium priority

### 3. Enable Ollama for Offline
For offline capability or testing:
```bash
export OLLAMA_ENABLED="true"
export OLLAMA_MODEL="llama3:8b"
```
- Gets lowest priority (1 instance)
- Only used when other providers unavailable
- Free and private

### 4. Monitor Costs
Higher priority = more API calls = higher costs. Balance accordingly:
- **High volume**: Use DeepSeek (cost-effective)
- **Best quality**: Use GPT-4 or Claude
- **Backup**: Enable OAuth/Ollama

## Future Enhancements

Potential improvements to the priority system:

1. **Dynamic Priority Adjustment**
   - Increase priority for faster providers
   - Decrease priority for rate-limited providers
   - Adjust based on success/failure rates

2. **Custom Priority Configuration**
   - Allow users to override default priorities
   - Environment variable: `PROVIDER_PRIORITY_OPENAI=15`
   - Config file support

3. **Cost-Based Priority**
   - Factor in API costs per token
   - Prefer cheaper providers when quality is similar
   - Budget-aware load balancing

4. **Quality-Based Priority**
   - Monitor translation quality scores
   - Adjust priorities based on verification results
   - Learn from user feedback

## Conclusion

The priority system ensures that:
- ✅ **API key providers do the heavy lifting** (3 instances each)
- ✅ **OAuth providers provide backup capacity** (2 instances each)
- ✅ **Free/local providers fill gaps** (1 instance each)
- ✅ **Zero configuration required** (automatic detection)
- ✅ **Optimal cost efficiency** (maximize paid API usage)

This results in faster, more reliable translations while ensuring you get maximum value from your API subscriptions.
