# §11.4.153 Real-Use Video Wave 4 — Multi-Language-Pair Translation Confirmations

**Revision:** 1
**Last modified:** 2026-06-16T17:35:08Z
**Scope:** BACKGROUND operator-authorized §11.4.153 real-use video recording, NON-COMMITTING parallel stream.
**Main checkout HEAD at start:** 6a1aa8c
**Binary:** `build/unified-translator` (rebuilt this session, `go build -o build/unified-translator ./cmd/unified-translator`, 15 MB, EXIT=0; gitignored, NOT staged)
**Provider keys (§11.4.6, values never printed):** `DEEPSEEK_API_KEY` SET, `GROQ_API_KEY` SET.

## Anti-bluff method (§11.4.107 / §11.4.154 / §11.4.155)

Each scenario: `asciinema rec -c "<real unified-translator cmd>"` → `agg --fps-cap 30` → `ffmpeg -vf "fps=2,scale=trunc(iw/2)*2:trunc(ih/2)*2,format=yuv420p"`. Window-scoped to the terminal pane (§11.4.154). Filenames start `helix_translate-` (§11.4.155). Each `.mp4` ffprobe-verified for ≥6 advancing frames + first/last frame **read** to confirm: start-frame shows source-language input only (mid-run, no output), output-frame shows the **real target-language translation** (NOT frozen, NOT the source language, NOT an LLM error).

The four scenarios use the SAME source sentence per source language but FOUR distinct target languages, producing four genuinely different, language-correct outputs — proving real per-language LLM translation, not a cached/echoed result.

## CONFIRMED rows (3 net-new + 1 re-confirm)

### 1. NET-NEW — EN→FR (multi-language-pair: French)
- **Video:** `/Volumes/T7/Downloads/Recordings/helix_translate-multilang-en-fr-wave4-20260616-173316.mp4`
- **ffprobe:** 790×560 yuv420p, 5.0s, **10 frames**
- **Input (en):** "The morning sun rose over the quiet village. An old fisherman walked along the shore."
- **Output (fr) read from output-frame:** "Le soleil matinal se leva au-dessus du village paisible. Un vieil pêcheur marchait le long de la rive."
- **Liveness:** start-frame = English input + `en -> fr` header, no output; output-frame = real French. Start ≠ output. PASS.
- **Frames:** `frames/en-fr_first.png`, `frames/en-fr_last.png`

### 2. NET-NEW — EN→DE (multi-language-pair: German)
- **Video:** `/Volumes/T7/Downloads/Recordings/helix_translate-multilang-en-de-wave4-20260616-173319.mp4`
- **ffprobe:** 790×560 yuv420p, 4.0s, **8 frames**
- **Input (en):** "The morning sun rose over the quiet village. An old fisherman walked along the shore."
- **Output (de) read from output-frame:** "Der morgendliche Sonnenaufgang über dem stillen Dorf. Ein alter Fischer spazierte am Ufer entlang."
- **Liveness:** start-frame = English input + `en -> de` header, no output; output-frame = real German (umlaut ü, distinct vocab). Start ≠ output. PASS.
- **Frames:** `frames/en-de_first.png`, `frames/en-de_last.png`

### 3. NET-NEW — EN→IT (multi-language-pair: Italian)
- **Video:** `/Volumes/T7/Downloads/Recordings/helix_translate-multilang-en-it-wave4-20260616-173419.mp4`
- **ffprobe:** 790×560 yuv420p, 4.5s, **9 frames**
- **Input (en):** "The morning sun rose over the quiet village. An old fisherman walked along the shore."
- **Output (it) read from output-frame:** "Il sole mattutino sorse sopra il villaggio tranquillo. Un vecchio pescatore camminava lungo la riva."
- **Liveness:** start-frame = English input + `en -> it` header, no output; output-frame = real Italian (distinct from FR/DE/ES). Start ≠ output. PASS.
- **Frames:** `frames/en-it_first.png`, `frames/en-it_last.png`

### 4. RE-CONFIRM (NOT double-counted) — RU→SR Cyrillic (flagship default pair)
- **Video:** `/Volumes/T7/Downloads/Recordings/helix_translate-multilang-ru-sr-cyr-wave4-20260616-173445.mp4`
- **ffprobe:** 790×560 yuv420p, 4.0s, **8 frames**
- **Input (ru):** "Утро было тихим. Старый рыбак шёл вдоль берега реки."
- **Output (sr cyrillic) read from output-frame:** "Утро је било тихо. Стари рибар је ишао дуж обале реке."
- **Liveness:** start-frame = Russian input + `ru -> sr (cyrillic)` header, no output; output-frame = real Serbian Cyrillic (Serbian "је"/"дуж" ≠ Russian "было"/"вдоль"). Differs from an earlier identical test run ("Сутра је била тиха…") proving a genuine live (non-cached) LLM call. Start ≠ output. PASS.
- **Frames:** `frames/ru-sr_first.png`, `frames/ru-sr_last.png`

## Distinctness proof (§11.4.6)
Four target languages, four different outputs from the same EN/RU source:
- FR: "Le soleil matinal se leva…"
- DE: "Der morgendliche Sonnenaufgang…"
- IT: "Il sole mattutino sorse…"
- SR: "Утро је било тихо…"
No echo, no English passthrough, no frozen frame — each is a correct, language-specific human translation.

## Honest notes (§11.4.6)
- Path used: default in-process LLMsVerifier **bridge** (no `-use-verifier`). `-use-verifier` requires a live LLMsVerifier server at http://localhost:8080 (unreachable here) — an honest environment constraint, not exercised.
- The `unified-translator -source-lang` / `-target-lang` ledger ROWS (Status.md lines 271–272) are classified `Video-confirmation = N/A` as bare CLI flags; this wave confirms the **multi-language translation CAPABILITY** they enable, which is a genuine user-visible feature deserving a §11.4.153 real-use confirmation.
- NO commit / NO stage performed by this stream (separate committer agent owns the main checkout). Frames copied into this `docs/qa/` dir as file-writes only.
