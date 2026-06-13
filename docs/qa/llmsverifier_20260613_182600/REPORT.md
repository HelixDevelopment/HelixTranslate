# LLMsVerifier real-key verification — QA evidence (§11.4.83)

**Run id:** llmsverifier_20260613_182600 · **Date:** 2026-06-13 · **Tested by:** AI-agent (subagent + conductor re-verification)

Keys sourced from `$HOME/api_keys.sh` inline only — **no key value printed, logged, or committed** (§11.4.10). This directory contains only key-free captured server output.

## What ran (mandatory steps)
1. Built the LLMsVerifier submodule (`digital.vasic.llmsverifier`, `go build ./cmd` exit 0).
2. Ran real provider discovery/verification with the operator's real keys (`config_full.yaml`, `${VAR}` env-expansion).
3. Started the verifier REST server; consumed its `/api/health`, `/api/providers`, `/api/models`.
4. Drove the **System's own** `internal/verifier` client against the live server — proving verified models flow end-to-end into the System.

## Real result
- **89 verified models** served (DeepSeek 2, Cerebras 2, Groq 16, Mistral 69), all `status:"verified"`, scores 92–95. See `api_models_head.json`, `api_providers.json`, `api_health.json`.
- Providers authenticated with real keys: **DeepSeek, Cerebras, Groq, Mistral** (others, e.g. OpenAI/Anthropic, have no key in `api_keys.sh` → honest auth failure, correct real behavior — §11.4.6).
- **System client fetched 89 models** end-to-end (after the Bug B fix below).

## Two real bugs found + fixed (mutation-proven §11.4.135 guards)
- **Bug A (submodule `llmverifier/models.go`, commit 95fb1cce):** `context_window` numeric form (Groq) crashed the whole provider decode → 0 models. Fixed via tolerant `UnmarshalJSON`. Guard: `llmverifier/context_window_unmarshal_test.go`.
- **Bug B (main `internal/verifier/client.go`) — load-bearing:** the System's SSOT client could not decode the server's `{"count":N,"models":[...]}` envelope (different field names) → System consumed **zero** verified models while the server reported healthy. Fixed via envelope-aware `decodeModelsResponse` (+ bare-array fallback). Guard: `internal/verifier/client_envelope_test.go` (FAILs with the exact production error when reverted).

## Honest gaps (not bluffed)
- The submodule's `verify` (report) pipeline and the `server` (SQLite DB) have no bridge — the served DB was populated via live discovery using the verifier's own client+database packages. Tracked architectural item.
- Full 30-provider exhaustive scoring of every model was time-bounded (not completed in one pass); per-provider reliability scoring was exercised on the seeded set.
