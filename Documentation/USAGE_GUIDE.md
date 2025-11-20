# Universal Ebook Translator - Quick Usage Guide

## 🚀 Fastest Way to Translate

### Option 1: Single Provider (Recommended for Now)

Due to network issues with Chinese endpoints (Qwen, Zhipu), use DeepSeek directly:

```bash
# Set API key
export DEEPSEEK_API_KEY="your-deepseek-api-key"

# Use the convenience script
./translate_with_deepseek.sh Books/your_book.epub

# Or manually
./build/translator -input Books/your_book.epub -provider deepseek -locale sr
```

**Benefits:**
- ✅ No network overhead
- ✅ Reliable connection
- ✅ 180s timeout prevents errors
- ✅ Fast and efficient

### Option 2: Multi-LLM (When Network Issues Resolved)

Once Qwen/Zhipu connectivity is fixed, or when using international providers:

```bash
# Add multiple providers
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export DEEPSEEK_API_KEY="your-deepseek-key"

# Run multi-LLM
./build/translator -input Books/your_book.epub -provider multi-llm -locale sr
```

**Benefits:**
- ✅ 9 instances (3+3+3)
- ✅ Automatic retry across providers
- ✅ Better load distribution
- ✅ Redundancy

---

## 📋 Common Commands

### Basic Translation
```bash
# Russian → Serbian (Cyrillic)
./build/translator -input book.epub -locale sr -provider deepseek

# With custom output
./build/translator -input book.epub -output translated.epub -provider deepseek -locale sr

# Latin script
./build/translator -input book.epub -provider deepseek -locale sr -script latin
```

### Language Detection
```bash
# Detect source language
./build/translator -input book.epub --detect
```

### Output Formats
```bash
# EPUB (default)
./build/translator -input book.epub -format epub -provider deepseek -locale sr

# Plain text
./build/translator -input book.epub -format txt -provider deepseek -locale sr
```

### Different Target Languages
```bash
# Serbian
./build/translator -input book.epub -locale sr -provider deepseek

# German
./build/translator -input book.epub -locale de -provider deepseek

# French
./build/translator -input book.epub -locale fr -provider deepseek
```

---

## 🔧 Configuration

### Environment Variables

**DeepSeek (Recommended):**
```bash
export DEEPSEEK_API_KEY="your-key"
```

**Other Providers:**
```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export ZHIPU_API_KEY="your-key"
export QWEN_API_KEY="your-key"
```

**Qwen OAuth:**
- Automatically detected from `~/.qwen/oauth_creds.json`
- No environment variable needed
- Medium priority (2 instances)

---

## 📊 Monitor Translation Progress

### Real-time Monitoring
```bash
# Check status
./monitor_translation.sh

# Watch continuously (updates every 5 seconds)
watch -n 5 ./monitor_translation.sh

# View log file
tail -f /tmp/translation_v3.log
```

### Check Output
```bash
# List translated files
ls -lh Books/*_sr*.epub

# Verify file type
file Books/translated_book.epub

# Check for untranslated Russian text
unzip -p Books/translated.epub | grep -o '[А-Яа-я]\{20,\}' | head -10
```

---

## 🐛 Troubleshooting

### "Context deadline exceeded"
**Problem:** HTTP timeout too short

**Solution:** Already fixed! Timeouts increased to 180s
```bash
# Rebuild if using old version
make build
```

### "No LLM instances available"
**Problem:** No API keys or OAuth credentials found

**Solution:** Set at least one API key
```bash
export DEEPSEEK_API_KEY="your-key"
```

### "TLS handshake timeout" (Qwen/Zhipu)
**Problem:** Network connectivity issues to Chinese servers

**Solution:** Use DeepSeek or international providers
```bash
# Use DeepSeek directly
./translate_with_deepseek.sh Books/book.epub

# Or add international providers
export OPENAI_API_KEY="your-key"
```

### "Translation failed after X attempts"
**Problem:** All providers failed

**Solution:** Check API keys and network
```bash
# Verify API key is set
echo $DEEPSEEK_API_KEY

# Test with simple file
./build/translator -input small_book.epub -provider deepseek -locale sr
```

### Output file not created
**Problem:** Too many translation failures

**Solution:** Check logs for errors
```bash
# View recent errors
grep -i "error\|fail" /tmp/translation_v3.log | tail -20

# Try with smaller file
./build/translator -input small_sample.epub -provider deepseek -locale sr
```

---

## 🎯 Best Practices

### 1. Start with DeepSeek
```bash
export DEEPSEEK_API_KEY="your-key"
./translate_with_deepseek.sh Books/book.epub
```
- Most reliable currently
- No network issues
- Good quality
- Cost-effective

