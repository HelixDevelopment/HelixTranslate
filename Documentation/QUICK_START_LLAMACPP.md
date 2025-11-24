# Quick Start Guide: llama.cpp Translation

## TL;DR - Critical Information

⚠️ **IMPORTANT**: Your system (M3 Pro, 18GB RAM) can safely run **ONLY 1 LLM instance at a time**

Running 2+ instances = System freeze! ❌

## Current Status

✅ **Test translation is running**: "Сон над бездной" → Serbian Cyrillic
- Progress: Processing chapters (38 total)
- Speed: ~27 tokens/second
- Model: Qwen 2.5 7B Instruct (Q4)
- Log file: `translation_llamacpp.log`

## Quick Commands

### Monitor Current Translation
```bash
# Watch progress in real-time
./monitor_llamacpp_translation.sh

# Or just check the log
tail -f translation_llamacpp.log
```

### Translate Multiple Books (Safe!)
```bash
# This script processes books sequentially to prevent freeze
./batch_translate_llamacpp.sh Books/ Books/Translated_Llamacpp/
```

### Translate Single Book
```bash
./cli -input book_ru.epub \
      -output book_sr.epub \
      -locale sr \
      -script cyrillic \
      -provider llamacpp
```

## Safety Rules

1. ✅ **DO**: Use `batch_translate_llamacpp.sh` for multiple books
2. ✅ **DO**: Wait for one translation to finish before starting another
3. ❌ **DON'T**: Run multiple `./cli -provider llamacpp` commands in parallel
4. ❌ **DON'T**: Try to speed up by running 2+ instances

## Why Only 1 Instance?

- Each LLM instance uses: **6-10GB RAM**
- Your system has: **18GB total RAM**
- 2 instances = **12-20GB** = System freeze! ❌
- 1 instance = **6-10GB** = Safe and stable ✅

**This is why your previous attempt froze the computer!**

## Files You Need

### Scripts
- `cli` - Main translator (rebuilt with llamacpp support) ✅
- `batch_translate_llamacpp.sh` - Safe batch translation ✅
- `monitor_llamacpp_translation.sh` - Progress monitoring ✅

### Directories
- `Books/` - Put Russian books here
- `Books/Translated_Llamacpp/` - Serbian translations appear here
- `logs/` - Translation logs and statistics

## Expected Performance

### Speed
- Model loading: 2-3 minutes (one-time per session)
- Translation: ~27 tokens/second
- Typical book (38 chapters): **3-10 hours**

### Quality
- ★★★★☆ High quality (comparable to commercial APIs)
- Proper Serbian Cyrillic (Ekavica dialect)
- Literary style preservation
- Context-aware translation

## Troubleshooting

### "System is frozen/slow"
→ You ran multiple instances! Force quit and restart with 1 instance only.

### "Model not found"
→ First run downloads model (5-10GB). This is normal, wait for download.

### "llama-cli not found"
→ Install: `brew install llama.cpp`

### "Out of memory"
→ Close other applications, ensure 8GB+ RAM available

## Cost
- **Setup**: Free (download 5-10GB model once)
- **Per book**: $0 (completely free)
- **Privacy**: 100% local, offline capable

## Next Steps

1. ✅ Current translation is running - let it finish
2. When done, check output: `Books/Translated_Llamacpp/Son_Nad_Bezdnoy_SR.epub`
3. For more books, use: `./batch_translate_llamacpp.sh Books/ Books/Translated_Llamacpp/`
4. Read full report: `LLAMACPP_TRANSLATION_REPORT.md`

## Help
```bash
# Show CLI help
./cli -help

# List available models
ls -lh ~/.cache/translator/models/

# Check system resources
./cli --hardware-info  # (if implemented)
```

---

**Remember**: Always 1 instance at a time! Sequential processing prevents freeze.
