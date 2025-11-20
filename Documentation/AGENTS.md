# AGENTS.md - FB2 Translation Project

## Build/Test Commands
```bash
# NEW: LLM-powered translation (RECOMMENDED for quality)
python3 llm_fb2_translator.py <input_fb2> [options]
python3 test_llm_translation.py --compare  # Test quality differences

# Main translation tools
python3 fb2_translator.py          # Create translation templates
python3 translation_helper.py      # Manage manual translations
python3 high_quality_fb2_translator.py  # Advanced translation with caching

# Test individual scripts
python3 -m py_compile script.py    # Syntax check
python3 script.py --help          # Check script functionality

# Install dependencies
pip3 install -r requirements.txt
```

## Code Style Guidelines

### Python Standards
- Use Python 3.x with type hints where beneficial
- Import standard library first, then third-party: `xml.etree.ElementTree`, `re`, `sys`, `pathlib`
- Use `pathlib.Path` for file operations
- Handle XML with `xml.etree.ElementTree` - register FB2 namespaces
- NEW: Support OpenAI, Anthropic, and Ollama LLM providers for translation

### Error Handling
- Use try/except blocks for file operations and XML parsing
- Graceful fallback when Google Translate unavailable
- Log translation errors and statistics

### Naming Conventions
- Classes: `PascalCase` (e.g., `TranslationHelper`, `AdvancedFB2Translator`)
- Functions/variables: `snake_case` (e.g., `create_translation_template`, `translation_count`)
- Constants: `UPPER_SNAKE_CASE` (e.g., `TRANSLATE_ENABLED`)

### FB2 Specific
- Always register namespaces: `http://www.gribuser.ru/xml/fictionbook/2.0`
- Preserve XML structure and encoding (UTF-8)
- Update document language to 'sr' for Serbian
- Handle both element text and tail text

### Translation Quality
- NEW: Use LLM (GPT-4, Claude, Ollama) for professional literary translation quality
- API keys should be loaded from environment variables, never hardcoded
- Cache translations to avoid API limits
- Use retry logic for network operations
- Support both Cyrillic and Latin Serbian scripts
- Validate output XML structure
- Advanced prompt engineering for literary context preservation

### Security Guidelines
- NEVER hardcode API keys in source code
- ALWAYS use environment variables for sensitive data
- Use `os.environ.get("API_KEY_NAME")` pattern
- Keep `.env` files in `.gitignore`