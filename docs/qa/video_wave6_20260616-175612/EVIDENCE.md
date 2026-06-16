# §11.4.153 Real-use Video Evidence — Wave 6 (20260616-175612)

Non-committing parallel stream. Real prompt → real DeepSeek LLM via default
in-process direct CLI path (NOT -use-verifier) → real result. Window-scoped
(terminal pane = CLI surface under test, §11.4.154). Filename prefix
`helix_translate-` (§11.4.155). §11.4.107 liveness: each clip ffprobe-verified
(non-zero duration + advancing frames) AND frame-extracted + content-READ
(English input frame ≠ translated output frame; no frozen/error/empty output).

Provider: deepseek (DEEPSEEK_API_KEY set, never printed). All clips 790×560 yuv420p.

## 5 NET-NEW DISTINCT features CONFIRMED this wave

| # | Feature | Video | ffprobe | Output read (real translation) |
|---|---------|-------|---------|--------------------------------|
| 1 | HTML→HTML round-trip (EN→ES) | helix_translate-html_html_roundtrip-wave6-20260616-175612.mp4 | 8.0s / 240 frames | Spanish HTML: "la farola guiaba a los barcos a través de la noche neblinosa. Su haz iluminaba las oscuras olas cada pocos segundos." (valid <html> output) |
| 2 | PDF input accepted (EN→ES) | helix_translate-pdf_input-wave6-20260616-175612.mp4 | 4.37s / 131 frames | from "PDF document, version 1.7": "El bosque se hizo silencioso a la caída del sol. Los pinos altos susurraban mientras el frío viento se deslizaba entre sus ramas." |
| 3 | DOCX input accepted (EN→ES) | helix_translate-docx_input-wave6-20260616-175612.mp4 | 8.03s / 241 frames | from "Microsoft Word 2007+": "El Puerto / Los barcos de pesca regresaron al puerto al atardecer. Gaviotas volaban sobre las redes llenas de peces plateados." |
| 4 | Language pair EN→ZH (Chinese) | helix_translate-en_zh_translate-wave6-20260616-175612.mp4 | 4.3s / 129 frames | Chinese: "一列列车在黎明时分穿越广阔的平原。农民们从金黄的田野中挥手道别，目送列车驶向远方的城市。" |
| 5 | Language pair EN→RU (Russian) | helix_translate-en_ru_translate-wave6-20260616-175612.mp4 | 4.2s / 126 frames | Russian: "Библиотека стояла тихо под дождем. Старые книги занимали деревянные полки, храня века забытых историй." |

Recordings dir: /Volumes/T7/Downloads/Recordings/
Verification frames: ./frames/<feature>_{start,end}.png

## Real bug findings (drafted, NOT confirmed-as-working)

### MINOR-W6-1: HTML/EPUB chapter-title duplication on conversion output
- **Where:** unified-translator HTML output (and likely EPUB output) — bookToString
  emits the chapter Title then sections that ALSO begin with the title, producing a
  duplicated title line in output (visible in HTML clip: "La FarolaLa Farola").
- **Severity:** cosmetic (Bug). Core translation correct; only the title line repeats.
- **Repro:** `unified-translator -i in.html -o out.html -source-lang en -target-lang es -provider deepseek`
  → output <body> has <p>La Farola</p><p>La FarolaLa Farola</p> (title once as heading,
  again concatenated into first section).
- **Status:** PENDING_FORENSICS (root cause = bookToString concatenating Title into
  Section.Content for single-section chapters; needs §11.4.102 systematic-debug). NOT
  blocking the 5 confirmed features — the translated text itself is correct & complete.

## Honest non-confirmations this wave
- REST POST /api/v1/translate/batch was NOT recorded: the REST createTranslator path
  routes through the LLMsVerifier bridge (bridge.Open), which requires runtime model
  verification; the task notes localhost:8080 is unreachable and prioritised the
  reliable default direct path. Deferred to a future wave with the live-server bridge.
