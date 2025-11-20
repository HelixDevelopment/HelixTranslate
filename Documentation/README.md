# FB2 Translator - Russian to Serbian E-book Translation

A comprehensive toolkit for translating Russian FB2 (FictionBook2) e-books to Serbian, supporting multiple translation methods from basic dictionary replacement to advanced AI-powered translation.

## Features

- **Multiple Translation Methods**
  - AI-powered translation (GPT-4, Claude, Zhipu AI, DeepSeek, Ollama)
  - Google Translate API integration
  - Dictionary-based translation
  - Manual template-based translation

- **Format Support**
  - FB2 (FictionBook2) input/output
  - EPUB conversion
  - PDF conversion
  - UTF-8 encoding support

- **Language Options**
  - Russian to Serbian translation
  - Both Cyrillic and Latin Serbian scripts
  - Context-aware literary translation

- **Advanced Features**
  - Translation caching
  - Error handling and retry logic
  - Translation statistics
  - Batch processing support

## Quick Start

### Installation

1. Clone this repository
2. Install dependencies:
```bash
pip3 install -r requirements.txt
```
3. Set up API keys as environment variables (NEVER hardcode them):
```bash
# Create a .env file (add to .gitignore)
echo "OPENAI_API_KEY=your-openai-key" >> .env
echo "ANTHROPIC_API_KEY=your-anthropic-key" >> .env
echo "ZHIPU_API_KEY=your-zhipu-key" >> .env
echo "DEEPSEEK_API_KEY=your-deepseek-key" >> .env

# Or export them directly
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"
```

### Basic Usage

**AI-Powered Translation (Recommended):**
```bash
# Using OpenAI GPT-4
export OPENAI_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2

# Using Zhipu AI (GLM-4)
export ZHIPU_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider zhipu

# Using DeepSeek (cost-effective)
export DEEPSEEK_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider deepseek

# Using Anthropic Claude
export ANTHROPIC_API_KEY="your-key"
python3 llm_fb2_translator.py book_ru.fb2 --provider anthropic

# Using local Ollama (free)
python3 llm_fb2_translator.py book_ru.fb2 --provider ollama --model llama3:8b
```

**Dictionary Translation (Fast, No API):**
```bash
python3 simple_fb2_translate.py input_ru.fb2 output_sr.b2
```

**Google Translate Translation:**
```bash
python3 high_quality_fb2_translator.py input_ru.fb2 output_sr.b2
```

## Translation Methods Comparison

| Method | Quality | Cost | Speed | Setup |
|--------|--------|-------|-------|-------|
| AI (GPT-4/Claude/Zhipu) | ★★★★★ | $$ | Medium | API Key |
| DeepSeek | ★★★★★ | $ | Medium | API Key |
| Ollama (Local) | ★★★★☆ | Free | Slow | Local Setup |
| Google Translate | ★★★☆☆ | $ | Fast | API Key |
| Dictionary | ★★☆☆☆ | Free | Very Fast | None |

## Configuration

### AI Translation Setup

**IMPORTANT**: Never hardcode API keys in configuration files. Use environment variables.

Create a configuration file:
```bash
python3 llm_fb2_translator.py --create-config my_config.json
```

Example configuration (NO API KEY):
```json
{
  "provider": "openai",
  "model": "gpt-4",
  "target_script": "cyrillic",
  "cache_translations": true,
  "max_tokens": 4000
}
```

API keys are loaded from environment variables:
```bash
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"
```

## Format Conversion

**FB2 to EPUB:**
```bash
python3 fb2_to_epub.py input_sr.b2 output_sr.epub
```

**FB2 to PDF:**
```bash
python3 fb2_to_pdf.py input_sr.b2 output_sr.pdf
```

## Manual Translation Workflow

For complete control over translation quality:

1. **Create Template:**
```bash
python3 fb2_translator.py input_ru.fb2  # Select option 2
```

2. **Extract Translation List:**
```bash
python3 translation_helper.py input_sr_template.b2  # Select option 1
```

3. **Edit `translation_list.txt`** with your translations

4. **Apply Translations:**
```bash
python3 translation_helper.py input_sr_template.b2  # Select option 2
```

## Project Structure

```
├── llm_fb2_translator.py          # AI-powered translation (recommended)
├── high_quality_fb2_translator.py # Google Translate with caching
├── simple_fb2_translate.py        # Dictionary-based translation
├── fb2_translator.py              # Template creation and manual workflow
├── translation_helper.py          # Manual translation management
├── fb2_to_epub.py                 # FB2 to EPUB conversion
├── fb2_to_pdf.py                  # FB2 to PDF conversion
├── requirements.txt               # Python dependencies
└── configs/                       # Example configuration files
```

## Requirements

- Python 3.7+
- Required packages (see requirements.txt)
- Optional: API keys for AI translation services
- Optional: Ollama for local AI translation
- Optional: pango for PDF conversion (macOS: `brew install pango`)

## Translation Quality

The AI-powered translation provides:
- **Contextual Understanding**: Maintains narrative context and literary style
- **Cultural Adaptation**: Handles idioms, metaphors, and cultural references
- **Consistent Voice**: Preserves character voice and author's style
- **Serbian Nuances**: Proper grammar, vocabulary, and expressions
- **Complex Sentences**: Natural Serbian sentence structures

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
1. Check the documentation in the project files
2. Review the example configurations
3. Test with a small FB2 file first
4. Check API key configuration for translation services

## Acknowledgments

- FB2 format specification by Gribuser
- OpenAI, Anthropic, Zhipu AI, and DeepSeek for AI translation services
- Google Translate API
- Ollama for local AI models