---
title: "Complete User Manual"
description: "Comprehensive guide for using the Universal Ebook Translator"
date: "2024-01-15"
weight: 5
---

# Complete User Manual

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [CLI Usage](#cli-usage)
5. [Web Interface](#web-interface)
6. [File Formats](#file-formats)
7. [Translation Providers](#translation-providers)
8. [Quality Control](#quality-control)
9. [Batch Processing](#batch-processing)
10. [Distributed Processing](#distributed-processing)
11. [Troubleshooting](#troubleshooting)
12. [Advanced Features](#advanced-features)

## Installation

### System Requirements

- **Operating System**: Windows 10+, macOS 10.15+, Linux (Ubuntu 18.04+)
- **Memory**: Minimum 4GB RAM (8GB recommended for large files)
- **Storage**: 1GB free space for application
- **Network**: Internet connection for cloud providers
- **Go Runtime**: 1.25.2+ (if building from source)

### Installation Methods

#### Method 1: Pre-compiled Binaries (Recommended)

**Windows**:
```powershell
# Download using PowerShell
Invoke-WebRequest -Uri "https://github.com/digital-vasic/translator/releases/latest/download/translator-windows-amd64.zip" -OutFile "translator.zip"
Expand-Archive -Path "translator.zip" -DestinationPath "."
Copy-Item "translator-windows-amd64.exe" "translator.exe"
```

**macOS**:
```bash
# Download using curl
curl -L https://github.com/digital-vasic/translator/releases/latest/download/translator-macos-amd64.tar.gz | tar xz
chmod +x translator-macos-amd64
sudo mv translator-macos-amd64 /usr/local/bin/translator
```

**Linux**:
```bash
# Download and install
curl -L https://github.com/digital-vasic/translator/releases/latest/download/translator-linux-amd64.tar.gz | tar xz
chmod +x translator-linux-amd64
sudo mv translator-linux-amd64 /usr/local/bin/translator
```

#### Method 2: Go Install

```bash
# Install directly from repository
go install github.com/digital-vasic/translator/cmd/cli@latest

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Method 3: Build from Source

```bash
# Clone repository
git clone https://github.com/digital-vasic/translator.git
cd translator

# Build
make build

# Install
sudo make install
```

#### Method 4: Docker

```bash
# Pull image
docker pull digitalvasic/translator:latest

# Run
docker run -v /path/to/your/books:/books digitalvasic/translator:latest
```

### Verification

```bash
# Verify installation
translator --version

# Should output: translator version X.X.X
```

## Quick Start

### First Translation

1. **Prepare API Key** (for cloud providers):
   ```bash
   export OPENAI_API_KEY="your-openai-key"
   # or
   export DEEPSEEK_API_KEY="your-deepseek-key"
   ```

2. **Translate a File**:
   ```bash
   translator translate book_ru.fb2 --from ru --to sr --provider deepseek
   ```

3. **Check Result**:
   ```bash
   ls -la *.epub
   # You should see: book_ru_sr.epub
   ```

### Basic Commands

```bash
# Help
translator --help

# Version info
translator --version

# Language detection
translator detect text.txt

# List supported formats
translator formats

# Start web interface
translator server --port 8080
```

## Configuration

### Configuration Files

The translator looks for configuration in this order:

1. `./translator.json` (current directory)
2. `~/.translator/translator.json` (user config)
3. `/etc/translator/translator.json` (system config)

### Sample Configuration

Create `translator.json`:

```json
{
  "translation": {
    "default_provider": "deepseek",
    "default_source_lang": "ru",
    "default_target_lang": "sr",
    "chunk_size": 1000,
    "max_concurrent": 5,
    "quality_threshold": 0.8
  },
  "llm_providers": {
    "openai": {
      "api_key": "${OPENAI_API_KEY}",
      "model": "gpt-4",
      "base_url": "https://api.openai.com/v1"
    },
    "deepseek": {
      "api_key": "${DEEPSEEK_API_KEY}",
      "model": "deepseek-chat",
      "base_url": "https://api.deepseek.com"
    },
    "zhipu": {
      "api_key": "${ZHIPU_API_KEY}",
      "model": "glm-4",
      "base_url": "https://open.bigmodel.cn/api/paas/v4"
    }
  },
  "file_processing": {
    "preserve_metadata": true,
    "preserve_images": true,
    "output_format": "same",
    "serbian_script": "cyrillic"
  },
  "logging": {
    "level": "info",
    "file": "translator.log",
    "max_size": "100MB",
    "max_backups": 5
  }
}
```

### Environment Variables

```bash
# API Keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export ZHIPU_API_KEY="..."
export DEEPSEEK_API_KEY="sk-..."
export QWEN_API_KEY="..."

# Database (if using)
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="translator"
export DB_USER="translator"
export DB_PASSWORD="password"

# Redis (if using for caching)
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export REDIS_PASSWORD=""

# Local providers
export OLLAMA_HOST="localhost:11434"
export LLAMACPP_MODEL_PATH="/path/to/model"
```

## CLI Usage

### Basic Commands

#### Translate Files

```bash
# Basic translation
translator translate input.fb2 --from ru --to sr

# Specify provider
translator translate input.epub --from ru --to sr --provider openai --model gpt-4

# Change output name
translator translate input.txt --from ru --to sr --output translated.txt

# Batch translation
translator translate *.fb2 --from ru --to sr --provider deepseek

# Directory translation
translator translate ./books/ --from ru --to sr --output ./translated/
```

#### Language Detection

```bash
# Detect file language
translator detect document.txt

# Detect text snippet
translator detect --text "Это текст на русском языке"

# Multiple files
translator detect *.txt
```

#### Format Conversion

```bash
# Convert without translation
translator convert book.fb2 --to epub --output book.epub

# Convert directory
translator convert ./input/ --to epub --output ./output/
```

### Advanced Options

```bash
# Custom chunking
translator translate large_book.epub --from ru --to sr --chunk-size 2000 --overlap 200

# Quality requirements
translator translate book.fb2 --from ru --to sr --quality-threshold 0.9

# Preserve metadata
translator translate book.epub --from ru --to sr --preserve-metadata --preserve-images

# Serbian script selection
translator translate book.fb2 --from ru --to sr --script latin

# Verbose output
translator translate book.fb2 --from ru --to sr --verbose

# Progress monitoring
translator translate book.fb2 --from ru --to sr --progress
```

### Batch Operations

```bash
# Create batch file
echo "book1.fb2 ru sr" > batch.txt
echo "book2.epub en sr" >> batch.txt
echo "book3.txt auto sr" >> batch.txt

# Run batch
translator batch --file batch.txt --provider deepseek

# Directory batch
translator batch --input ./source/ --output ./translated/ --from ru --to sr

# Parallel processing
translator batch --input ./books/ --parallel 4 --provider deepseek
```

## Web Interface

### Starting the Server

```bash
# Basic server
translator server

# Custom port
translator server --port 8080

# Production with TLS
translator server --tls --cert-file server.crt --key-file server.key

# Custom config
translator server --config production.json
```

### Using the Web Interface

1. **Open Browser**: Navigate to `http://localhost:8080`

2. **Upload File**: Drag and drop or click to select file

3. **Configure Translation**:
   - Source language (auto-detect available)
   - Target language
   - LLM provider
   - Quality settings

4. **Monitor Progress**: Real-time progress bar and status updates

5. **Download Result**: Download translated file when complete

### Web Interface Features

- **Drag & Drop Upload**: Intuitive file upload
- **Live Progress**: Real-time translation progress
- **Batch Upload**: Multiple file translation
- **Preview Mode**: Preview translations before download
- **Quality Scores**: See translation quality metrics
- **History**: Track recent translations
- **Settings**: Configure preferences

## File Formats

### Supported Input Formats

#### FB2 (FictionBook 2.0)
- **Features**: Full metadata support, images, annotations
- **Best for**: Fiction books, literary works
- **Encoding**: UTF-8 required
- **Size limit**: 50MB

#### EPUB 2.0/3.0
- **Features**: Rich formatting, images, CSS styling
- **Best for**: Modern ebooks, complex layouts
- **Encoding**: UTF-8 required
- **Size limit**: 100MB

#### TXT
- **Features**: Plain text, encoding detection
- **Best for**: Simple documents, testing
- **Encoding**: Auto-detected (UTF-8, CP1251, etc.)
- **Size limit**: 10MB

#### HTML
- **Features**: Web content, CSS styling
- **Best for**: Web articles, online content
- **Encoding**: UTF-8 required
- **Size limit**: 20MB

#### PDF
- **Features**: Document formatting, images, OCR
- **Best for**: Professional documents, manuals
- **Encoding**: OCR extraction
- **Size limit**: 100MB

#### DOCX
- **Features**: Word documents, formatting, images
- **Best for**: Office documents, manuscripts
- **Encoding**: Unicode
- **Size limit**: 50MB

### Output Format Options

- **Same as Input**: Preserve original format
- **EPUB**: Standard ebook format
- **PDF**: Print-friendly format
- **TXT**: Plain text
- **HTML**: Web-ready format
- **DOCX**: Word document

### Format Conversion Examples

```bash
# FB2 to EPUB
translator convert book.fb2 --to epub

# EPUB to PDF
translator convert book.epub --to pdf

# Multiple files
translator convert *.fb2 --to epub --output ./epub_books/

# Directory conversion
translator convert ./documents/ --to pdf --output ./pdfs/
```

## Translation Providers

### OpenAI GPT

**Setup**:
```bash
export OPENAI_API_KEY="sk-..."
```

**Models**:
- `gpt-4`: Best quality, higher cost
- `gpt-4-turbo`: Fast, good quality
- `gpt-3.5-turbo`: Fast, lower cost

**Best For**:
- General purpose translation
- Technical documents
- Fast turnaround needed

**Example**:
```bash
translator translate book.fb2 --from ru --to sr --provider openai --model gpt-4
```

### DeepSeek

**Setup**:
```bash
export DEEPSEEK_API_KEY="sk-..."
```

**Models**:
- `deepseek-chat`: General purpose
- `deepseek-coder`: Technical content

**Best For**:
- Large volume translations
- Cost-effective projects
- Consistent terminology

### Zhipu AI (GLM-4)

**Setup**:
```bash
export ZHIPU_API_KEY="..."
```

**Models**:
- `glm-4`: Latest model
- `glm-3-turbo`: Faster variant

**Best For**:
- Russian to Slavic languages
- Literary translations
- Cultural content

### Anthropic Claude

**Setup**:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Models**:
- `claude-3-opus-20240229`: Best quality
- `claude-3-sonnet-20240229`: Balanced
- `claude-3-haiku-20240307`: Fastest

**Best For**:
- Literary works
- Creative content
- Maintaining authorial voice

### Ollama (Local)

**Setup**:
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull model
ollama pull llama2:7b
ollama pull mistral:7b

# Set host
export OLLAMA_HOST="localhost:11434"
```

**Models**:
- `llama2:7b`, `llama2:13b`
- `mistral:7b`
- `codellama:7b`

**Best For**:
- Privacy-sensitive content
- Offline translation
- Cost control

### LlamaCPP

**Setup**:
```bash
# Download model
wget https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.2-GGUF/resolve/main/mistral-7b-instruct-v0.2.Q4_K_M.gguf

# Set path
export LLAMACPP_MODEL_PATH="/path/to/model.gguf"
```

**Best For**:
- Custom model fine-tuning
- Specialized domains
- Maximum control

## Quality Control

### Understanding Quality Scores

**Score Ranges**:
- **0.9-1.0**: Excellent, publication-ready
- **0.8-0.9**: Good, minimal editing needed
- **0.7-0.8**: Acceptable, requires review
- **0.6-0.7**: Needs improvement
- **Below 0.6**: Poor quality

**Quality Metrics**:
- **Accuracy**: Translation correctness
- **Fluency**: Natural language flow
- **Consistency**: Terminology consistency
- **Completeness**: All content translated

### Quality Configuration

```json
{
  "quality": {
    "threshold": 0.8,
    "auto_retry": true,
    "retry_provider": "openai",
    "max_retries": 3,
    "verify_grammar": true,
    "check_consistency": true
  }
}
```

### Manual Review Workflow

```bash
# Translate with quality requirements
translator translate book.fb2 --from ru --to sr --quality-threshold 0.9

# If quality is low, try different provider
translator translate book.fb2 --from ru --to sr --provider openai --quality-threshold 0.9

# Create review template
translator translate book.fb2 --from ru --to sr --template review-template.fb2

# Apply manual edits
translator apply-edits review-template.fb2 --edited-changes.txt
```

### Quality Improvement Tips

1. **Provider Selection**: Use OpenAI for technical content, Claude for literature
2. **Chunk Size**: Smaller chunks for complex content
3. **Overlap**: Use overlap for context preservation
4. **Iterative Translation**: Translate, review, re-translate problematic sections
5. **Custom Prompts**: Use domain-specific prompts for specialized content

## Batch Processing

### Batch File Format

Create a text file with translation tasks:

```
# Format: input_file source_lang target_lang [provider] [options]
book1.fb2 ru sr deepseek --quality-threshold 0.9
book2.epub en sr openai --model gpt-4
article.html auto sr zhipu --script latin
document.pdf auto sr deepseek --preserve-images
```

### Running Batch Operations

```bash
# From file
translator batch --file tasks.txt

# Directory
translator batch --input ./source/ --output ./translated/ --from ru --to sr

# Parallel processing
translator batch --file tasks.txt --parallel 4 --provider deepseek

# With progress monitoring
translator batch --file tasks.txt --progress --verbose
```

### Batch Configuration

```json
{
  "batch": {
    "max_concurrent": 5,
    "fail_fast": false,
    "retry_failed": true,
    "max_retries": 3,
    "output_directory": "./translated/",
    "create_subdirs": true,
    "preserve_structure": true
  }
}
```

### Batch Monitoring

```bash
# Monitor progress
translator batch-status --batch-id batch-uuid-v4

# List all batches
translator batch-list

# Cancel batch
translator batch-cancel --batch-id batch-uuid-v4

# Download results
translator batch-download --batch-id batch-uuid-v4 --output ./results.zip
```

## Distributed Processing

### Architecture Overview

```
Coordinator (Server)
├── Task Queue
├── Load Balancer
└── Progress Monitor

Workers (SSH Nodes)
├── Translation Engine
├── Local Cache
└── Progress Reporter
```

### Setting Up Coordinator

```bash
# Create coordinator config
cat > coordinator.json << EOF
{
  "server": {
    "port": 8443,
    "host": "0.0.0.0"
  },
  "distributed": {
    "mode": "coordinator",
    "worker_timeout": 300,
    "heartbeat_interval": 30
  },
  "database": {
    "type": "postgresql",
    "connection": "postgres://user:pass@localhost/translator"
  }
}
EOF

# Start coordinator
translator-server --config coordinator.json --mode coordinator
```

### Setting Up Workers

```bash
# Create worker config
cat > worker.json << EOF
{
  "server": {
    "port": 8445
  },
  "distributed": {
    "mode": "worker",
    "coordinator": "coordinator.example.com:8443",
    "worker_id": "worker-001",
    "max_concurrent": 4
  },
  "translation": {
    "providers": ["deepseek", "zhipu"]
  }
}
EOF

# Deploy workers
translator-deploy --hosts worker1.example.com,worker2.example.com --config worker.json
```

### SSH Worker Deployment

```bash
# Generate SSH keys
translator ssh-keygen --output ssh_key

# Deploy to multiple workers
translator deploy-ssh \
  --hosts "worker1:22,worker2:22,worker3:22" \
  --username translator \
  --key ssh_key \
  --config worker.json

# Monitor workers
translator worker-status
translator worker-health
```

### Load Balancing

```bash
# Configure load balancing
translator load-balancer --algorithm round-robin --weights "worker1:2,worker2:1,worker3:3"

# Monitor performance
translator monitor --realtime
translator stats --by-worker
```

## Troubleshooting

### Common Issues

#### Installation Problems

**Issue**: `command not found: translator`
```bash
# Solution: Add to PATH
echo 'export PATH=$PATH:/path/to/translator' >> ~/.bashrc
source ~/.bashrc
```

**Issue**: Permission denied
```bash
# Solution: Make executable
chmod +x translator
```

#### Configuration Issues

**Issue**: API key not found
```bash
# Check environment variables
env | grep API_KEY

# Set properly
export OPENAI_API_KEY="your-key"
```

**Issue**: Config file not found
```bash
# Check locations
translator --debug translate test.txt
# Look for config file path in output

# Create config
translator config --init
```

#### Translation Issues

**Issue**: Translation failed
```bash
# Check provider status
translator provider-status

# Try different provider
translator translate book.fb2 --from ru --to sr --provider openai

# Reduce chunk size
translator translate book.fb2 --from ru --to sr --chunk-size 500
```

**Issue**: Poor quality
```bash
# Increase quality threshold
translator translate book.fb2 --from ru --to sr --quality-threshold 0.9

# Use higher-end provider
translator translate book.fb2 --from ru --to sr --provider openai --model gpt-4
```

#### Performance Issues

**Issue**: Slow translation
```bash
# Increase concurrency
translator translate book.fb2 --from ru --to sr --max-concurrent 8

# Use faster provider
translator translate book.fb2 --from ru --to sr --provider deepseek

# Check system resources
translator system-status
```

### Debug Mode

```bash
# Enable debug logging
translator --debug translate book.fb2 --from ru --to sr

# Verbose output
translator --verbose translate book.fb2 --from ru --to sr

# Check logs
tail -f translator.log
```

### Getting Help

```bash
# General help
translator --help

# Command-specific help
translator translate --help

# Version info
translator --version

# System check
translator doctor
```

## Advanced Features

### Custom Prompts

```json
{
  "translation": {
    "custom_prompt": "You are a professional translator specializing in literary works. Translate the following text from {source_lang} to {target_lang}, preserving the author's voice and cultural nuances. Pay special attention to idiomatic expressions and maintain the literary style."
  }
}
```

### Translation Memory

```bash
# Initialize translation memory
translator tm-init --database tm.db

# Add translations
translator tm-add --source "Hello" --target "Привет" --pair en-ru

# Search memory
translator tm-search --source "Hello" --pair en-ru

# Export/Import
translator tm-export --output tm_export.json
translator tm-import --input tm_export.json
```

### Pre-translation Analysis

```bash
# Analyze file before translation
translator analyze book.fb2 --output analysis.json

# View analysis
cat analysis.json
# {
#   "language": "ru",
#   "complexity": "medium",
#   "word_count": 50000,
#   "estimated_time": 1800,
#   "specialized_terms": ["technical", "literary"]
# }

# Preparation phase
translator prepare book.fb2 --analysis analysis.json --output prepared.fb2
```

### Post-translation Polish

```bash
# Polish completed translation
translator polish translated_book.epub --provider openai --quality-target 0.95

# Multi-pass polishing
translator polish translated_book.epub --passes 3 --providers "openai,claude,zhipu"
```

### Script Conversion

```bash
# Serbian Cyrillic to Latin
translator script-convert book_sr_cyrillic.epub --to latin --output book_sr_latin.epub

# Auto-detect and convert
translator script-convert book_sr.epub --auto --output book_sr_converted.epub
```

## Reference

### Command Line Options

#### Global Options
- `--help, -h`: Show help
- `--version, -v`: Show version
- `--config`: Specify config file
- `--verbose, -V`: Verbose output
- `--debug, -d`: Debug mode
- `--quiet, -q`: Quiet mode

#### Translation Options
- `--from, -f`: Source language
- `--to, -t`: Target language
- `--provider, -p`: LLM provider
- `--model, -m`: Model name
- `--output, -o`: Output file/directory
- `--quality-threshold, -q`: Minimum quality (0.0-1.0)
- `--chunk-size, -c`: Text chunk size
- `--overlap, -O`: Overlap between chunks
- `--script, -s`: Script (cyrillic/latin)
- `--preserve-metadata`: Preserve metadata
- `--preserve-images`: Preserve images

#### Server Options
- `--port`: Server port
- `--host`: Server host
- `--tls`: Enable TLS
- `--cert-file`: TLS certificate file
- `--key-file`: TLS private key file
- `--mode`: Server mode (coordinator/worker)

#### Batch Options
- `--file`: Batch file
- `--input`: Input directory
- `--parallel, -P`: Parallel jobs
- `--max-concurrent`: Max concurrent tasks
- `--fail-fast`: Stop on first error
- `--progress`: Show progress

### Language Codes

- `ru`: Russian
- `sr`: Serbian
- `en`: English
- `de`: German
- `fr`: French
- `es`: Spanish
- `it`: Italian
- `pt`: Portuguese
- `auto`: Auto-detect

### Provider Names

- `openai`: OpenAI GPT models
- `anthropic`: Anthropic Claude models
- `zhipu`: Zhipu GLM models
- `deepseek`: DeepSeek models
- `qwen`: Qwen models
- `gemini`: Google Gemini models
- `ollama`: Ollama local models
- `llamacpp`: LlamaCPP local models

### Exit Codes

- `0`: Success
- `1`: General error
- `2`: Configuration error
- `3`: Network error
- `4`: File not found
- `5`: Permission denied
- `6`: Invalid input
- `7`: Translation failed
- `8`: Quality threshold not met
- `9`: Rate limit exceeded
- `10`: Server error

---

For additional support, visit our [documentation](https://docs.translator.digital) or [GitHub repository](https://github.com/digital-vasic/translator).