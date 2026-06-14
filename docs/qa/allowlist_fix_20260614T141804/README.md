# Fix — stale provider model-allowlists (gemini/groq/zhipu) + live-proof (§11.4.107/§11.4.6)

**Revision:** 1
**Last modified:** 2026-06-14T14:20:00Z

The central ValidModels allowlist (pkg/translator/llm/llm.go) only accepted DEPRECATED
models for gemini/groq/zhipu and REJECTED current ones at the LOCAL gate, so those
providers could not even attempt a current model (the request never reached the API).
Added current models (additive — old names kept for back-compat).

## RED → GREEN (gate level, live-proven)
Before: `model 'X' is not valid for provider 'Y'. Valid models: [<stale list>]` (local gate, never hit the API).
After: the current models PASS the gate and REACH the live provider API:

| Provider | Model added | Live API response (proves it reached the provider) | Verdict |
|---|---|---|---|
| groq | llama-3.3-70b-versatile | HTTP 429 "Rate limit reached for model `llama-3.3-70b-versatile`" | model VALID (recognized; TPD quota exhausted today) |
| gemini | gemini-1.5-flash (+1.5-pro/2.0/2.5) | HTTP 400 "API Key not found" | model+gate OK; GEMINI_API_KEY invalid/expired (operator) |
| zhipu | glm-4-flash (+plus/air/...) | HTTP 400 code 1211 "模型不存在" | gate OK; this key's account/endpoint lacks the model (operator/needs zhipu-client investigation) |

## Honest boundary (§11.4.6)
This fix removes the stale-allowlist BLOCKER and is live-proven to let current models
reach the providers. FULL end-to-end translation for gemini/zhipu still depends on valid
credentials/quota/account (operator-side) — NOT claimed as working here. DeepSeek and
Mistral already prove full end-to-end translation (see e2e_format_matrix_* / e2e_multiprovider_*).
A separate follow-up may be warranted: gemini/zhipu CLIENTS may have an endpoint/model-format
issue like the Qwen one fixed earlier (the live APIs reject documented models) — flagged, not
guessed.
