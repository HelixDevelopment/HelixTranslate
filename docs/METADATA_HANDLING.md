# Metadata Handling in Translation Pipeline

## Overview

This document describes how book metadata (title, authors, cover, etc.) is handled during the translation process, including which fields are translated and which are preserved in their original form.

## Metadata Fields

The `ebook.Metadata` structure includes the following fields:

| Field | Type | Description |
|-------|------|-------------|
| Title | string | Book title |
| Authors | []string | List of author names |
| Description | string | Book description/summary |
| Publisher | string | Publishing house name |
| Language | string | ISO language code (e.g., "ru", "sr") |
| ISBN | string | International Standard Book Number |
| Date | string | Publication date |
| Cover | []byte | Cover image data (binary) |

## Translation Rules

### Fields that MUST BE TRANSLATED

**✅ Title**
- **Rule**: Translate to target language
- **Reason**: Readers should see the book title in their language
- **Example**: "Преступление и наказание" → "Злочин и казна"

**✅ Description**
- **Rule**: Translate to target language
- **Reason**: Book summary should be in reader's language
- **Example**: "Роман Достоевского о..." → "Роман Достојевског о..."

### Fields that MUST STAY in ORIGINAL FORM

**✅ Authors**
- **Rule**: Keep original author names (proper nouns)
- **Reason**: Author names are proper nouns and should not be translated
- **Example**: "Фёдор Достоевский" (stays as is)

**✅ Publisher**
- **Rule**: Keep original publisher name (proper noun)
- **Reason**: Publisher names are proper nouns
- **Example**: "АСТ" (stays as is)

**✅ ISBN**
- **Rule**: Keep original ISBN (unique identifier)
- **Reason**: ISBN is a standardized unique identifier that never changes
- **Example**: "978-5-17-123456-7" (stays as is)

**✅ Date**
- **Rule**: Keep original publication date
- **Reason**: Historical record should not be changed
- **Example**: "2019-03-15" (stays as is)

**✅ Cover**
- **Rule**: Keep original image data (binary)
- **Reason**: Cover is visual content, not text to translate
- **Note**: Cover image is preserved as binary data ([]byte)

### Fields that MUST BE UPDATED

**✅ Language**
- **Rule**: Update from source language code to target language code
- **Example**: "ru" → "sr"
- **Reason**: Language code should reflect the language of the content

## Implementation Details

### EPUB Parser Enhancement

The EPUB parser (`pkg/ebook/epub_parser.go`) now extracts ALL metadata fields:

```go
// Extracts from content.opf:
- dc:title → Metadata.Title
- dc:creator → Metadata.Authors
- dc:description → Metadata.Description
- dc:publisher → Metadata.Publisher
- dc:language → Metadata.Language
- dc:identifier → Metadata.ISBN
- dc:date → Metadata.Date

// Extracts cover image:
- Detects cover from manifest (id="cover-image" or properties="cover-image")
- Reads image binary data from ZIP
- Stores in Metadata.Cover
```

### EPUB Writer Enhancement

The EPUB writer (`pkg/ebook/epub_writer.go`) now includes ALL metadata in output:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Translated Title</dc:title>
    <dc:creator>Original Author</dc:creator>
    <dc:description>Translated Description</dc:description>
    <dc:publisher>Original Publisher</dc:publisher>
    <dc:language>sr</dc:language>
    <dc:identifier>ISBN from source</dc:identifier>
    <dc:date>Original date</dc:date>
    <meta name="cover" content="cover-image"/>
  </metadata>
  <manifest>
    <item id="cover-image" href="cover.jpg" media-type="image/jpeg"
          properties="cover-image"/>
    <!-- chapters -->
  </manifest>
</package>
```

### Markdown Pipeline Integration

The markdown pipeline preserves metadata in YAML frontmatter:

```markdown
---
title: Translated Title
authors: Original Author(s)
description: Translated Description
publisher: Original Publisher
language: sr
isbn: Original ISBN
date: Original Date
cover_file: cover.jpg
---

# Book Content
...
```

## Usage in Translation Pipeline

### Complete Translation Flow

```
1. EPUB Parsing:
   ┌─────────────────────────────────────────┐
   │ Read EPUB → Extract ALL metadata        │
   │ - Title, Authors, Description, etc.     │
   │ - Cover image (binary)                  │
   └─────────────────────────────────────────┘
                    ↓
