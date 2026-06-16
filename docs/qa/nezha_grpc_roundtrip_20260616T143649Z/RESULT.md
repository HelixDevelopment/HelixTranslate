# Task 2 — gRPC real translate round-trip on nezha (§11.4.107 real content)

**Run UTC:** 20260616T143649Z
**Surface:** gRPC `nezha.local:50061` TranslationService (image 2d0c925, post-fix)
**Client:** `cmd/grpc-translate-probe/main.go` (standalone wire-protocol client; built to /tmp, NOT committed as a binary)
**Provider:** deepseek (key present in grpc container; bridge selects strongest verified model)

## Flow (StartTranslation → poll GetTranslationStatus → read output file)
1. Staged real Russian input into the grpc container: `/tmp/grpc_in.txt` (via `podman cp`).
2. `StartTranslation(input=/tmp/grpc_in.txt, output=/tmp/grpc_out.epub, src=ru, dst=sr, provider=deepseek)` → status `started`.
3. Polled `GetTranslationStatus` every 3s: `running 50%` → `completed 100%` in **3m21s**.
4. Completed status reported 3 Files: `original_md`, `translated_md`, `epub`.

## Real content (§11.4.107 — not placeholder)
| | Text |
|---|---|
| Original (RU) | `Доброе утро. Меня зовут Иван. Я живу в большом городе. Сегодня хорошая погода, и я хочу пойти на прогулку в парк.` |
| Translated (SR) | `Добар дан. Зовем се Иван. Живим у великом граду. Данас је лепо време, а хтео сам да одем на шетњу у парк.` |

Genuine fluent Serbian Cyrillic — correct translation of the source.

## EPUB artifact (§11.4.38 — open + verify)
- `file` → `EPUB document`; valid zip with mimetype, META-INF/container.xml, OEBPS/content.opf, toc.ncx, chapter1.xhtml (5 files, 1759 B).
- `OEBPS/chapter1.xhtml` contains the real Serbian translation: **PASS** (`grep "Зовем се Иван"` matched).

## Verdict: PASS — gRPC async file-job translate works end-to-end on the live stack with real LLM output and a valid translated EPUB.
