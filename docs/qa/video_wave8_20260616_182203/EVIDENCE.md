# §11.4.153 Real-Use Video Confirmation — Wave 8

**Revision:** 1
**Last modified:** 2026-06-16T18:22:00Z

Run-id: `video_wave8_20260616_182203`. Binary: `build/unified-translator`
(`go build -o build/unified-translator ./cmd/unified-translator`, exit 0, 2026-06-16).
Path: real input → real LLM (default in-process API path, NOT `-use-verifier`) → real output.
Recordings: window-scoped (asciinema `--window-size 100x30`, §11.4.154), `helix_translate-`
prefixed (§11.4.155), wave8-suffixed (no collision with prior 48 helix_translate-* recordings;
none deleted). §11.4.107: source frame ≠ output frame verified (first frame = source only +
in-progress; last frame = real translated output). Each video ffprobe-verified advancing frames.

## Features confirmed (5)

| # | Feature | Provider | Model | Source→Target | Real output (read from last video frame) | Video (mp4) |
|---|---------|----------|-------|---------------|------------------------------------------|-------------|
| 1 | GROQ provider path (distinct from deepseek) | groq | llama-3.3-70b-versatile | EN→ES | "La vieja farola se erguía sola sobre el acantilado rocoso. Cada noche su haz barría el mar oscuro, guiando a las naves perdidas de vuelta a puerto." | helix_translate-groq-en-es-wave8-20260616_182021.mp4 (115 frames, 3.83s) |
| 2 | EN→KO (Korean / Hangul) new pair | deepseek | deepseek-chat | EN→KO | "새벽에 용이 잠든 숨겨진 성을 찾기 위해 용감한 기사가 고대 숲을 가로질렀다." | helix_translate-en-ko-korean-wave8-20260616_182059.mp4 (106 frames, 3.53s) |
| 3 | DE→EN reverse direction (non-EN source) | deepseek | deepseek-chat | DE→EN | "The scientist worked all night in his lab. In the morning, he finally discovered the solution to the problem." | helix_translate-de-en-reverse-wave8-20260616_182101.mp4 (114 frames, 3.80s) |
| 4 | FR→ES (non-English↔non-English) | deepseek | deepseek-chat | FR→ES | "El anciano marinero contaba historias de tormentas y tesoros ocultos en el fondo del mar." | helix_translate-fr-es-pair-wave8-20260616_182111.mp4 (107 frames, 3.57s) |
| 5 | EN→SR Latin script (distinct from RU→SR Cyrillic) | deepseek | deepseek-chat | EN→SR (-script latin) | "U srcu planina živela je mudra stara žena koja je znala tajne svake lekovite biljke." (all-Latin, no Cyrillic) | helix_translate-en-sr-latin-wave8-20260616_182113.mp4 (109 frames, 3.63s) |

Recordings live at `/Volumes/T7/Downloads/Recordings/` (raw, gitignored). Frame stills
committed here as durable in-repo evidence (§11.4.83): `groq_en_es_first.png` (source-only,
LLM in-flight), `groq_en_es_last.png`, `en_ko_last.png`, `de_en_last.png`, `fr_es_last.png`,
`en_sr_latin_last.png`.

## §11.4.6 finding (not a bug)

`unified-translator -h` provider help text lists only `openai, anthropic, zhipu, deepseek,
qwen, gemini` — GROQ is omitted from the help string, BUT groq IS fully wired
(`pkg/translator/llm/groq.go`, factory `llm.go:276-277`, env-var map `main.go:444`,
whitelist `llm.go:69`) and works end-to-end (feature #1 confirmed). The stale `-h` string
is a minor doc-accuracy nit, NOT a functional defect. Valid Groq models (whitelist):
llama-3.1-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b-32768, llama-3.3-70b-versatile,
gemma2-9b-it.

## NO-bluff attestation

All 5 verdicts rest on a real video whose last frame shows a "Translation completed
successfully" line with the real provider AND real target-language output text distinct from
the source. No frozen frame, no LLM error, no empty output, no mocked response.
