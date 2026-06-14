# E2E translation proof — DeepSeek EN→Serbian (run e2e_deepseek_20260614T130829)

**Revision:** 1
**Last modified:** 2026-06-14T13:10:00Z

Real end-to-end, real-system, anti-bluff proof (§11.4.83 / §11.4.107) that the
HelixTranslate pipeline translates a real document with a real LLM provider after
this session's 70+ bug fixes — captured during a same-session real run.

## Command (real system, real network)

```
go build -o build/unified-translator ./cmd/unified-translator
# DEEPSEEK_API_KEY sourced from ~/api_keys.sh into the env (never printed/committed — §11.4.10)
./build/unified-translator \
  -i test/assets/crow_and_pitcher_en.txt \
  -o crow_sr.txt \
  -source-lang en -target-lang sr \
  -provider deepseek -model deepseek-chat -timeout 180s
```

Result: `Translation completed successfully | provider=deepseek model=deepseek-chat duration=2.89s` (exit 0).

## What was produced

- A **valid EPUB** (the CLI emits EPUB): `mimetype` (application/epub+zip) + `META-INF/container.xml` + `OEBPS/content.opf` + `OEBPS/toc.ncx` + `OEBPS/chapter1.xhtml`. Raw artifact: `qa-results/e2e_deepseek_20260614T130829/crow_sr.txt` (git-ignored).
- Source (EN): `source_en.txt`
- Translated chapter text extracted from the EPUB (SR Cyrillic): `translated_chapter_sr.txt`
- Provider session report: `session_report.md`

## Anti-bluff verification (§11.4.107)

| Check | Result |
|---|---|
| Real provider call | DeepSeek `deepseek-chat`, 2.89s, exit 0 |
| Output is real Serbian Cyrillic | **205 Cyrillic characters** ("Гавран и крчаг … Жедан гавран наиђе на крчаг…") |
| Differs from source | yes — no English source phrase ("thirsty crow") present |
| No placeholder/bluff text | yes — no TODO/placeholder/"translation failed"/"xlate(" |
| Valid output ebook | yes — well-formed EPUB zip with mimetype + OPF + NCX + chapter |
| No credential leak | yes — `grep` of the API key over all evidence files = clean (§11.4.10) |

Source: `A thirsty crow found a pitcher with a little water…`
→ Translated: `Жедан гавран наиђе на крчаг с мало воде на дну…`

## Provider coverage note (operator action)

`~/api_keys.sh` provides DeepSeek, Gemini, Zhipu, Groq, Mistral, OpenRouter, +~30
more — but **NOT OpenAI or Anthropic**. To prove/enable those providers end-to-end,
add `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` to `~/api_keys.sh`.

This run also surfaced and fixed a real CLI bug: `cmd/unified-translator` only
accepted the `-api-key` flag at its validation gate and ignored the provider
`*_API_KEY` env vars (which `resolveProviderAPIKey` already supported and the docs
advertise) — now the gate falls back to the env var.
