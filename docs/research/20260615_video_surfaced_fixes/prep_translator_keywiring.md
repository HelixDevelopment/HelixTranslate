# Deep Research — preparation-translator API-key wiring fix

**Revision:** 1
**Last modified:** 2026-06-15T00:00:00Z
**Authority:** §11.4.150 deep multi-angle web research (run in parallel with the fix per §11.4.150(F))
**Scope:** Fix (1) — `cmd/preparation-translator` CLI/coordinator never read the provider API key from a flag or environment variable; the fix adds a `-api-key` flag, a per-provider `*_API_KEY` env fallback, and a per-provider default model.

## The question

(i) Does reading the provider API key from a CLI flag with a per-provider `*_API_KEY` environment-variable fallback follow best practice / is it the best available solution? (ii) Are we masking a deeper problem — specifically, is per-provider env-key reading consistent with the project's existing `unified-translator` resolver, and does the new flag/env path leak secrets or duplicate logic that should be shared?

## Angle 1 — 12-factor config + CLI precedence convention (external authority)

The Twelve-Factor App "Config" factor mandates storing deployment-varying config — explicitly including API keys — in **environment variables**, not in code or committed config files, because config files get accidentally committed, get scattered, and are framework-specific. Env vars are the universal, language-agnostic standard that stays out of the repo.

The established CLI convention (Cobra/Viper, and Go CLI practice generally) is a strict precedence: **flag > environment variable > config file > built-in default**. A flag explicitly passed on the command line overrides an env var, which overrides a config file, which overrides a hard-coded default.

The fix implements exactly this: `-api-key` flag takes precedence; when absent, fall back to the provider's well-known `*_API_KEY` env var. This is the textbook-correct precedence and the textbook-correct storage location for a secret. Per-provider default model is the "built-in default" tier — correct.

- The Twelve-Factor App — Config — https://12factor.net/config (accessed 2026-06-15)
- Cobra — Configuration Precedence Rules (flag > env > config > default) — https://app.studyraid.com/en/read/11421/357759/configuration-precedence-rules (accessed 2026-06-15)
- Cobra — Building a 12-Factor App with Viper — https://cobra.dev/docs/tutorials/12-factor-app/ (accessed 2026-06-15)

## Angle 2 — Internal consistency with the project's existing resolver (no-deeper-problem check, §11.4.150(C))

The concern flagged in the brief: is per-provider env-key reading *consistent* with `unified-translator`, or did the fix invent a divergent scheme (which would be a latent maintenance/behaviour-mismatch defect)?

Captured FACT from the codebase (`cmd/unified-translator/main.go`):
- Lines 541–554 define the same per-provider env-var map the fix uses: `openai→OPENAI_API_KEY`, `anthropic→ANTHROPIC_API_KEY`, `deepseek→DEEPSEEK_API_KEY`, `zhipu→ZHIPU_API_KEY`, `qwen→QWEN_API_KEY`, `gemini→GEMINI_API_KEY`, plus groq/mistral/xai/cohere/togetherai.
- Line 592: `-api-key` flag default `""`.
- Lines 668–677: flag-first, then "Fall back to the provider's well-known env var … when -api-key was not passed", with a clear error if neither is present.

The prep-translator fix therefore **mirrors the already-shipped unified-translator pattern** (same env-var names, same flag-first-then-env precedence, same clear-error-on-missing). This is consistency-confirmed, not a divergence — it makes the two CLIs behave identically for key resolution. It also matches the env-var names documented in the project CLAUDE.md ("API keys come from environment variables only").

Residual observation (improvement candidate, NOT a defect): the two CLIs now hold two copies of the provider→env-var map. A future refactor could lift the map + resolver into a shared `pkg/` helper (DRY), which would also be the natural place to add the remaining providers (the prep map should be kept in sync with unified's 11-entry map). This is a §11.4.124-style "extend/share, don't duplicate" follow-up, not a blocker for this fix.

Secrets check (§11.4.10): reading from a flag/env into memory and never logging it is the correct handling. The fix must not print the key (the unified-translator error message at line 677 correctly names only the provider, never the value) — the prep fix should follow the same no-echo rule.

- Project source — `cmd/unified-translator/main.go:541-554, 592, 668-677` (captured this session)
- Project governance — `CLAUDE.md` "Configuration": "API keys come from environment variables only — never commit them."

## Finding

**Best-practice-confirmed.** Flag-with-`*_API_KEY`-env-fallback + per-provider default model is the correct, conventional solution (12-factor storage + flag>env>default precedence) and is **consistent with the project's existing unified-translator resolver** (same env-var map, same precedence) — no deeper problem masked. One non-blocking improvement: de-duplicate the provider→env map into a shared helper and keep prep's provider list in sync with unified's. No secret-leak risk provided the key is never logged.

Deep-research 2026-06-15: https://12factor.net/config, https://app.studyraid.com/en/read/11421/357759/configuration-precedence-rules, https://cobra.dev/docs/tutorials/12-factor-app/
