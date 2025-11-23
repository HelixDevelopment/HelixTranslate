# Universal Multi-Format Multi-Language Translation Examples

This document demonstrates the universal capabilities of the translation system across different formats and languages.

## Format Support Examples

### Input Formats (Auto-detected)
- **FB2** (FictionBook2) - `book.fb2`
- **EPUB** - `book.epub` 
- **TXT** - `book.txt`
- **HTML** - `book.html`
- **PDF** - `book.pdf`
- **DOCX** - `book.docx`

### Output Formats
- **EPUB** (default) - Standard ebook format
- **TXT** - Plain text
- **FB2** - FictionBook2 format
- **HTML** - Web format

## Language Translation Examples

### European Languages
```bash
# English to German
./translator -input book.epub -locale de

# French to Spanish
./translator -input livre.epub -source french -target spanish

# Italian to Portuguese
./translator -input libro.epub -source it -target pt
```

### Asian Languages
```bash
# English to Chinese
./translator -input book.epub -locale zh

# Japanese to Korean
./translator -input book.epub -source ja -target ko

# English to Arabic
./translator -input book.epub -locale ar -script arabic
```

### Slavic Languages
```bash
# Russian to Serbian
./translator -input book.fb2 -locale sr

# Polish to Czech
./translator -input book.epub -source pl -target cs

# Ukrainian to Bulgarian
./translator -input book.epub -source uk -target bg
```

### Multi-script Support
```bash
# Serbian Cyrillic to Latin
./translator -input book.epub -locale sr -script latin

# Arabic to Latin transliteration
./translator -input book.epub -source ar -target en -script latin

# Chinese to Pinyin
./translator -input book.epub -source zh -target en -script pinyin
```

## Provider Examples

### AI Translation Services
```bash
# OpenAI GPT-4 (high quality)
export OPENAI_API_KEY="your-key"
./translator -input book.epub -locale de -provider openai -model gpt-4

# Anthropic Claude (literary quality)
export ANTHROPIC_API_KEY="your-key"
./translator -input book.epub -locale fr -provider anthropic -model claude-3-sonnet

# DeepSeek (cost-effective)
export DEEPSEEK_API_KEY="your-key"
./translator -input book.epub -locale es -provider deepseek

# Zhipu AI (GLM-4)
export ZHIPU_API_KEY="your-key"
./translator -input book.epub -locale ja -provider zhipu
```

### Local Translation
```bash
# Ollama (free, local)
./translator -input book.epub -locale de -provider ollama -model llama3:8b

# llama.cpp (GPU accelerated)
./translator -input book.epub -locale fr -provider llamacpp

# Dictionary (fast, offline)
./translator -input book.epub -locale es -provider dictionary
```

### Distributed Translation
```bash
# Multi-LLM coordination with fallback
./translator -input book.epub -locale de -provider distributed -config config.distributed.json
```

## Advanced Features

### Auto-detection
```bash
# Auto-detect source language
./translator -input mystery_book.epub -detect

# Auto-detect format and language
./translator -input unknown_file -locale es
```

### Custom Scripts
```bash
# Cyrillic script
./translator -input book.epub -locale ru -script cyrillic

# Latin script
./translator -input book.epub -locale sr -script latin

# Arabic script
./translator -input book.epub -locale ar -script arabic

# Default script (language-specific)
./translator -input book.epub -locale ja -script default
```

### Batch Processing
```bash
# Translate entire directory
./translator -input ./books/ -output ./translated/ -locale es -recursive

# Parallel processing
./translator -input ./books/ -locale de -parallel -max-concurrency 4
```

## Quality Settings

### High Quality (Recommended)
```bash
./translator -input book.epub -locale de -provider openai -model gpt-4 -quality high
```

### Balanced Quality
```bash
./translator -input book.epub -locale es -provider deepseek -quality medium
```

### Fast Translation
```bash
./translator -input book.epub -locale fr -provider dictionary -quality fast
```

## Specialized Translation

### Technical Documents
```bash
./translator -input manual.epub -locale de -domain technical -provider openai
```

### Literary Works
```bash
./translator -input novel.epub -locale fr -domain literary -provider anthropic -style preserve
```

### Legal Documents
```bash
./translator -input contract.epub -locale es -domain legal -provider openai -precision high
```

## Configuration Examples

### Multi-Provider Setup
```json
{
  "providers": [
    {"name": "openai", "weight": 0.4, "model": "gpt-4"},
    {"name": "anthropic", "weight": 0.3, "model": "claude-3-sonnet"},
    {"name": "deepseek", "weight": 0.3, "model": "deepseek-chat"}
  ],
  "fallback_strategy": "sequential",
  "quality_threshold": 0.8
}
```

### Language-Specific Settings
```json
{
  "languages": {
    "de": {
      "formality": "formal",
      "regional_variant": "DE"
    },
    "fr": {
      "formality": "standard",
      "regional_variant": "FR"
    },
    "pt": {
      "formality": "standard",
      "regional_variant": "BR"
    }
  }
}
```

## Performance Optimization

### GPU Acceleration
```bash
# Enable GPU for local models
./translator -input book.epub -locale de -provider llamacpp -gpu-layers 35
```

### Memory Management
```bash
# Limit memory usage
./translator -input book.epub -locale es -provider ollama -max-memory 8G
```

### Caching
```bash
# Enable translation cache
./translator -input book.epub -locale fr -cache enabled -cache-ttl 24h
```

This demonstrates the truly universal nature of the translation system - any format, any language, any direction!