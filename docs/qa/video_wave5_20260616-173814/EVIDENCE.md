# §11.4.153 Real-Use Video Recording — WAVE 5

**Revision:** 1
**Last modified:** 2026-06-16T17:45:00Z
**Run-id:** video_wave5_20260616-173814
**Authority:** §11.4.153 (per-feature video confirmation), §11.4.107 (liveness/real-content), §11.4.154 (window-scoped + fresh-corpus), §11.4.155 (`helix_translate-` project-name prefix), §11.4.6 (no-guessing — only "confirmed" what a real video proves).
**Stream:** Background OPERATOR-AUTHORIZED non-committing parallel stream. This agent did NOT commit / stage / `git add` (§11.4.84/§11.4.119 — a separate committer agent owns the main checkout).

## Environment (§11.4.6)

- Main checkout HEAD: `6a1aa8c` (matched at start).
- `DEEPSEEK_API_KEY` set (verified, never printed); `GROQ_API_KEY` set.
- Path used: **default in-process bridge translate path** (NOT `-use-verifier` — LLMsVerifier at localhost:8080 unreachable this session).
- Build: `go build -o build/unified-translator ./cmd/unified-translator` → OK (gitignored, NOT staged). `go build -o build/server ./cmd/server` → OK.
- Recording toolchain: `asciinema rec -c "<cmd>"` → `agg --fps-cap 30` → `ffmpeg fps=2 -r 2 yuv420p even-dims`, window-scoped (terminal pane only).
- Recordings dir: `/Volumes/T7/Downloads/Recordings/` — wave5 filenames distinct (`helix_translate-<feature>-wave5-20260616-173814.mp4`); **NO prior `helix_translate-*` files deleted** (they are distinct prior-wave evidence, not this agent's own stale dups; distinct wave5 naming avoids collision per §11.4.154).
- Live REST server (scenarios 4-5): `cmd/server` on `127.0.0.1:9543` (fresh port, /tmp config copy), stopped after recording (§11.4.14 cleanup).

## CONFIRMED features (5 net-new, distinct from waves 1-4)

Waves 1-4 covered: many providers, DOCX/PDF/HTML→MD output, EPUB→EPUB, TXT→TXT, FB2→FB2, format-detect, CLI `-script` Cyrillic↔Latin, REST/gRPC text translate, markdown round-trip, verify-models CLI, workable-items CLI, EN→FR/DE/IT/ES, RU→SR. Wave 5 picks 5 genuinely-distinct NOT-yet-confirmed features.

| # | Feature | Video | Real output read from output frame (§11.4.107) |
|---|---------|-------|----------------------------------------------|
| 1 | **unified-translator EN→PT (Portuguese)** — new language pair | `helix_translate-en-pt-translate-wave5-20260616-173814.mp4` (9 frames, 4.5s, 790×560 yuv420p) | INPUT English → OUTPUT Portuguese: "O antigo farol erguia-se sozinho na praia rochosa. Sua luz guiava navios seguros de volta para casa, mesmo durante a tempestade. O guarda subia as escadas todas as noites." (start-frame English ≠ output-frame Portuguese) |
| 2 | **unified-translator EN→JA (Japanese)** — new language pair, non-Latin output | `helix_translate-en-ja-translate-wave5-20260616-173814.mp4` (9 frames, 4.5s) | INPUT English → OUTPUT Japanese: "古い灯台は、岩盤の海岸にひとりで立ち並んでいた。灯火は嵐のなかでも船を安全に帰港に導いていた。守り人はいずれの夜も階段を登り、灯台を守っていた。" (start-frame Latin ≠ output-frame Japanese script) |
| 3 | **`-multipass` multi-pass LLM polishing engine** (run with `-model deepseek-chat`) | `helix_translate-multipass-polishing-wave5-20260616-173814.mp4` (16 frames, 8.0s) | Session report Step 4: "Multi-pass Polishing ✅ Success — Duration: 27.016265041s — Polished over 1 pass(es) with deepseek" + polished Spanish output. The 27s is genuine extra LLM polishing work (the engine `pkg/verification` actually ran), not a no-op. |
| 4 | **REST `POST /api/v1/convert/script`** (Cyrillic↔Latin, deterministic) — distinct from CLI `-script` | `helix_translate-rest-convert-script-wave5-20260616-173814.mp4` (6 frames, 3.0s) | Real JSON both directions: Cyrillic→Latin `{"converted":"Sistem treba prevesti","original":"Систем треба превести","target":"latin"}`; Latin→Cyrillic `{"converted":"Ово је тестни текст","original":"Ovo je testni tekst","target":"cyrillic"}` (original ≠ converted) |
| 5 | **REST `POST /api/v1/translate/fb2`** → real translated EPUB | `helix_translate-rest-translate-fb2-wave5-20260616-173814.mp4` (16 frames, 8.0s) | INPUT FB2 (English) → POST → valid EPUB (2016 bytes, mimetype `application/epub+zip`, proper OEBPS) → extracted chapter text REAL Serbian: "Стари свјетионич стоје сам на каменитој обали. Његова светлост водило бродове безбедно кући кроз олују." (input English ≠ output Serbian) |

All 5 output frames were READ (not assumed) and copied into `frames/`. The FB2 output EPUB sample is in `frames/fb2_output_sample.epub`.

## Honest characterizations / findings (§11.4.6 — NOT bluffed)

- **`-multipass` default-model is BROKEN (§11.4.146 finding, real bug):** with the DEFAULT model `gpt-4`, multipass fails — session report shows Step 4 "Multi-pass polishing **failed**, keeping base translation: model 'gpt-4' is not valid for provider 'deepseek'. Valid models: [deepseek-chat deepseek-coder deepseek-v4-flash deepseek-v4-pro]" (the polisher's `VerifiedFactory`→`NewLLMTranslatorWithConfig` `ValidModels` whitelist rejects the default model — same whitelist class as the Wave-1 bridge-translator blocker). The base translation correctly survives. **multipass is confirmed working ONLY when invoked with an explicit valid model** (`-model deepseek-chat`). The default-`gpt-4`-multipass path is a real defect worth a tracked fix; a video of the default path would have shown failure, so it was NOT used for the confirmation.
- **`POST /api/v1/translate/fb2` ignores `target_lang`/`source_lang` form fields:** the handler (`pkg/api/handler.go:84-85`) is HARDCODED `sourceLang=ru, targetLang=sr (Serbian)`. So the endpoint genuinely translates + returns a real EPUB, but always into Serbian Cyrillic regardless of requested target. The FEATURE (FB2→translated-EPUB) is confirmed working; the hardcoded-target is an honest limitation note, not a bluff. Worth a tracked fix to honor the requested target.

## Status.md row updates for the committer window (this agent does NOT commit)

Running headline before wave 5 (per Status.md Rev 14): **43 video-confirmed** (the prompt cited 46; the doc's own re-derivable count is 43 — going by what the doc proves per §11.4.6).

**Net-new confirmations: +5 → new running total 43 + 5 = 48.**

Suggested edits for the committer:
- Add to the Anti-bluff note: "Wave 5 of 2026-06-16 added **5 net-new** — EN→PT translate (`helix_translate-en-pt-translate-wave5-20260616-173814.mp4`), EN→JA translate (`helix_translate-en-ja-translate-wave5-20260616-173814.mp4`), `-multipass` polishing engine (`helix_translate-multipass-polishing-wave5-20260616-173814.mp4`, Step 4 ✅ 27s real polish, requires `-model deepseek-chat`; default `gpt-4` path BROKEN per §11.4.146), REST `POST /api/v1/convert/script` (`helix_translate-rest-convert-script-wave5-20260616-173814.mp4`), REST `POST /api/v1/translate/fb2` → real Serbian EPUB (`helix_translate-rest-translate-fb2-wave5-20260616-173814.mp4`, hardcoded ru→sr target). All ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, ≥6 frames, output frame read). Running total 43 + 5 = **48**."
- Flip the `Video-confirmed` summary cell 43 → **48**.
- Row line 656 `api/handler.go | POST /api/v1/translate/fb2`: set Video-confirmation to `helix_translate-rest-translate-fb2-wave5-20260616-173814.mp4` (note: hardcoded ru→sr target).
- Row line 658 `api/handler.go | POST /api/v1/convert/script`: set Video-confirmation to `helix_translate-rest-convert-script-wave5-20260616-173814.mp4`.
- Add/flip the unified-translator `-multipass` row: Video-confirmation `helix_translate-multipass-polishing-wave5-20260616-173814.mp4` (caveat: requires explicit valid `-model`; default-model path broken — open a §11.4.146 bug item).
- Add the EN→PT and EN→JA language-pair confirmations under the unified-translator DeepSeek translate coverage.

## Suggested new tracker items (committer to file)

1. **BUG (§11.4.146):** `-multipass` with default model `gpt-4` fails (`ValidModels` whitelist rejects it for deepseek/most providers) — multipass silently no-ops (base translation kept). Fix: default the polisher model to a provider-valid model, or bypass the whitelist on the polish path like the `BestModel`/invoke path does.
2. **BUG:** `POST /api/v1/translate/fb2` hardcodes `ru→sr` (`pkg/api/handler.go:84-85`), ignoring `source_lang`/`target_lang` form fields. Fix: read the form fields and pass them through.
