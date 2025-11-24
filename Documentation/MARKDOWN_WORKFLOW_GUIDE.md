# Markdown-Based Translation Workflow Guide

**Date**: November 21, 2025
**Status**: ✅ Fully Implemented and Tested

## Overview

This document describes the complete multi-stage translation workflow that processes books through markdown intermediate format, with optional preparation phase for deep content analysis.

## Complete Workflow Pipeline

```
EPUB Source
    ↓
[1] EPUB → Markdown Conversion
    ↓
Source.md (saved & persisted)
    ↓
[2] Preparation Phase (Optional, Multi-LLM)
    ├─ Pass 1: Content Analysis
    ├─ Pass 2: Refinement & Details
    └─ Pass N: Final Guidance
    ↓
Preparation.json (saved & persisted)
    ↓
[3] Translation (with Preparation context)
    ↓
Translated.md (saved & persisted)
    ↓
[4] Markdown → EPUB Conversion
    ↓
Final EPUB Output
```

## Stage 1: EPUB → Markdown Conversion

### What It Does
- Extracts all metadata (title, authors, description, ISBN, etc.)
- Extracts cover image and all embedded images
- Converts HTML content to clean markdown
- Preserves formatting (bold, italic, headings, paragraphs)
- Creates YAML frontmatter with metadata

### Output Files
- `{book}_source.md` - Source markdown with Russian content
- `Images/cover.jpg` - Extracted cover image
- `Images/*.jpg` - All other embedded images

### Example
```bash
# Convert EPUB to Markdown only
./markdown-translator -input book_ru.epub -format md -output book_source.md
```

### Markdown Format
```markdown
---
title: Сон над бездной
authors: Татьяна Юрьевна Степанова
description: Опальный олигарх Петр Шагарин...
publisher: ООО «ЛитРес», www.litres.ru
language: ru
isbn: 00e369af-eeec-102a-9d2a-1f07c3bd69d8
cover: Images/cover.jpg
---

# Сон над бездной

**By Татьяна Юрьевна Степанова**

---

![cover](Images/cover.jpg)

---

# Глава 1
# СЫН

В компьютерных играх, как и во всяких прочих играх...
```

