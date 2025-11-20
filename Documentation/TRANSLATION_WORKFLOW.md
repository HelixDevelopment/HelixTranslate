# Russian to Serbian FB2 Translation Guide

## Overview

This guide explains how to translate any Russian FB2 book to Serbian Cyrillic using the scripts in this repository.

## Prerequisites

Install required Python packages:
```bash
pip3 install googletrans==4.0.0-rc1 ebooklib weasyprint openai anthropic
```

For PDF conversion, also install system dependencies:
```bash
brew install pango  # macOS
```

### API Keys (Required for AI Translation)

IMPORTANT: Set API keys as environment variables - NEVER hardcode them in code:

```bash
# Create a .env file (ensure it's in .gitignore)
cat > .env << EOF
OPENAI_API_KEY=your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key
ZHIPU_API_KEY=your-zhipu-key
DEEPSEEK_API_KEY=your-deepseek-key
EOF

# Or export them directly in your shell
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"
```

## Translation Methods

### Method 0: AI-Powered Translation (NEW - Highest Quality)

Professional literary translation using LLMs:

```bash
# Using Zhipu AI (cutting edge GLM-4)
export ZHIPU_API_KEY="your-key"
python3 llm_fb2_translator.py mybook_ru.fb2 --provider zhipu

# Using DeepSeek (cost-effective, excellent quality)
export DEEPSEEK_API_KEY="your-key"
python3 llm_fb2_translator.py mybook_ru.fb2 --provider deepseek

# Using OpenAI GPT-4
export OPENAI_API_KEY="your-key"
python3 llm_fb2_translator.py mybook_ru.fb2 --provider openai

# Using Anthropic Claude
export ANTHROPIC_API_KEY="your-key"
python3 llm_fb2_translator.py mybook_ru.fb2 --provider anthropic

# Using local Ollama (free, offline)
python3 llm_fb2_translator.py mybook_ru.fb2 --provider ollama --model llama3:8b
```

### Method 1: Basic Dictionary Translation (Fast)

Fast translation using a predefined dictionary of common Russian to Serbian words:

```bash
python3 simple_fb2_translate.py input_ru.fb2 output_sr.b2
```

**Example:**
```bash
python3 simple_fb2_translate.py Ratibor_1f.b2 Ratibor_1f_sr_basic.b2
```

### Method 2: Template-Based Manual Translation

Create a translation template for manual editing:

```bash
# Create template
python3 fb2_translator.py input_ru.fb2
# Select option 2 for manual template

# Extract text for translation
python3 translation_helper.py input_sr_template.b2
# Select option 1 to create translation list

# Edit the translation_list.txt file with Serbian translations

# Apply translations back to FB2
python3 translation_helper.py input_sr_template.b2
# Select option 2 to apply translations
```

### Method 3: Automatic Google Translate Translation

Direct translation using Google Translate API:

```bash
python3 translate_fb2_direct.py input_ru.fb2 output_sr.b2
```

**Example:**
```bash
python3 translate_fb2_direct.py Ratibor_1f.b2 Ratibor_1f_sr_auto.b2
```

## Format Conversion

### Convert to EPUB

```bash
python3 fb2_to_epub.py input_sr.b2 output_sr.epub
```

**Example:**
```bash
python3 fb2_to_epub.py Ratibor_1f_sr_basic.b2 Ratibor_1f_sr_basic.epub
```

### Convert to PDF

```bash
python3 fb2_to_pdf.py input_sr.b2 output_sr.pdf
```

**Example:**
```bash
python3 fb2_to_pdf.py Ratibor_1f_sr_basic.b2 Ratibor_1f_sr_basic.pdf
```

## Complete Workflow Example

```bash
# 1. Translate Russian FB2 to Serbian (basic method)
python3 simple_fb2_translate.py mybook_ru.fb2 mybook_sr.b2

# 2. Convert to EPUB
python3 fb2_to_epub.py mybook_sr.b2 mybook_sr.epub

# 3. Convert to PDF
python3 fb2_to_pdf.py mybook_sr.b2 mybook_sr.pdf
```

## Script Descriptions

- **simple_fb2_translate.py**: Fast dictionary-based translation
- **fb2_translator.py**: Interactive translation with multiple options
- **translation_helper.py**: Manual translation workflow helper
- **translate_fb2_direct.py**: Direct Google Translate translation
- **fb2_to_epub.py**: Convert FB2 to EPUB format
- **fb2_to_pdf.py**: Convert FB2 to PDF format

## File Naming Convention

All scripts accept input and output files as arguments:

- Input: Any `.fb2` file (Russian original)
- Output: Automatically generated if not specified
  - `*_sr_basic.b2` for basic translation
  - `*_sr_cyrillic.b2` for high-quality translation
  - `*.epub` for EPUB format
  - `*.pdf` for PDF format

## Quality Notes

- **Basic translation**: Fast but limited to dictionary words
- **Manual translation**: Highest quality but time-consuming
- **Automatic translation**: Good quality but may have errors
- **Professional review**: Recommended for final publication

## Git Ignore

All generated files are excluded from version control:
- `*.fb2`, `*.epub`, `*.pdf` (e-book formats)
- `*_sr_*`, `*_translated*`, `*_template*` (translation artifacts)
- `*translation_list*` (translation files)
- Python cache files and logs

## Troubleshooting

1. **Google Translate errors**: Use basic dictionary method instead
2. **PDF conversion issues**: Install Pango library (`brew install pango`)
3. **Encoding issues**: All files use UTF-8 encoding
4. **Missing dependencies**: Install required Python packages

## Language Support

- **Source**: Russian (ru)
- **Target**: Serbian Cyrillic (sr)
- **Scripts**: Supports both Cyrillic and Latin Serbian output