2. Metadata Translation:
   ┌─────────────────────────────────────────┐
   │ Translate ONLY:                         │
   │ - Title                                 │
   │ - Description                           │
   │                                         │
   │ Preserve ORIGINAL:                      │
   │ - Authors, Publisher, ISBN, Date, Cover│
   │                                         │
   │ Update:                                 │
   │ - Language (ru → sr)                    │
   └─────────────────────────────────────────┘
                    ↓
3. Content Translation:
   ┌─────────────────────────────────────────┐
   │ Translate book chapters                 │
   └─────────────────────────────────────────┘
                    ↓
4. EPUB Generation:
   ┌─────────────────────────────────────────┐
   │ Create EPUB with:                       │
   │ - Translated Title & Description        │
   │ - Original Authors, Publisher, etc.     │
   │ - Updated Language code                 │
   │ - Original Cover image                  │
   │ - Translated Content                    │
   └─────────────────────────────────────────┘
```

### Example Code

```go
// Parse source EPUB
parser := ebook.NewUniversalParser()
book, _ := parser.Parse("source_ru.epub")

// Original metadata extracted automatically:
// book.Metadata.Title = "Преступление и наказание"
// book.Metadata.Authors = ["Фёдор Достоевский"]
// book.Metadata.Description = "Роман о..."
// book.Metadata.Publisher = "АСТ"
// book.Metadata.ISBN = "978-5-17-123456-7"
// book.Metadata.Date = "2019-03-15"
// book.Metadata.Language = "ru"
// book.Metadata.Cover = [binary image data]

// Translate title and description
book.Metadata.Title = translator.Translate(book.Metadata.Title)
book.Metadata.Description = translator.Translate(book.Metadata.Description)

// Update language code
book.Metadata.Language = "sr"

// Authors, Publisher, ISBN, Date, Cover stay unchanged

// Translate content
for i := range book.Chapters {
    book.Chapters[i].Title = translator.Translate(book.Chapters[i].Title)
    // ... translate sections
}

// Write translated EPUB with ALL metadata preserved/updated
writer := ebook.NewEPUBWriter()
writer.Write(book, "translated_sr.epub")
```

## Cover Image Handling

### Cover Extraction (EPUB Parser)

The parser detects cover images using multiple strategies:

1. Manifest item with `id="cover"` or `id="cover-image"`
2. Manifest item with `properties="cover-image"` (EPUB 3)
3. Manifest item with href containing "cover"
4. Meta tag with `name="cover"` pointing to manifest item

Once detected, the cover image is:
- Read as binary data from the EPUB ZIP archive
- Stored in `book.Metadata.Cover` as `[]byte`

### Cover Inclusion (EPUB Writer)

The writer includes cover in output EPUB:

1. Writes cover image to `OEBPS/cover.jpg`
2. Adds cover to manifest:
   ```xml
   <item id="cover-image" href="cover.jpg"
         media-type="image/jpeg"
         properties="cover-image"/>
   ```
3. Adds cover meta tag:
   ```xml
   <meta name="cover" content="cover-image"/>
   ```

## Benefits

✅ **Complete Metadata Preservation**: All metadata from source book is preserved
✅ **Proper Attribution**: Authors and publishers correctly attributed
✅ **Library Compatibility**: ISBN and publication dates maintained for library systems
✅ **Professional Output**: Cover images included in translated books
✅ **Reader Experience**: Translated titles and descriptions for discoverability
✅ **Historical Accuracy**: Original publication information preserved

## Testing Recommendations

Test metadata handling with:

1. **Title Translation**: Verify title is translated
2. **Author Preservation**: Verify authors remain in original form
3. **Cover Preservation**: Verify cover image appears in output
4. **Metadata Completeness**: Verify all fields present in output
5. **EPUB Validity**: Verify output EPUB passes validation
6. **E-Reader Compatibility**: Test on actual e-readers

## Future Enhancements

Potential improvements:

- [ ] Support for multiple cover sizes (thumbnail, full)
- [ ] Cover image format detection (JPEG, PNG, etc.)
- [ ] Series information preservation
- [ ] Edition information tracking
- [ ] Rights and copyright preservation
- [ ] Subject/genre metadata preservation
- [ ] Contributor roles (translator, editor, etc.)

## See Also

- [EPUB Parser Documentation](../pkg/ebook/epub_parser.go)
- [EPUB Writer Documentation](../pkg/ebook/epub_writer.go)
- [Markdown Pipeline Documentation](MARKDOWN_PIPELINE.md)
- [Translation Pipeline Documentation](TRANSLATION.md)
