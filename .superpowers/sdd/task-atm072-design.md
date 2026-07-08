# ATM-072 Design Proposal: Markdown as First-Class CLI Input Format

**Revision:** 1
**Last modified:** 2026-07-08T14:45:00Z
**Scope:** Investigation + design proposal only (no source edits per §11.4.122)

## Summary

`.md` files are supported as OUTPUT (`generateOutput` at `main.go:1125` handles `"txt", "md"`) but NOT as INPUT. Passing `-i book.md` fails with "unknown or unsupported format" because `detectByExtension` has no `.md` case.

## Evidence

**Format detector:** `pkg/format/detector.go:162-190` — `detectByExtension` switch has cases for fb2/epub/pdf/mobi/azw/azw3/txt/html/docx/rtf but NOT `.md`. Falls through to `FormatUnknown`.

**Parser dispatch:** `pkg/ebook/parser.go:88-90` — `FormatUnknown` returns error `"unknown or unsupported format"`. No fallback to TXT.

**Output support:** `cmd/unified-translator/main.go:1125` — `case "txt", "md":` handles `.md` output (writes plain text to a `.md` file).

**Help text:** `main.go:726` — "Input ebook file (FB2, EPUB, PDF, DOCX, TXT, HTML)" — correctly omits `.md` (it's not supported).

**Markdown package:** `pkg/markdown/` contains EPUB↔Markdown converters (`epub_to_markdown.go`, `markdown_to_epub.go`) used by `cmd/markdown-translator`. NOT wired into the main CLI pipeline.

## What "first-class markdown input" means

Currently, `.md` input works IF you rename it to `.txt` — the TXT parser reads it as plain text, translates the text content, and the output preserves no markdown structure (headers, bold, links, lists become translated text without formatting).

First-class markdown input would:
1. Detect `.md` as a format (add to `detectByExtension`)
2. Parse markdown structure (headers, paragraphs, lists, code blocks, links)
3. Translate only translatable text (skip code blocks, URLs, image paths)
4. Preserve markdown structure in output

## Design options

### Option A — Register `.md` as TXT alias (minimal, 1-line fix)

**Approach:** Add `case "md":` to `detectByExtension` returning `FormatTXT`. The TXT parser reads the file, translates all text content. Markdown structure is preserved only incidentally (translating "## Header" produces "## Header" because the `#` is not translatable text).

**Pros:** 1-line fix. `.md` files work immediately. Structure preservation is "good enough" for many cases.
**Cons:** Code blocks, URLs, image paths get translated (the TXT parser doesn't know they're special). Bold/italic markers (`**text**`) may get mangled by the LLM. Not truly "first-class."

### Option B — Add a dedicated markdown parser (proper, medium complexity)

**Approach:** Create `pkg/ebook/markdown_parser.go` that:
1. Parses markdown into a tree (headings, paragraphs, lists, code blocks, links)
2. Extracts only translatable text nodes (skip code blocks, URLs, image alt-text)
3. Returns a `Book` with chapters = headings, sections = paragraphs
4. After translation, reconstructs markdown from the translated text + preserved structure

**Pros:** Proper structure preservation. Code blocks/URLs/images untouched. Translation quality improves (LLM sees clean prose, not markdown syntax).
**Cons:** Medium complexity (need a markdown parser — use `goldmark` or similar). New parser needs full test suite. More surface area.

### Option C — Use existing `pkg/markdown` converters (reuse, but indirect)

**Approach:** Convert `.md` → EPUB (using `markdown_to_epub.go`) → translate EPUB → convert back to `.md` (using `epub_to_markdown.go`). Leverages existing, tested converters.

**Pros:** Reuses existing code. EPUB path is well-tested.
**Cons:** Lossy round-trip (markdown → EPUB → markdown may lose some formatting). Adds complexity to the pipeline. Two format conversions instead of one.

## Recommendation

**Option A as immediate fix, Option B as follow-up feature.**

Option A is a 1-line fix that unblocks `.md` input TODAY. The TXT parser handles it "good enough" — markdown syntax characters (`#`, `*`, `-`, `` ` ``) are mostly preserved because the LLM treats them as non-translatable. It's not perfect (code blocks get translated), but it's better than "unknown or unsupported format."

Option B is the proper solution but is a non-trivial feature (new parser + test suite + integration). Should be a separate work item, not a gate fix.

**Implementation plan (Option A):**
1. Add `case "md":` to `detectByExtension` in `pkg/format/detector.go` — returns `FormatTXT`
2. Update help text in `main.go:726` to include `.md` in input list
3. Add test: `.md` file detected as TXT format
4. Add test: `.md` file translates successfully (content preserved)
5. RED→GREEN per §11.4.43

**Risk:** VERY LOW — it's adding an alias, not changing behavior. The TXT parser already handles the content correctly.

## Git history

`.md` output support was added in the `generateOutput` function. No prior attempt to add `.md` input support.
