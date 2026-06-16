# /api/v1/providers fix — live sink-side re-validation on nezha (§11.4.108/§11.4.69)

**Run:** 2026-06-16T17:08:28Z · **Endpoint:** `GET https://nezha.local:18443/api/v1/providers` (cmd/server, pkg/api Handler.listProviders)
**Fix under test:** commit `388a2eb` — listProviders serves the REAL configured provider set instead of a hardcoded static {openai,anthropic,zhipu,deepseek} list.
**Image:** rebuilt nezha `helixtranslate:nezha` (08900424e481) via the §11.4.108-fixed `nezha-deploy.sh reboot` (af23440), which recreated ALL app services onto the fresh image.

## Result: PASS (§11.4.69 positive sink-side evidence)

- **19 configured providers** returned: cerebras, cloudflare, cohere, deepseek, gemini, groq, hyperbolic, kimi, mistral, modal, nia, novita, publicai, replicate, sambanova, siliconflow, upstage, vulavula, zhipu.
- Each entry carries the NEW honest fields: `configured:true`, `available`, `requires_api_key`, `description`, `models` — derived from `config.Translation.Providers` + key presence.
- **Strictly different from the old static 4-list** (was exactly {openai,anthropic,zhipu,deepseek} regardless of config). The new response is config-driven: `openai` is correctly ABSENT (no openai key configured on nezha) — pre-fix it was always wrongly present.

## Why this proves the fix (§11.4.6)
The old endpoint returned the same 4 providers regardless of configuration. The live response returns 19 config-driven providers WITH the new configured/available/requires_api_key fields — only possible from the fixed config-reading code path (388a2eb). `openai` absent + 16+ providers present that the old list omitted = the §11.4.108 runtime signature of the fix on the clean deployed artifact.

## Artifacts
- `providers_response.json` — full live JSON
- `analysis.txt` — count + names + new-field + not-old-static-4 checks
