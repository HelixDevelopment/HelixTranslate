# CLI Reference

Complete command-line interface guide for the Universal Ebook Translator.

## Usage

```bash
translator [options] -input <file>
```

## Required Options

- `-i, -input <file>` - Input ebook file (any format: FB2, EPUB, TXT, HTML)

## Translation Options

- `-locale <code>` - Target language locale (e.g., sr, de, fr, es)
- `-language <name>` - Target language name (e.g., Serbian, German, French) - case-insensitive
- `-source <lang>` - Source language (optional, auto-detected if not specified)
- `-script <type>` - Output script (default, cyrillic, latin, arabic, etc.) [default: default]

## Provider Options

- `-p, -provider <name>` - Translation provider:
  - `dictionary` - Fast dictionary-based translation (default)
  - `openai` - OpenAI GPT models
  - `anthropic` - Anthropic Claude models
  - `zhipu` - Zhipu AI GLM-4 models
  - `deepseek` - DeepSeek models
  - `qwen` - Alibaba Qwen models
  - `gemini` - Google Gemini models
  - `ollama` - Local Ollama models
  - `llamacpp` - Local llama.cpp models
  - `multi-llm` - Multi-LLM coordinator (uses multiple providers)
- `-model <name>` - LLM model name (e.g., gpt-4, claude-3-sonnet)
- `-api-key <key>` - API key for LLM provider
- `-base-url <url>` - Base URL for LLM provider

## Multi-LLM Options

When using `-provider multi-llm`, additional options control LLM provider selection:

- `-disable-local-llms` - Disable local LLM providers (Ollama, llama.cpp), use only API providers
- `-prefer-distributed` - Prefer distributed workers over local LLMs (when available)

## Output Options

- `-o, -output <file>` - Output file (auto-generated if not specified)
- `-f, -format <format>` - Output format (epub, fb2, txt) [default: epub]

## Utility Options

- `-detect` - Detect source language and exit
- `-create-config <file>` - Create a config file template
- `-v, -version` - Show version
- `-h, -help` - Show help

## Environment Variables

### API Keys
- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `ZHIPU_API_KEY` - Zhipu AI API key
- `DEEPSEEK_API_KEY` - DeepSeek API key
- `QWEN_API_KEY` - Qwen (Alibaba Cloud) API key
- `GEMINI_API_KEY` - Google Gemini API key

### Local LLM Configuration
- `OLLAMA_ENABLED` - Enable Ollama integration ("true" to enable)
- `OLLAMA_MODEL` - Default Ollama model (default: "llama3:8b")

## Examples

### Basic Translation
```bash
# Translate any ebook to Serbian (auto-detect source)
translator -input book.epub

# Translate to German
translator -input book.epub -locale de
translator -input book.epub -language German

# Specify source language
translator -input book.epub -source en -locale de
```

### Provider-Specific Translation
```bash
# OpenAI GPT-4
export OPENAI_API_KEY="your-key"
translator -input book.epub -provider openai -model gpt-4

# Anthropic Claude
export ANTHROPIC_API_KEY="your-key"
translator -input book.epub -provider anthropic -model claude-3-sonnet-20240229

# Zhipu AI (GLM-4)
export ZHIPU_API_KEY="your-key"
translator -input book.epub -provider zhipu

# DeepSeek
export DEEPSEEK_API_KEY="your-key"
translator -input book.epub -provider deepseek

# Google Gemini
export GEMINI_API_KEY="your-key"
translator -input book.epub -provider gemini -model gemini-pro

# Local Ollama
translator -input book.epub -provider ollama -model llama3:8b
```

### Multi-LLM Coordinator
```bash
# Use multiple LLM providers with automatic failover
export OPENAI_API_KEY="your-key"
export DEEPSEEK_API_KEY="your-key"
translator -input book.epub -provider multi-llm

# Disable local LLMs, use only API providers
translator -input book.epub -provider multi-llm -disable-local-llms

# Prefer distributed workers (when available)
translator -input book.epub -provider multi-llm -prefer-distributed
```

### Output Formats
```bash
# Output as EPUB (default)
translator -input book.fb2 -output book.epub

# Output as plain text
translator -input book.epub -format txt

# Serbian Latin script
translator -input book.fb2 -script latin
```

### Utility Commands
```bash
# Detect language only
translator -input mystery_book.epub -detect

# Create config template
translator -create-config my_config.json

# Show version
translator -version
```

## Multi-LLM Coordinator Details

The multi-LLM coordinator automatically discovers and uses available LLM providers based on API keys and environment configuration. It provides:

- **Automatic failover** between providers
- **Load balancing** across multiple instances per provider
- **Priority-based selection** (API keys > OAuth > free/local)
- **Rate limit handling** with automatic cooldown
- **Consensus translation** for improved quality

### Provider Priority
1. **API Key providers** (10 priority): OpenAI, Anthropic, Zhipu, DeepSeek, Qwen
2. **OAuth providers** (5 priority): Qwen (OAuth)
3. **Local/Free providers** (1 priority): Ollama

### Instance Count per Provider
- API key providers: 3 instances each
- OAuth providers: 2 instances each
- Local providers: 1 instance each

### Configuration Flags
- `--disable-local-llms`: Skips Ollama and other local LLM providers, using only remote API providers
- `--prefer-distributed`: Logs preference for distributed workers (implementation depends on deployment setup)

## Error Handling

The CLI provides clear error messages for common issues:

- Missing input file
- Unsupported language codes
- Invalid provider configuration
- API key issues
- Network connectivity problems

All errors exit with appropriate status codes for scripting.</content>
</xai:function_call">Write file documentation/CLI.md