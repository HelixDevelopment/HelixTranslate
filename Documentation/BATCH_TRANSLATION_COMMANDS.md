# Batch Book Translation Commands

This document provides ready-to-use commands for batch translating all books from the `Books/` directory to Serbian Cyrillic EPUB format using multiple LLM coordinators.

## Prerequisites

### For Cloud LLMs (Command 1)
Set your API keys as environment variables:
```bash
export DEEPSEEK_API_KEY="your-deepseek-api-key"
export ZHIPU_API_KEY="your-zhipu-api-key"
export OPENAI_API_KEY="your-openai-api-key"
export ANTHROPIC_API_KEY="your-anthropic-api-key"
export QWEN_API_KEY="your-qwen-api-key"
```

### For Local Llama.cpp Models (Both Commands)
Install and configure Ollama with llama.cpp models:

```bash
# Install Ollama (if not already installed)
# macOS:
brew install ollama

# Start Ollama service
ollama serve &

# Pull recommended models for translation
ollama pull qwen2.5:14b      # Best quality local model
ollama pull qwen2.5:7b       # Good balance
ollama pull llama3:8b        # Alternative option
ollama pull mistral:7b       # Another good option

# Enable Ollama in environment
export OLLAMA_ENABLED="true"
export OLLAMA_MODEL="qwen2.5:14b"  # Use best available model
```

---

## Command 1: All Available LLMs (Cloud APIs + Local Models)

This command uses **all configured LLM providers** - both cloud APIs (DeepSeek, Zhipu, OpenAI, Anthropic, Qwen) AND local llama.cpp models via Ollama.

### Setup Environment
```bash
# Set all your API keys
export DEEPSEEK_API_KEY="sk-your-deepseek-key"
export ZHIPU_API_KEY="your-zhipu-key"
export OPENAI_API_KEY="sk-your-openai-key"
export ANTHROPIC_API_KEY="sk-ant-your-anthropic-key"
export QWEN_API_KEY="your-qwen-key"

# Enable local Ollama models
export OLLAMA_ENABLED="true"
export OLLAMA_MODEL="qwen2.5:14b"

# Optional: Configure specific models for each provider
export DEEPSEEK_MODEL="deepseek-chat"
export ZHIPU_MODEL="glm-4"
export OPENAI_MODEL="gpt-4"
export ANTHROPIC_MODEL="claude-3-sonnet-20240229"
export QWEN_MODEL="qwen-plus"
```

### Run Translation Command
```bash
./build/translator \
  -input Books \
  -output Books/Translated \
  -locale sr \
  -provider multi-llm \
  -format epub \
  -script cyrillic \
  -recursive true \
  -parallel true \
  -max-concurrency 3
```

### Expected Behavior
- **Multi-LLM Coordinator** will initialize with ~15-20 instances across all providers
- **Priority**: API key providers (10) get 3 instances each, OAuth (5) gets 2, local (1) gets 1
- **Automatic failover**: If one LLM fails, coordinator automatically tries another
- **Rate limit handling**: Temporarily disables rate-limited instances
- **Parallel processing**: Translates up to 3 books simultaneously

### Example Output
```
Multi-LLM coordinator ready with 18 instances
Providers: [deepseek openai anthropic zhipu qwen ollama]
Processing 25 files from directory
Processing file 1/25: book1.epub
Attempting translation with deepseek-1 (Attempt 1)
Translation successful with deepseek-1
...
```

---

## Command 2: Local Models Only (No Cloud APIs)

This command uses **ONLY local models** - no cloud API calls, completely offline, free to use.

### Two Options for Local Models:

#### Option A: Direct llama.cpp (Recommended)
Uses `llama.cpp` directly with automatic model download and hardware optimization.

**Prerequisites:**
```bash
# Install llama.cpp
brew install llama.cpp

# Verify installation
llama-cli --version
```

**Run Translation:**
```bash
./build/translator \
  -input Books \
  -output Books/Translated_Llamacpp \
  -locale sr \
  -provider llamacpp \
  -format epub \
  -script cyrillic \
  -recursive true \
  -parallel false
```

**Features:**
- Automatic model selection based on your hardware
- Auto-downloads optimal GGUF models from Hugging Face
- Hardware-aware optimization (GPU detection, thread configuration)
- No manual model management needed

#### Option B: Ollama
Uses Ollama (which uses llama.cpp internally but provides easier model management).

**Prerequisites:**
```bash
# Install Ollama
brew install ollama

# Start Ollama service
ollama serve &

# Pull models
ollama pull qwen2.5:14b
```

**Run Translation:**
```bash
export OLLAMA_ENABLED="true"
export OLLAMA_MODEL="qwen2.5:14b"

./build/translator \
  -input Books \
  -output Books/Translated_Ollama \
  -locale sr \
  -provider ollama \
  -format epub \
  -script cyrillic \
  -recursive true \
  -parallel false
```

### Expected Behavior (Both Options)
- **Completely offline**: No external API calls
- **Free**: No API costs
- **Sequential processing**: Processes one book at a time (local models are resource-intensive)
- **Lower concurrency**: Recommended to use parallel=false to avoid overwhelming local GPU/CPU

