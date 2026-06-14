# E2E Proof — PDF input revived (unipdf license-gate removed)

**Date:** 2026-06-14
**Constitution:** §11.4 anti-bluff, §11.4.107 captured evidence, §11.4.120 gate reconciliation, §11.4.135 regression guard, §11.4.146 reproduce→fix→extend

## Defect
PDF input shipped backed by `github.com/unidoc/unipdf`, whose text extractor is
LICENSE-GATED (`ExtractText` → "unipdf license code required" for every real PDF).
PDF input was non-functional — a §11.4 "ships but cannot be used" defect masked by
unit tests that only fed invalid bytes and asserted failure.

## Fix
Swapped to MIT `github.com/ledongthuc/pdf`; `unidoc/unipdf` dropped from go.mod
via `go mod tidy`. Wired `FormatPDF` into `pkg/format/detector.go` IsSupported +
GetSupportedFormats (§11.4.120). Companion DOCX fix landed earlier (stdlib OOXML).

## Real-system proof (this directory)
- `input_sample_en.pdf` — minimal valid single-page English PDF (3 known sentences)
- `output_sr.epub` — real DeepSeek (deepseek-chat) translation output, 1.9s real API call
- `extracted_chapter.xhtml` — OEBPS/chapter1.xhtml extracted from the EPUB
- `run.log` — translator run log (CLI never prints API keys)

### Extracted Serbian (Cyrillic) output
> Храбри витез јахао је преко тихог сванућа у зору. Носио је старо писмо
> запечаћено црвеним воском. Планине су стајале тихо под бледим јутарњим небом.

(= "The brave knight rode across the silent valley at dawn. He carried an old
letter sealed with red wax. The mountains stood quiet under a pale morning sky.")

### Anti-bluff checks (all PASS)
1. Cyrillic chars in output: 138 (>20) — real target-script text
2. Output differs from English source (no pass-through)
3. No placeholder/TODO/mock text
4. Non-trivial length: 279 chars
5. NO API-key leak across all evidence (§11.4.10)

## Permanent regression guard (§11.4.135)
`pkg/ebook/pdf_extraction_regression_test.go` +
`pkg/ebook/testdata/sample_text.pdf` — asserts real extraction of the known
sentences AND rejects any "license"-error string. Mutation (§1.1): revert to the
unipdf extractor → extraction returns empty/license-error → test FAILs.
