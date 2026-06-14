# E2E Proof — CLI honors -o output format (was: always emitted EPUB)

**Date:** 2026-06-14
**Constitution:** §11.4 anti-bluff, §11.4.98 self-driving tests, §11.4.107 captured evidence, §11.4.120 gate reconciliation, §11.4.135 regression guard, §11.4.146 reproduce→fix→extend

## Defect
`cmd/unified-translator` ALWAYS called `generateEPUB` for the output step,
ignoring the `-o` extension. `-o book.txt` / `-o book.fb2` wrote EPUB (PK-zip)
bytes into a misnamed file — a silent wrong-output defect (§11.4: user asks for
one format, silently gets another).

## Fix
New `generateOutput(content, outputPath, inputFile)` dispatches on the output
extension: `.epub` (default) → EPUB; `.txt`/`.md` → translated text written
directly; `.fb2` → `ebook.FB2Writer`; unsupported extension → explicit error
(§11.4.6), never a misnamed EPUB.

## Real-system proof (this directory)
Same English PDF → real DeepSeek deepseek-chat → two output formats:
- `out.txt` — UTF-8 plain text (first bytes `d0a1` = Cyrillic 'С', NOT `PK`),
  135 Cyrillic chars, real Serbian content.
- `out.fb2` — valid `<FictionBook>` XML, 138 Cyrillic chars.
- `run_txt.log`, `run_fb2.log` — translator logs (no API-key printing).

### out.txt content
> Садржај документа
> Храбри витез јахао је кроз тиху долину у зору.
> Носио је старо писмо запечаћено црвеним воском.
> Планине су стајале тихо под бледим јутарњим небом.

## Permanent regression guard (§11.4.135)
`cmd/unified-translator/output_format_test.go` — asserts .txt is not a zip,
.fb2 is FictionBook XML, unsupported ext errors. MUTATION-PROVEN: forcing
generateOutput to always-EPUB → .txt + .fb2 subtests FAIL (verified this session).

## Companion gate reconciliations (§11.4.120)
- `test/unit/format_detector_test.go` — PDF + DOCX moved to supportedFormats
  (their real extractors landed); MOBI stays unsupported.
- `cmd/cli/main_comprehensive_test.go` TestTranslateEbookFunction/with_app_config
  — was a §11.4.98 bluff: skip-guard fired only when NO key present, then forced
  openai + a fake "config-key", dialled REAL OpenAI, 401'd, and failed NoError
  whenever any other provider key was in env. Rewritten to a self-driving
  httptest OpenAI mock (config BaseURL → mock); asserts success + mock-was-hit +
  non-empty output. No real network, no env dependency.
