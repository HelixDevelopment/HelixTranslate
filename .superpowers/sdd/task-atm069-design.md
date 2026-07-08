# ATM-069 Design Proposal: Inert Config Fields (DOCXConfig/PDFConfig)

**Revision:** 1
**Last modified:** 2026-07-08T14:20:00Z
**Scope:** Investigation + design proposal only (no source edits per §11.4.122)

## Summary

3 config fields are **genuinely inert** — declared, tested for initialization, but never consumed in parsing logic. No config.json populates them (not a §11.4.108 lie-class). All introduced in "Auto-commit" scaffolds.

---

## Per-field analysis

### 1. `DOCXConfig.MinTextLength` (default: 1)

**State:** INERT — declared at `docx_parser.go:40`, default set at line 56, never referenced in parsing logic.

**Evidence:**
- Declaration: `pkg/ebook/docx_parser.go:40` — `MinTextLength int \`yaml:"min_text_length"\``
- Default: `docx_parser.go:56` — `MinTextLength: 1`
- Test: `docx_parser_test.go:140-141` — asserts default is 1. `docx_parser_test.go:150/164` — asserts custom value is stored. But NO test asserts the value affects parsing output.
- Parsing logic: `docx_parser.go` `Parse()` function never reads `config.MinTextLength`. All paragraphs are extracted regardless of length.
- Config files: `config.json` and `internal/working/config*.json` do NOT populate `min_text_length`.
- Git history: `81f7329` ("Auto-commit") — scaffold, never wired.

**What wiring would mean:** In the DOCX parser's paragraph extraction loop, skip paragraphs where `len(text) < config.MinTextLength`. This filters short paragraphs (e.g., page numbers, single-word headers) from translation. **Impact:** changes translation output — paragraphs currently included would be dropped. Default must be 1 (preserve current behavior) or 0 (include all).

**OPTION A — Wire:** Add `if len(strings.TrimSpace(text)) < parser.config.MinTextLength { continue }` in the paragraph extraction loop. Test: create a DOCX with known short+long paragraphs, parse with `MinTextLength=5`, assert short paragraphs absent from output. Default stays 1 (backward-compatible).

**OPTION B — Remove:** Drop the field + yaml tag. No operator relies on it (not in any config file). Clean.

**RECOMMENDATION:** Wire (OPTION A) with default=1. It's a useful filter (short paragraphs like "1", "Page 2", "—"' are noise in translation). Low risk since default preserves current behavior. Needs a RED→GREEN test.

---

### 2. `DOCXConfig.IgnoreStyles` (default: [])

**State:** INERT — declared at `docx_parser.go:41`, default set at line 57, never referenced in parsing logic.

**Evidence:**
- Declaration: `pkg/ebook/docx_parser.go:41` — `IgnoreStyles []string \`yaml:"ignore_styles"\``
- Default: `docx_parser.go:57` — `IgnoreStyles: []string{}`
- Test: `docx_parser_test.go:151/168-169` — asserts custom value is stored. NO test asserts it affects parsing output.
- Parsing logic: `docx_parser.go` `Parse()` never reads `config.IgnoreStyles`. All styles are extracted.
- Config files: NOT populated.
- Git history: `81f7329` — scaffold.

**What wiring would mean:** In the DOCX parser, check each paragraph's style name against `config.IgnoreStyles`; skip matching paragraphs. **Impact:** changes output. Useful for ignoring TOC, index, headers, footers, etc.

**OPTION A — Wire:** Add style-check in paragraph extraction. Requires: (a) extract the paragraph's style name from the DOCX XML (already available in the parsing context), (b) check against `config.IgnoreStyles` list, (c) skip if matched. Test: DOCX with "Title" + "Normal" paragraphs, `IgnoreStyles=["Title"]`, assert "Title" paragraph absent.

**OPTION B — Remove:** Clean. No operator uses it.

**RECOMMENDATION:** Wire (OPTION A) with default=[] (backward-compatible). Useful for excluding non-translatable content (TOC, headers, footers). Medium complexity — need to extract style name from DOCX paragraph properties.

---

### 3. `PDFConfig.MinTextLength` (default: 1)

**State:** INERT — declared at `pdf_parser.go:33`, default set at line 46, never referenced in parsing logic.

**Evidence:**
- Declaration: `pkg/ebook/pdf_parser.go:33` — `MinTextLength int \`yaml:"min_text_length"\``
- Default: `pdf_parser.go:46` — `MinTextLength: 1`
- Test: `pdf_parser_test.go:160-161` — asserts default is 1. `pdf_parser_test.go:172/186` — asserts custom value stored. NO test asserts it affects parsing.
- Parsing logic: `pdf_parser.go` `Parse()` never reads `config.MinTextLength`.
- Config files: NOT populated.
- Git history: `cc07da9` — scaffold.

**What wiring would mean:** Same as DOCX — skip text blocks shorter than threshold. PDF text extraction is noisier than DOCX (headers, footers, page numbers, watermarks), so this filter is MORE useful here.

**OPTION A — Wire:** Same pattern as DOCX. Filter in the text-block extraction loop.

**OPTION B — Remove:** Clean.

**RECOMMENDATION:** Wire (OPTION A) with default=1. PDF text extraction produces many short noise blocks; filtering them improves translation quality. Same risk profile as DOCX (default preserves behavior).

---

## Summary table

| Field | Parser | Declared | Default | Consumed? | Config.json? | Recommendation |
|-------|--------|----------|---------|-----------|-------------|---------------|
| `DOCXConfig.MinTextLength` | docx_parser.go:40 | Yes | 1 | No | No | Wire (default=1) |
| `DOCXConfig.IgnoreStyles` | docx_parser.go:41 | Yes | [] | No | No | Wire (default=[]) |
| `PDFConfig.MinTextLength` | pdf_parser.go:33 | Yes | 1 | No | No | Wire (default=1) |

## §11.4.108 lie-class assessment

**NOT a lie-class.** No config.json populates these fields, so no operator is setting values expecting behavior that doesn't exist. The fields are inert scaffolds, not broken promises. However, the YAML tags mean a user COULD set `min_text_length: 50` in a config file and get silently ignored — that would be a lie. Wiring the fields prevents this future lie-class.

## Operator decision required

All 3 fields should be wired (preserving backward-compatible defaults). If the operator prefers removal instead, the YAML tags and test assertions must also be removed. The wiring is low-risk since defaults preserve current behavior.

## Git history

All fields introduced in "Auto-commit" scaffolds (`81f7329` for DOCX, `cc07da9` for PDF). No subsequent commit attempted to wire them.