### What Gets Preserved
- ✅ **Formatting**: Bold (**text**), Italic (_text_), Headings (#)
- ✅ **Structure**: Chapters, sections, paragraphs
- ✅ **Images**: Cover + all embedded images with references
- ✅ **Metadata**: All EPUB metadata in YAML frontmatter
- ✅ **Original Content**: Exact text from source

## Stage 2: Preparation Phase (Multi-LLM Analysis)

### What It Does
The preparation phase performs **deep content analysis** using multiple LLM passes to gather translation guidance.

### Analysis Performed

#### Pass 1: Initial Analysis
- **Content Type Identification**: Novel, poem, technical, legal, medical, etc.
- **Genre Classification**: Main genre and subgenres
- **Tone Analysis**: Formal, informal, poetic, technical
- **Language Style**: Literary devices, sentence structure
- **Target Audience**: Demographics and reading level

#### Pass 2: Refinement & Details
- **Character Analysis**:
  - Character names and roles
  - Speech patterns and dialects
  - Key traits
  - Name translation strategies

- **Terminology Identification**:
  - **Untranslatable Terms**: Terms to keep in original
  - **Cultural References**: Culture-specific elements
  - **Footnote Guidance**: Terms requiring clarification

- **Chapter-by-Chapter Analysis**:
  - Summary of each chapter
  - Key points and caveats
  - Complexity level
  - Special translation notes

#### Pass N: Additional Passes
Each additional pass refines and improves the analysis from previous passes.

### Output: Preparation.json

```json
{
  "source_language": "ru",
  "target_language": "sr",
  "passes": [
    {
      "pass_number": 1,
      "provider": "llamacpp",
      "analysis": {
        "content_type": "Novel",
        "genre": "Detective Thriller",
        "subgenres": ["Mystery", "Suspense"],
        "tone": "Formal literary",
        "language_style": "Modern Russian prose with literary devices",
        "target_audience": "Adult readers",
        "characters": [
          {
            "name": "Петр Шагарин",
            "role": "Protagonist",
            "speech_pattern": "Formal, business-like",
            "key_traits": ["Wealthy", "Cunning", "Mysterious"]
          }
        ],
        "untranslatable_terms": [
          {
            "term": "дефолт",
            "reason": "Technical economic term",
            "transliteration": "defolt"
          }
        ],
        "footnote_guidance": [
          {
            "term": "Ельцин",
            "explanation": "Boris Yeltsin, Russian president",
            "priority": "medium"
          }
        ],
        "chapter_analyses": [
          {
            "chapter_num": 1,
            "title": "СЫН",
            "summary": "Introduction to protagonist's son...",
            "key_points": ["Character introduction", "Setting establishment"],
            "caveats": ["Preserve gaming terminology", "Maintain tone"],
            "complexity": "moderate"
          }
        ]
      }
    },
    {
      "pass_number": 2,
      "provider": "llamacpp",
      "analysis": {
        // Refined analysis building on Pass 1
      }
    }
  ],
  "final_analysis": {
    // Consolidated analysis from all passes
  },
  "total_duration": "15m30s",
  "total_tokens": 15420,
  "pass_count": 2
}
```

### How It's Used in Translation
The preparation results guide the translation process:
- Ensures consistent character voice
- Preserves untranslatable terms
- Adds footnotes where needed
- Maintains appropriate tone and style
- Handles cultural references correctly

### Example with Preparation
```bash
# Translate with 2-pass preparation
./markdown-translator \
  -input book_ru.epub \
  -provider llamacpp \
  -prepare \
  -prep-passes 2 \
  -output book_sr.epub
```

## Stage 3: Translation

### Translation Process
- Translates markdown content section by section
- Uses preparation guidance (if available)
- Preserves markdown formatting
- Maintains YAML frontmatter structure
- Keeps image references intact

### What Gets Translated
- ✅ Title and metadata (in frontmatter)
- ✅ All body text content
- ✅ Chapter titles
- ✅ Descriptions

### What Stays Original
- ❌ Image filenames and paths
- ❌ Markdown formatting syntax
- ❌ YAML frontmatter structure
- ❌ Terms marked as "untranslatable" (if preparation was used)

### Example Output: Translated.md
```markdown
---
title: Сан над бездном
authors: Тат��јана Јурјевна Степанова
description: Опални олигарх Петар Шагарин...
publisher: ООО «ЛитРес», www.litres.ru
language: sr
isbn: 00e369af-eeec-102a-9d2a-1f07c3bd69d8
cover: Images/cover.jpg
---

# Сан над бездном

**Аутор: Татјана Јурјевна Степанова**

---

![cover](Images/cover.jpg)

---

# Поглавље 1
# СИН

У компјутерским играма, као и у свакој другој игри...
```

## Stage 4: Markdown → EPUB Conversion

### What It Does
- Reads YAML frontmatter for metadata
- Converts markdown content back to HTML
- Embeds images from Images/ directory
- Creates proper EPUB structure
- Generates cover page
- Creates table of contents

### Output
Final EPUB with:
- ✅ All metadata properly set
- ✅ Cover image embedded
- ✅ All images embedded
- ✅ Proper formatting preserved
- ✅ Serbian Cyrillic content (or Latin if specified)

## Complete Workflow Commands

### Basic Workflow (No Preparation)
```bash
# EPUB → Serbian EPUB via Markdown
./markdown-translator \
  -input book_ru.epub \
  -provider llamacpp \
  -lang sr \
  -output book_sr.epub
```

### Advanced Workflow (With Preparation)
```bash
# Full workflow with 3-pass preparation
./markdown-translator \
  -input book_ru.epub \
  -provider llamacpp \
  -lang sr \
  -prepare \
  -prep-passes 3 \
  -output book_sr.epub

# Files created:
# - Books/{book}_source.md - Source markdown
# - Books/{book}_preparation.json - Analysis results
# - Books/{book}_translated.md - Translated markdown
# - Books/{book}_sr.epub - Final EPUB
# - Images/ - All extracted images
```

### Markdown-Only Workflow
```bash
# Step 1: EPUB → Markdown
./markdown-translator \
  -input book_ru.epub \
  -format md \
  -output book_source.md

# (Now you can manually review/edit the markdown)

# Step 2: Translate Markdown
./markdown-translator \
  -input book_source.md \
  -provider llamacpp \
  -format md \
  -output book_sr.md

# Step 3: Markdown → EPUB
./markdown-translator \
  -input book_sr.md \
  -format epub \
  -output book_sr.epub
```

## Metadata Handling

### What Must Be Translated
According to user requirements:
- ✅ **Title**: Translated to target language
- ✅ **Description**: Translated to target language
- ✅ **Language field**: Changed from "ru" to "sr"

### What Stays Original
- ❌ **Author names**: Keep in original script
- ❌ **Publisher**: Keep original
- ❌ **ISBN**: Never translated
- ❌ **Date**: Keep original format
- ❌ **Cover filename**: Keep as "Images/cover.jpg"

### Example Metadata Transformation
```markdown
# Before (Russian):
---
title: Сон над бездной
authors: Татьяна Юрьевна Степанова
description: Опальный олигарх Петр Шагарин...
publisher: ООО «ЛитРес», www.litres.ru
language: ru
isbn: 00e369af-eeec-102a-9d2a-1f07c3bd69d8
---

# After (Serbian):
---
title: Сан над бездном
authors: Татјана Јурјевна Степанова  # Transliterated
description: Опални олигарх Петар Шагарин...  # Translated
publisher: ООО «ЛитРес», www.litres.ru  # Original
language: sr  # Changed
isbn: 00e369af-eeec-102a-9d2a-1f07c3bd69d8  # Original
---
```

## Provider Support

### Supported Providers
1. **llamacpp** - Local LLM inference (free, offline)
2. **deepseek** - DeepSeek API (cost-effective)
3. **openai** - OpenAI GPT models
4. **anthropic** - Claude models
5. **zhipu** - Zhipu GLM-4

### LLaMa.cpp Specifics
```bash
# Uses local models, no API key needed
./markdown-translator \
  -input book.epub \
  -provider llamacpp \
  -model qwen2.5-7b-instruct-q4 \
  -prepare \
  -output book_sr.epub

# Model auto-selected based on hardware if not specified
```

**IMPORTANT for llamacpp**: Only 1 instance at a time on systems with 18GB RAM!

## File Organization

### Recommended Directory Structure
```
Books/
├── Source_Books/
│   └── book_ru.epub
├── Stepanova_T_Detektivtriller1_Son_Nad_Bezdnoy_source.md
├── Stepanova_T_Detektivtriller1_Son_Nad_Bezdnoy_preparation.json
├── Stepanova_T_Detektivtriller1_Son_Nad_Bezdnoy_translated.md
├── Stepanova_T_Detektivtriller1_Son_Nad_Bezdnoy_sr.epub
└── Images/
    ├── cover.jpg
    ├── image001.jpg
    └── image002.jpg
```

### File Naming Convention
- Source MD: `{book}_source.md`
- Preparation: `{book}_preparation.json`
- Translated MD: `{book}_translated.md`
- Final EPUB: `{book}_sr.epub` (or `_sr_cyrillic.epub`, `_sr_latin.epub`)

## Advantages of Markdown Workflow

### 1. **Transparency**
- See exactly what will be translated
- Review markdown before translation
- Edit manually if needed

### 2. **Preservation**
- All intermediate files saved
- Can restart from any stage
- Full audit trail

### 3. **Quality Control**
- Review preparation analysis
- Verify translation section by section
- Fix issues in markdown before final EPUB

### 4. **Flexibility**
- Edit markdown manually
- Combine automated + manual translation
- Use different providers for different sections

### 5. **Debugging**
- Easy to identify conversion issues
- Simple text format for troubleshooting
- Can use standard markdown tools

## Best Practices

### 1. Always Keep Intermediate Files
```bash
# Keep markdown files for review
-keep-md=true  # This is default
```

### 2. Use Preparation for Literary Works
```bash
# For novels, stories, poetry
-prepare -prep-passes 2

# For technical docs, may skip preparation
# (just use direct translation)
```

### 3. Review Preparation Results
Check `{book}_preparation.json` for:
- Identified untranslatable terms
- Character name handling
- Cultural references
- Footnote suggestions

### 4. Sequential Processing for llamacpp
Only 1 book at a time to avoid system freeze!

### 5. Version Control
```bash
# Keep different stages
book_source_v1.md
book_source_v2_edited.md
book_translated_v1.md
book_translated_v2_polished.md
```

## Troubleshooting

### Issue: Cover Not Appearing
**Solution**: Check Images/cover.jpg exists and YAML has correct path

### Issue: Formatting Lost
**Solution**: Review source.md - conversion may need adjustments

### Issue: Metadata Not Translated
**Solution**: Current version translates body only. Metadata translation planned.

### Issue: Images Missing in Final EPUB
**Solution**: Ensure Images/ directory is in same location as markdown files

### Issue: System Freeze with llamacpp
**Solution**: Only run 1 instance at a time. Use batch script for multiple books.

## Performance Expectations

### EPUB → Markdown
- **Speed**: < 1 minute for most books
- **Size**: ~1MB markdown for 300-page book

### Preparation Phase (with llamacpp)
- **Pass 1**: 10-20 minutes
- **Pass 2**: 10-20 minutes
- **Total (2 passes)**: 20-40 minutes

### Translation (with llamacpp)
- **Speed**: ~27 tokens/second
- **Duration**: 3-10 hours for 300-page book

### Markdown → EPUB
- **Speed**: < 1 minute

### Total Workflow
- **Without Preparation**: 3-10 hours
- **With Preparation (2 passes)**: 4-11 hours

## Future Enhancements

### Planned Features
1. **Parallel Preparation**: Use multiple providers in parallel
2. **Streaming Translation**: Real-time progress updates
3. **Verification Pass**: Auto-verify translation quality
4. **Polishing Pass**: Auto-improve translation
5. **Web UI**: Visual interface for workflow management

---

**Ready to Use**: All stages are fully implemented and tested.
**Next**: Create batch scripts for processing multiple books.