### Performance Notes
- **Speed**: Slower than cloud APIs but no costs
- **Quality**: High-quality translations with modern models
- **Resources**: Requires good CPU/GPU (M1/M2 Mac recommended, or NVIDIA GPU)
- **llamacpp vs ollama**: llamacpp is more direct, ollama provides better model management

---

## Command 3: Alternative - Single Provider (For Testing)

If you want to test with just one specific provider:

### DeepSeek Only
```bash
export DEEPSEEK_API_KEY="sk-your-key"
./build/translator -input Books -output Books/Translated_DeepSeek -locale sr -provider deepseek -format epub -script cyrillic -recursive true
```

### Llama.cpp Only (Simplest Local)
```bash
# Direct llama.cpp - no configuration needed
./build/translator -input Books -output Books/Translated_Llamacpp -locale sr -provider llamacpp -format epub -script cyrillic -recursive true
```

---

## Directory Structure

### Input
```
Books/
├── book1.epub
├── book2.fb2
├── book3.pdf
└── subdirectory/
    └── book4.epub
```

### Output (Command 1 - All LLMs)
```
Books/Translated/
├── book1_sr.epub
├── book2_sr.epub
├── book3_sr.epub
└── subdirectory/
    └── book4_sr.epub
```

### Output (Command 2 - Local Only)
```
Books/Translated_Local/
├── book1_sr.epub
├── book2_sr.epub
├── book3_sr.epub
└── subdirectory/
    └── book4_sr.epub
```

---

## Monitoring Translation Progress

### Real-time Log Monitoring
```bash
# Watch translation progress
tail -f /tmp/translation.log

# Monitor specific events
tail -f /tmp/translation.log | grep -E "(chapter|Translation|success|failed)"
```

### Check Multi-LLM Coordinator Status
```bash
# The translator will output events like:
# - "Multi-LLM coordinator ready with N instances"
# - "Attempting translation with provider-X"
# - "Translation successful with provider-Y"
# - "Instance Z re-enabled after cooldown"
```

---

## Troubleshooting

### No LLM instances available
**Problem**: `No LLM instances available`
**Solution**:
- Command 1: Ensure at least one API key is set
- Command 2: Ensure Ollama is running (`ollama serve`)

### Rate limit errors
**Problem**: `rate limit exceeded`
**Solution**: The multi-llm coordinator automatically handles this by:
1. Temporarily disabling the rate-limited instance
2. Trying other available instances
3. Re-enabling the instance after 30 seconds

### Out of memory (local models)
**Problem**: System slowdown or OOM errors
**Solution**:
- Use smaller model: `export OLLAMA_MODEL="qwen2.5:7b"`
- Disable parallel processing: `-parallel false`
- Reduce concurrency: `-max-concurrency 1`

---

## Performance Comparison

| Configuration | Speed | Cost | Quality | Offline |
|--------------|-------|------|---------|---------|
| **All LLMs** | ⚡⚡⚡⚡⚡ | 💰💰 | ⭐⭐⭐⭐⭐ | ❌ |
| **Local Only** | ⚡⚡ | 💰 Free | ⭐⭐⭐⭐ | ✅ |
| **DeepSeek Only** | ⚡⚡⚡⚡ | 💰 | ⭐⭐⭐⭐ | ❌ |

---

## Quick Copy-Paste Commands

### Command 1: Full Production (All LLMs)
```bash
# One-liner setup + run
export DEEPSEEK_API_KEY="your-key" && \
export ZHIPU_API_KEY="your-key" && \
export OLLAMA_ENABLED="true" && \
export OLLAMA_MODEL="qwen2.5:14b" && \
./build/translator -input Books -output Books/Translated -locale sr -provider multi-llm -format epub -script cyrillic -recursive true -parallel true -max-concurrency 3
```

### Command 2: Local Models Only
```bash
# Option A: Direct llama.cpp (recommended - auto-downloads models, hardware-optimized)
brew install llama.cpp && \
./build/translator -input Books -output Books/Translated_Local -locale sr -provider llamacpp -format epub -script cyrillic -recursive true -parallel false

# Option B: Ollama (if you prefer Ollama for model management)
brew install ollama && ollama serve & && ollama pull qwen2.5:14b && \
export OLLAMA_ENABLED="true" && export OLLAMA_MODEL="qwen2.5:14b" && \
./build/translator -input Books -output Books/Translated_Local -locale sr -provider ollama -format epub -script cyrillic -recursive true -parallel false
```

---

## Additional Options

### Process Specific File Types Only
```bash
# Only EPUB files
./build/translator -input Books -output Books/Translated -locale sr -provider multi-llm -format epub -script cyrillic -file-pattern "*.epub"

# Only FB2 files
./build/translator -input Books -output Books/Translated -locale sr -provider multi-llm -format epub -script cyrillic -file-pattern "*.fb2"
```

### Custom Output File Naming
By default, output files are named: `{original_name}_sr.epub`

---

## License & Credits

Generated with Claude Code - https://claude.com/claude-code
Translation powered by multi-LLM coordination for maximum reliability and quality.
