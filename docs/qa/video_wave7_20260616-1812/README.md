# §11.4.153 Real-Use Video Recording — WAVE 7

**Revision:** 1
**Last modified:** 2026-06-16T18:14:00Z

NON-COMMITTING parallel stream. Main checkout HEAD 0726c20. Real input → real
DeepSeek (default in-process bridge, NOT -use-verifier) → real output. Each clip
window-scoped (110×32 terminal pane, §11.4.154), `helix_translate-` prefix
(§11.4.155), frames extracted + content-read (§11.4.107: source frame ≠ output
frame). Prior `helix_translate-*` recordings NOT deleted (wave7 naming avoids
collision per §11.4.154 same-scope rule).

## CONFIRMED (4) — real video path + real output read

| # | Feature | Binary / endpoint | Clip | Real result read from frame |
|---|---------|-------------------|------|------------------------------|
| 1 | EPUB→EPUB round-trip (EN→ES) | `unified-translator -i in.epub -o out.epub` | `helix_translate-epub-to-epub-en-es-wave7-20260616-181058.mp4` | EN "The old man walked along the shore at dawn…" → ES "El anciano caminaba por la orilla al amanecer…"; output verified `EPUB document` |
| 3 | Script conversion Cyrillic→Latin (pkg/script, NO LLM translation) | `unified-translator -script latin` (sr→sr) | `helix_translate-script-cyrillic-to-latin-wave7-20260616-181012.mp4` | "Добар дан свима…ђ…џ" → "Dobar dan svima…đ…dž"; zero Cyrillic remaining |
| 4 | preparation-translator analysis (characters/terminology/culture) | `preparation-translator -input … -analysis …` | `helix_translate-preparation-analysis-en-es-wave7-20260616-181123.mp4` | genre "science fiction"; subgenres space opera + military sci-fi; chars Sarah Mills (protagonist) / Admiral Chen (antagonist); untranslatable terms Captain Sarah Mills, Admiral Chen, starship Aurora, New Geneva |
| 5 | REST POST /api/v1/translate/string (local ./build/server :8080) | curl → JSON | `helix_translate-rest-translate-string-en-es-wave7-20260616-181155.mp4` | JSON `translated_text` = "Las montañas estaban cubiertas de nieve y el río fluía tranquilamente por el valle.", provider deepseek (NOT just session_id) |

All 4: ffprobe advancing frames (3–4 read frames), yuv420p, even dims. Frame
images in `frames/`.

## FAILED / BROKEN (1) — real bug, NOT confirmed (§11.4.146 / §11.4.138)

### markdown-translator EPUB→MD: empty-payload data loss → garbage output

`markdown-translator -input w7_in.epub -output w7_md.md -format md -lang es -provider deepseek`

- **Step 1 (EPUB→MD) is CORRECT**: source markdown contains real content —
  "# The Lighthouse", "The old man walked along the shore at dawn…" (see
  `BUG_markdown_translator_source_was_correct.md`).
- **Step 3 (translate) is BROKEN**: it sends EMPTY strings to DeepSeek per
  block. DeepSeek replies with its "It seems you may have accidentally sent an
  empty message…" boilerplate, which is written into the output AS IF it were
  the translation (see `BUG_markdown_translator_garbage_output.md`). The real
  chapter text is lost; output is unusable.
- Bridge source reported: `in-process` (confirms it is the real default path,
  NOT the unreachable verifier).
- Root cause (§11.4.102): markdown translation step feeds empty/whitespace
  segments to the LLM instead of the actual block content — markdown→translate
  payload-extraction defect, distinct from the markdown→EPUB round-trip fixes in
  prior commits.
- **NOT video-recorded as a confirmation** (would be a bluff video per
  §11.4.153). Captured text evidence committed here instead.

## NET-NEW count

Reported separately because committed Status.md base is stale mid-edit (a
separate committer agent is editing it; this stream did NOT read it). Against the
wave 1–6 coverage stated in the dispatch (TXT translate; lang pairs
FR/DE/IT/PT/JA/ZH/RU/SR; -multipass; HTML→HTML; PDF/DOCX input; REST
/convert/script + /translate/fb2), all 4 confirmed features are DISTINCT:

- **NET-NEW confirmed: 4** (EPUB→EPUB round-trip; standalone -script
  Cyrillic→Latin; preparation-translator analysis pass; REST
  /api/v1/translate/string).
- **NET-NEW bug found: 1** (markdown-translator EPUB→MD empty-payload garbage).
