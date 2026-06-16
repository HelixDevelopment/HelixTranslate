# Task 1 — Live re-validation of fix 6a1aa8c on nezha (§11.4.108 PENDING → DONE)

**Run UTC:** 20260616T142656Z
**Main checkout HEAD:** 6a1aa8c
**Live image BEFORE reboot:** f3904ccd71c2 (pre-fix)
**Live image AFTER reboot:** 2d0c925325e3 (contains fix; `IsKnownProvider` gate)
**Surface:** server-TLS `https://nezha.local:18443/api/v1/translate`
**Deploy:** rsync fixed source → nezha:~/helixtranslate, then `bash scripts/nezha-deploy.sh reboot server api-server` (image id changed f3904ccd→2d0c925, runtime signature = behavior below).

## RED baseline (pre-fix binary f3904ccd) — the bug
| Case | HTTP | Body |
|---|---|---|
| `provider:"unsupported-provider"` | **200** (BUG) | silent substitution → `provider:"llm-novita"`, `translated:"Привет, свет"` |
| `provider:"deepseek"` (valid) | 200 | real translation via bridge (`provider:"llm-novita"`) |

## GREEN (post-fix binary 2d0c925) — fix proven, no regression
| Case | HTTP | Body | Verdict |
|---|---|---|---|
| `provider:"unsupported-provider"` | **400** | `{"error":"unsupported provider: unsupported-provider"}` | PASS — bug fixed |
| `provider:"totally-made-up-xyz"` | **400** | `{"error":"unsupported provider: totally-made-up-xyz"}` | PASS — gate variety |
| `provider:"deepseek"` (valid) | **200** | `translated:"Добро јутро, како стаса?"` (real Serbian) | PASS — no regression |
| empty provider (default) | **200** | `translated:"Књига лежи на столу."` (real Serbian) | PASS — no regression |

RED→GREEN polarity flip proven on the clean new artifact (§11.4.115 + §11.4.108 runtime-signature = the live 400/200 behavior).

## Notes
- The `provider` response field reports `llm-novita` because the LLMsVerifier bridge selects the strongest verified model regardless of the (admitted) requested provider name — documented design (handler.go createTranslator comment). The fix gates *admission* (known→pass, unknown→400), not model selection.
- The `server` binary blocks ~3.5 min at startup on `bridge.Open` (5m30s timeout) while selecting verified models from real providers — NOT a crash/regression; it bound successfully and is HEALTHY. Container healthcheck label lags (30s interval) but the endpoint serves 200.
- api-server :18080 returns 404 for `/api/v1/translate` (different route set on that container); server-TLS :18443 is the canonical translate surface.