### 2. Use Multi-LLM for Large Projects
```bash
# When connectivity is good
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export DEEPSEEK_API_KEY="your-key"

./build/translator -input large_book.epub -provider multi-llm -locale sr
```
- Better parallelization
- Automatic retry
- Load distribution

### 3. Monitor Long Translations
```bash
# Start translation in background
nohup ./build/translator -input book.epub -provider deepseek -locale sr > translation.log 2>&1 &

# Monitor progress
watch -n 10 "tail -20 translation.log | grep -E 'chapter|success|error'"
```

### 4. Verify Translation Quality
```bash
# Check for Russian text (indicates untranslated sections)
unzip -p output.epub | grep -o '[А-Яа-я]\{20,\}' | head -20

# If Russian text found, sections may be untranslated
```

---

## 📈 Performance Tips

### Optimize for Speed
1. **Use DeepSeek** - Fastest currently
2. **Increase parallelization** - Add more providers
3. **Monitor progress** - Catch issues early

### Optimize for Quality
1. **Use GPT-4 or Claude** - Best quality
2. **Enable multi-LLM** - Multiple providers validate
3. **Review output** - Check for errors

### Optimize for Cost
1. **Use DeepSeek** - Most cost-effective
2. **Cache results** - Reuse translations
3. **Test with samples** - Before full translation

---

## 🔐 Security Notes

### API Keys
- Never commit API keys to git
- Use environment variables only
- Keep keys secure

### OAuth Credentials
- Stored in `~/.qwen/oauth_creds.json` or `~/.translator/qwen_credentials.json`
- File permissions: 0600 (owner read/write only)
- Never versioned (excluded in .gitignore)

### Credential Priority
1. API key from environment (highest)
2. OAuth from ~/.translator/
3. OAuth from ~/.qwen/

---

## 📚 Examples

### Example 1: Simple Translation
```bash
export DEEPSEEK_API_KEY="your-key"
./build/translator -input Books/russian_book.epub -locale sr -provider deepseek
```

### Example 2: Custom Output with Latin Script
```bash
./build/translator \
  -input Books/book.epub \
  -output Books/serbian_latin.epub \
  -provider deepseek \
  -locale sr \
  -script latin
```

### Example 3: Multi-LLM Translation
```bash
export OPENAI_API_KEY="key1"
export DEEPSEEK_API_KEY="key2"
./build/translator -input Books/book.epub -provider multi-llm -locale sr
```

### Example 4: Translate to German
```bash
./build/translator -input Books/russian_book.epub -locale de -provider deepseek
```

### Example 5: Plain Text Output
```bash
./build/translator \
  -input Books/book.epub \
  -format txt \
  -output translated.txt \
  -provider deepseek \
  -locale sr
```

---

## 🎓 Advanced Usage

### Custom Model Selection
```bash
# Specify model
export DEEPSEEK_MODEL="deepseek-chat"
./build/translator -input book.epub -provider deepseek -locale sr

# For OpenAI
export OPENAI_MODEL="gpt-4-turbo"
```

### Custom Base URL
```bash
# For custom endpoints
export OPENAI_BASE_URL="https://custom.api.com/v1"
./build/translator -input book.epub -provider openai -locale sr
```

### Source Language Detection
```bash
# Auto-detect source (default)
./build/translator -input book.epub -locale sr -provider deepseek

# Specify source explicitly
./build/translator -input book.epub -source ru -locale sr -provider deepseek
```

---

## 🆘 Getting Help

### Show Version
```bash
./build/translator --version
```

### Show Help
```bash
./build/translator --help
```

### List Supported Languages
```bash
./build/translator --help | grep -A 20 "Supported Languages"
```

### Check Configuration
```bash
# Verify API keys are set
env | grep -E "(OPENAI|ANTHROPIC|DEEPSEEK|ZHIPU|QWEN)_API_KEY"

# Check OAuth credentials
ls -la ~/.qwen/oauth_creds.json ~/.translator/qwen_credentials.json 2>/dev/null
```

---

## 📞 Quick Reference

**Fastest translation:**
```bash
./translate_with_deepseek.sh Books/book.epub
```

**Monitor progress:**
```bash
./monitor_translation.sh
```

**Check output:**
```bash
ls -lh Books/*_sr*.epub
```

**Verify quality:**
```bash
unzip -p Books/output.epub | grep -o '[А-Яа-я]\{20,\}' | head -10
```

**View logs:**
```bash
tail -f /tmp/translation_v3.log
```

---

**For detailed documentation, see:**
- `QWEN_INTEGRATION_SUMMARY.md` - Qwen setup and OAuth
- `PRIORITY_SYSTEM.md` - Multi-LLM prioritization
- `SESSION_SUMMARY.md` - Technical implementation details
- `FINAL_SUMMARY.md` - Executive summary

**Ready to translate!** 🚀
