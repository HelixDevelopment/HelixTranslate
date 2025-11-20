# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Russian to Serbian FB2 (FictionBook2) e-book translation toolkit. The project provides multiple translation methods ranging from simple dictionary replacement to advanced Google Translate API integration, along with format conversion tools for EPUB and PDF output.

## Prerequisites

Install required Python packages:
```bash
pip3 install -r requirements.txt
```

### API Keys (Required)

For security, API keys should be set as environment variables rather than hardcoded:

```bash
# For Zhipu AI (GLM-4)
export ZHIPU_API_KEY="your-zhipu-api-key"

# For DeepSeek
export DEEPSEEK_API_KEY="your-deepseek-api-key"

# For OpenAI GPT
export OPENAI_API_KEY="your-openai-api-key"

# For Anthropic Claude
export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

Or create a `.env` file (ensure it's in `.gitignore`):
```
ZHIPU_API_KEY=your-zhipu-api-key
DEEPSEEK_API_KEY=your-deepseek-api-key
OPENAI_API_KEY=your-openai-api-key
ANTHROPIC_API_KEY=your-anthropic-api-key
```

For specific LLM providers:
```bash
# OpenAI GPT
pip3 install openai>=1.0.0

# Anthropic Claude  
pip3 install anthropic>=0.25.0

# Local Ollama (no pip install needed)
# Install Ollama from: https://ollama.ai/
# Pull model: ollama pull llama3:8b
```

For PDF conversion on macOS:
```bash
brew install pango
```

## Common Commands

### Translation Methods

**NEW: Advanced LLM Translation (Highest Quality):**
```bash
# Using OpenAI GPT-4
export OPENAI_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2

# Using Zhipu AI (cutting edge GLM-4)
export ZHIPU_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider zhipu

# Using DeepSeek (cost-effective, excellent quality)
export DEEPSEEK_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider deepseek

# Using Anthropic Claude
export ANTHROPIC_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider anthropic

# Using local Ollama (free, offline)
python3 llm_fb2_translator.py book_ru.fb2 --provider ollama --model llama3:8b

# Create config template
python3 llm_fb2_translator.py --create-config my_config.json
```

**Basic dictionary translation (fastest):**
```bash
python3 simple_fb2_translate.py <input_ru.fb2> [output_sr.b2]
# Example: python3 simple_fb2_translate.py Ratibor_1f.b2 Ratibor_1f_sr_basic.b2
```

**Advanced translation with Google Translate:**
```bash
python3 high_quality_fb2_translator.py <input_ru.fb2> [output_sr.b2]
# Example: python3 high_quality_fb2_translator.py Ratibor_1f.b2 Ratibor_1f_sr_cyrillic.b2
```

**Direct Google Translate:**
```bash
python3 translate_fb2_direct.py <input_ru.fb2> <output_sr.b2>
```

**Template-based manual translation workflow:**
```bash
# 1. Create template
python3 fb2_translator.py <input_ru.fb2>  # Select option 2

# 2. Extract translation list
python3 translation_helper.py <input_sr_template.b2>  # Select option 1

# 3. Edit translation_list.txt manually

# 4. Apply translations
python3 translation_helper.py <input_sr_template.b2>  # Select option 2
```

**Auto-translate empty entries in translation list:**
```bash
python3 auto_translate_list.py translation_list.txt [output_file.txt]
```

### Format Conversion

**Convert FB2 to EPUB:**
```bash
python3 fb2_to_epub.py <input_sr.b2> <output_sr.epub>
```

**Convert FB2 to PDF:**
```bash
python3 fb2_to_pdf.py <input_sr.b2> <output_sr.pdf>
```

### Testing

**Syntax check:**
```bash
python3 -m py_compile <script.py>
```

**Test script functionality:**
```bash
python3 <script.py> --help
```

## Architecture Overview

### FB2 XML Structure Handling

All scripts use `xml.etree.ElementTree` for XML parsing and maintain the FB2 namespace structure:
- Primary namespace: `http://www.gribuser.ru/xml/fictionbook/2.0`
- XLink namespace: `http://www.w3.org/1999/xlink`

**Namespace registration pattern:**
```python
ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
ET.register_namespace('l', "http://www.w3.org/1999/xlink")
```

### Translation Approaches

**1. LLM-Powered Translation (`llm_fb2_translator.py`) - NEW & RECOMMENDED**
- **Providers**: OpenAI GPT-4, **Zhipu AI (GLM-4)**, DeepSeek, Anthropic Claude, local Ollama
- **Quality**: Professional literary translation quality
- **Features**:
  - Context-aware deep language understanding
  - Cultural nuance preservation
  - Literary style matching
  - Consistent character voice
  - Advanced prompt engineering for translation
  - Translation caching and statistics
  - Support for both Cyrillic and Latin Serbian scripts
- **API**: OpenAI/Anthropic/Zhipu APIs or local Ollama (free)
- **Cost-effectiveness**: Multiple options from free to premium

**2. Simple Dictionary Replacement (`simple_fb2_translate.py`)**
- Uses predefined Russian-Serbian word pairs
- Fast, no API dependencies
- Limited to dictionary terms
- Good for quick, basic translations

