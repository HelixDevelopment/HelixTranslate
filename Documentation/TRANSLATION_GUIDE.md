# FB2 Book Translation Guide - Russian to Serbian

## Files Created

1. **Ratibor_1f_sr_template.b2** - Template with Russian text and placeholders for Serbian
2. **translation_list.txt** - Text-only list of all Russian text to translate
3. **Ratibor_1f_sr_sample.b2** - Sample with some sections translated
4. **fb2_translator.py** - Main translation tool
5. **translation_helper.py** - Helper for managing translations
6. **sample_translation.py** - Script that created the sample

## Translation Process Options

### Option 0: AI-Powered Translation (NEW - Recommended)

Set up API keys as environment variables (NEVER hardcode in code):
```bash
# For Zhipu AI (GLM-4)
export ZHIPU_API_KEY="your-zhipu-api-key"

# For DeepSeek (cost-effective)
export DEEPSEEK_API_KEY="your-deepseek-api-key"

# For OpenAI GPT-4
export OPENAI_API_KEY="your-openai-api-key"

# For Anthropic Claude
export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

Then translate:
```bash
# Using Zhipu AI (cutting edge quality)
python3 llm_fb2_translator.py Ratibor_1f.b2 --provider zhipu

# Using DeepSeek (cost-effective, excellent quality)
python3 llm_fb2_translator.py Ratibor_1f.b2 --provider deepseek

# Using OpenAI GPT-4
python3 llm_fb2_translator.py Ratibor_1f.b2 --provider openai

# Using local Ollama (free)
python3 llm_fb2_translator.py Ratibor_1f.b2 --provider ollama --model llama3:8b
```

### Option 1: Manual Translation (Recommended for Best Quality)

1. Open `translation_list.txt` in a text editor
2. For each entry:
   - Find the Russian text (RU: ...)
   - Add Serbian translation after SR:
   - Example:
     ```
     RU: Отзвуки
     SR: Одјеци
     ```

3. After completing all translations:
   ```bash
   python3 translation_helper.py
   # Select option 2 to apply translations
   ```

4. The final translated file will be: `Ratibor_1f_sr_translated.b2`

### Option 2: Use Professional Translation Service

1. Send `translation_list.txt` to a professional Russian-Serbian translator
2. Receive the completed file
3. Apply translations using `translation_helper.py` option 2

### Option 3: Semi-Automatic Translation

1. Use an online translator to get initial translations
2. Review and edit for quality and cultural context
3. Apply corrections manually

## Important Notes for Serbian Translation

### Script Choice
- **Cyrillic** (ћирилица) - Official script, more traditional
- **Latin** (латиница) - Also widely used in Serbia

### Cultural Adaptation
- Preserve names (Ратибор → Ратибор)
- Adapt cultural references where appropriate
- Maintain the tone and style of the original

### Formatting
- Keep XML structure intact
- Maintain paragraph breaks
- Preserve emphasis and formatting

## Sample Translations Provided

The sample file (`Ratibor_1f_sr_sample.b2`) includes translations for:
- Title: "Отзвуки" → "Одјеци"
- Author name preserved: "Ратибор" → "Ратибор"
- First paragraph of the story

## Testing the Translation

1. Open the final `.b2` file in an FB2 reader
2. Check for:
   - Text display correctly
   - Formatting preserved
   - No broken XML structure
3. Read through to verify flow and readability

## Tools and Scripts

### fb2_translator.py
- Creates translation templates
- Attempts automatic translation (with fallback to manual)

### translation_helper.py
- Manages the translation process
- Creates editable translation lists
- Applies completed translations back to FB2 format

### Sample Translation Script
- Demonstrates how translations should look
- Provides reference for style and formatting

## Recommendations for High-Quality Translation

1. **Literary Style**: This is a fiction book, maintain literary style
2. **Character Voice**: Preserve character voices and personalities
3. **Cultural Nuances**: Adapt or explain Russian cultural references
4. **Technical Terms**: Translate technical terms appropriately
5. **Dialogue**: Ensure dialogue sounds natural in Serbian

## Final Steps

1. Complete all translations in `translation_list.txt`
2. Apply translations using `translation_helper.py`
3. Review the final FB2 file
4. Test in multiple FB2 readers
5. Consider having a native Serbian speaker review

## Alternative Approach

If you prefer a different approach, you can:
1. Extract all Russian text to a plain text file
2. Translate using your preferred method
3. Manually insert translations into the FB2 template
4. Validate XML structure before finalizing

## Support Files

All scripts include error handling and will guide you through the process.
If you encounter any issues, check:
- File permissions
- XML structure integrity
- Proper UTF-8 encoding