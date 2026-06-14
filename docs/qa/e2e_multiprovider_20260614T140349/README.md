# E2E proof — multi-provider + stale model-allowlist audit (§11.4.107 / §11.4.6 / §11.4.150)

**Revision:** 1
**Last modified:** 2026-06-14T14:05:00Z

## Multi-provider breadth — PROVEN
Two distinct real providers translate the same source EN→Serbian Cyrillic end-to-end:
- **DeepSeek** (deepseek-chat) — proven across all 5 input formats (see e2e_format_matrix_*).
- **Mistral** (mistral-small-latest) — PASS, 247 Cyrillic chars:
  `**Врана и крчаг** … Једна жедна врана пронашла је крчаг у којем је на дну било мало воде…`

## Real bug fixed this round (proven)
`cmd/unified-translator` `resolveProviderAPIKey` env-key map was MISSING `zhipu`, so
`ZHIPU_API_KEY` never satisfied the gate ('API key required for provider=zhipu').
Added `"zhipu": "ZHIPU_API_KEY"`. PROOF: after the fix, zhipu passes the gate and
reaches the real Zhipu API (it now returns a model-name error from the live API
instead of the local gate error — gate fixed).

## FLAGGED — systemic stale provider model-allowlists (real, NOT fixed; research-gated §11.4.150)
The per-provider valid-model allowlists in pkg/translator/llm reject CURRENT models
and only accept DEPRECATED ones (a moving target; needs per-provider current-model
research before changing):

| Provider | Allowlist accepts (stale) | Current models it REJECTS | Evidence |
|---|---|---|---|
| gemini | gemini-pro, gemini-pro-vision | gemini-2.0-flash, gemini-1.5-* | "model 'gemini-2.0-flash' is not valid… Valid models: [gemini-pro gemini-pro-vision]" |
| groq | llama-3.1-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b-32768 | llama-3.3-70b-versatile | "Valid models: [llama-3.1-70b-versatile llama-3.1-8b-instant mixtral-8x7b-32768]" |
| zhipu | glm-4, glm-3-turbo | glm-4-flash, glm-4-plus, glm-4-air | factory allows glm-4 but the LIVE API returns code 1211 "模型不存在" (glm-4 no longer exists) |

DeepSeek + Mistral allowlists are current (both translate successfully).