**3. Advanced Translation (`high_quality_fb2_translator.py`)**
- Context-aware Google Translate API integration
- Translation caching to avoid API limits
- Retry logic for network failures
- Post-processing for quality improvements
- Reference translations for common terms
- Translation statistics tracking

**4. Template-based Manual Translation**
- Creates FB2 template with `[TRANSLATE: text]` markers
- Extracts text to `translation_list.txt` for manual editing
- Applies completed translations back to FB2

### Translation Quality Comparison

| Method | Quality | Cost | Status |
|--------|--------|-------|---------|
| **LLM (GPT-4/Claude/Zhipu/DeepSeek)** | ★★★★★ | $$-$ | Excellent |
| **Google Translate** | ★★★☆☆ | $ | Limited |
| **Dictionary** | ★★☆☆☆ | Free | None |
| **Manual** | ★★★★★ | $$$$ | Perfect |

**Why LLM Translation is Superior:**
- **Contextual Understanding**: LLMs understand narrative context, not just individual words
- **Literary Voice**: Maintains consistent tone and character voice throughout the work
- **Cultural Adaptation**: Handles idioms, metaphors, and cultural references appropriately
- **Serbian Nuances**: Properly uses Serbian-specific expressions, grammar, and vocabulary
- **Style Matching**: Adapts to the original author's literary style
- **Complex Sentences**: Translates complex Russian sentence structures into natural Serbian

### Element Processing Pattern

All translators recursively process FB2 elements handling:
1. **Element text** - Main text content
2. **Child elements** - Nested structure (paragraphs, emphasis, etc.)
3. **Tail text** - Text after closing tag but before next element

Example pattern from codebase:
```python
def process_element(element):
    # 1. Process element.text
    if element.text:
        element.text = translate(element.text)

    # 2. Process children recursively
    for child in element:
        process_element(child)

    # 3. Process element.tail
    if element.tail:
        element.tail = translate(element.tail)
```

### Metadata Updates

Translated documents update:
- Language field: Changed from 'ru' to 'sr'
- Book title: Translated when available
- Document structure: Preserved intact

### Translation Cache System

The advanced translator (`high_quality_fb2_translator.py`) implements:
- **Translation cache**: Dictionary keyed by `(text, context)` tuples
- **Reference translations**: Pre-defined high-quality translations for common terms
- **Statistics tracking**: Counts for total/translated/cached/error entries

## Code Style Requirements

### Python Standards
- Use Python 3.x with type hints where beneficial
- Import order: Standard library first, then third-party
- Use `pathlib.Path` for file operations
- Handle XML with `xml.etree.ElementTree`

### Naming Conventions
- Classes: `PascalCase` (e.g., `AdvancedFB2Translator`, `FB2Translator`)
- Functions/variables: `snake_case` (e.g., `translate_text`, `translation_count`)
- Constants: `UPPER_SNAKE_CASE` (e.g., `TRANSLATE_ENABLED`)

### Error Handling
- Use try/except blocks for file operations and XML parsing
- Graceful fallback when Google Translate unavailable
- Log translation errors and statistics
- Provide helpful error messages with context

### FB2-Specific Requirements
- Always register FB2 namespaces before parsing
- Preserve XML structure and UTF-8 encoding
- Update document language metadata to 'sr' for Serbian
- Handle both element text and tail text in translations
- Validate output XML structure integrity

### Translation Quality Guidelines
- Cache translations to avoid API rate limits
- Use retry logic (typically 3 attempts) for network operations
- Support both Cyrillic and Latin Serbian scripts
- Post-process translations to fix common issues (quotes, punctuation, capitalization)
- Preserve formatting (newlines, emphasis, whitespace)

## File Naming Conventions

### Input/Output Patterns
- Input: `*.fb2` (Russian original)
- Output suffixes:
  - `*_sr_basic.b2` - Basic dictionary translation
  - `*_sr_cyrillic.b2` - High-quality Cyrillic translation
  - `*_sr_latin.b2` - High-quality Latin translation
  - `*_sr_template.b2` - Manual translation template
  - `*.epub` - EPUB format
  - `*.pdf` - PDF format

### Git Ignore
All generated files are excluded from version control (see `.gitignore`):
- E-book formats: `*.fb2`, `*.b2`, `*.epub`, `*.pdf`
- Translation artifacts: `*_sr_*`, `*_translated*`, `*_template*`, `*translation_list*`
- Python cache: `*.pyc`, `__pycache__/`

## Language Support Details

- **Source language**: Russian (ru)
- **Target language**: Serbian Cyrillic (sr)
- **Script options**: Both Cyrillic (ћирилица) and Latin (латиница)
- **Cyrillic-Latin mapping**: Implemented in `high_quality_translate.py` and `high_quality_fb2_translator.py`

### Serbian Script Conversion
The advanced translator includes complete Cyrillic-to-Latin character mapping for Serbian, allowing output in either script based on user preference.

## Critical Implementation Notes

1. **XML Preservation**: FB2 format is strict XML - maintain structure integrity at all costs
2. **Encoding**: Always use UTF-8 encoding for all file operations
3. **API Rate Limiting**: Google Translate has rate limits - implement delays between requests
4. **Graceful Degradation**: Scripts should provide alternatives when Google Translate unavailable
5. **Context Awareness**: Advanced translator uses element context to improve translation quality
6. **Quality Assurance**: All translations should preserve formatting, punctuation, and paragraph structure
