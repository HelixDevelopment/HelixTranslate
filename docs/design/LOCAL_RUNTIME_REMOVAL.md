# Local Runtime Removal — All-Local-Inference (llama.cpp + Ollama + SSH-remote + distributed) Removal & Redirect-to-Bridge Map

**Revision:** 1
**Last modified:** 2026-06-15T00:00:00Z
**Status:** DESIGN / RESEARCH (uncommitted) — READ-ONLY survey, no source edited, no commit. FACT-cited per §11.4.6.
**Authority:** Operator decision 2026-06-15 — FORBID ALL local runtimes (llama.cpp AND Ollama AND SSH-remote-llama.cpp/distributed). Every component uses ONLY LLMsVerifier-obtained (API) strongest verified models via `pkg/bridge.BestTranslator` (bridge built in parallel stream; see `docs/design/LLMSVERIFIER_BRIDGE.md`).
**Scope:** This is the §11.4.145 8-angle impact-research + §11.4.150 deep-research **removal+redirect map** for the implementation stream. It does NOT implement. The bridge doc covers llama.cpp at site granularity; this doc EXTENDS the scope to Ollama + SSH + distributed and supplies the 8-angle GO/NO-GO, the gate plan, the Obsolete list, and the phased checklist.

---

## 0. TL;DR for the conductor

- **Local-runtime non-test source sites: ~46** across three runtimes — **llama.cpp ≈ 25** (19 enumerated in the bridge doc §2 + the 6 `cmd/translate-ssh` / `cmd/ssh-translation` llama-only clusters), **Ollama ≈ 13**, **SSH/distributed-coordination ≈ 8** (provider-routing arms + worker config). Test cluster ≈ 20 files.
- **8-angle verdict: GO with conditions.** The redirect-to-bridge path is sound; the removal is bounded. Two NO-GO-until-resolved risks need operator input (R1 distributed/SSH = whole end-user capability removal under §11.4.122; R2 offline/no-key CI determinism — bridge needs keys, removing local fallback removes the only no-key path).
- **Components needing redirect to `bridge.BestTranslator`: 9** — `unified-translator`, `cmd/server`(+`pkg/api/handler.go createTranslator`), `grpc-server`, `preparation-translator`, `markdown-translator`, `ebook-translator`, `cmd/cli`, `cmd/translator`, `pkg/api/batch_handlers.go`. Plus `pkg/coordination/multi_llm.go` (Ollama discovery arm) and `pkg/distributed/coordinator.go` (local-provider priority arm).
- **Obsolete (§11.4.90) — local-ONLY binaries with no API translate path (FACT): `cmd/translate-ssh`, `cmd/ssh-translation`.** Both translate exclusively via remote llama.cpp; no `executeAPI`/`NewLLMTranslator`/`VerifiedFactory` path exists in either (grep returned empty). Reason `superseded-by-design-change`. Operator pre-approved removal/obsolete via the bridge-doc D2; this doc confirms the FACT that they are local-only.
- **Operator-decision points (surface, do NOT decide):** **R1** — distributed-mode + SSH workers are a distinct end-user capability (§11.4.122): remove entirely vs redirect workers to call the verified API? **R2** — offline/no-key CI: removing local fallback means CI with NO `*_API_KEY` has NO translate path; confirm the honest path is SKIP-with-reason (§11.4.3), never a re-introduced silent local fallback. Both already align with bridge-doc D1/D2/D-ollama — this doc upgrades **D-ollama to a firm REMOVE** per the explicit operator decision.

---

## 1. Per-runtime site inventory (FACT, file:line cited)

### 1.1 llama.cpp (≈25 non-test source sites)

Authoritatively enumerated in `docs/design/LLMSVERIFIER_BRIDGE.md` §2 (19 sites: `llamacpp.go`, `llamacpp_provider.go`, `llm.go:35/71/246/269-270`, `unified-translator` flags+`executeLlamaCppTranslation`, `cmd/cli`, `cmd/markdown-translator`, `cmd/translator`, `cmd/ebook-translator`, `pkg/api/handler.go:573/1814`, `pkg/api/batch_handlers.go`, `pkg/grpc/*`, `pkg/grpc/proto/translator.pb.go`, `pkg/distributed/coordinator.go`+`pairing.go`, `pkg/challenge_runner/provider_challenges.go`, `pkg/report/report_generator.go`). **This doc adds the SSH llama-only cluster** (bridge-doc #8/#9 expanded):
- `cmd/translate-ssh/main.go` — `LlamaConfig llm.LlamaCppProviderConfig` (`:42`), default `BinaryPath:"/usr/local/bin/llama.cpp"` (`:201-202`), `step3TranslateMarkdown` shells/installs remote llama.cpp (`:295-296,647-810` — uploads `setup_llama.py`/`test_llama.py`/`install_llamacpp.sh`, probes `/usr/local/bin/llama.cpp` etc.). **No API translate path.**
- `cmd/ssh-translation/main.go` — workers hardwired `Type:"llamacpp"`, `BinaryPath:"/usr/local/bin/llama.cpp"` (`:133-144`). **No API translate path.**
- `cmd/translator/main.go:36` — `LlamaConfig llm.LlamaCppProviderConfig`.
- Type defs (`llamacpp_provider.go:15-16,47,58,94`): `LlamaCppProviderConfig`, `LlamaCppWorker`, `NewLlamaCppProvider`.

### 1.2 Ollama (≈13 non-test source sites, FACT)

| # | Site (file:line) | Role | Action |
|---|---|---|---|
| O1 | `pkg/translator/llm/ollama.go` (whole file) | `OllamaClient`, `localhost:11434` `/api/generate` | **DELETE** (§11.4.124 git-history first) |
| O2 | `pkg/translator/llm/llm.go:34` | `ProviderOllama` const | **REMOVE** |
| O3 | `pkg/translator/llm/llm.go:70` | `ValidModels[ProviderOllama]` row | **REMOVE** |
| O4 | `pkg/translator/llm/llm.go:245-246` | "Ollama/LlamaCpp custom-model warn" | **REMOVE** (whole arm; both gone) |
| O5 | `pkg/translator/llm/llm.go:267-268` | `case ProviderOllama:` factory branch | **REMOVE**; unknown → error |
| O6 | `cmd/unified-translator/main.go:46,618,702-703,743` | provider doc + `case "mock","ollama"` no-key arm | **REMOVE** ollama from doc/help; `case "ollama"` → error/redirect |
| O7 | `cmd/cli/main.go:669,679,721-722` | `ollama` help + `-disable-local-llms` flag + example | **REMOVE** ollama refs; `-disable-local-llms` becomes no-op/removed |
| O8 | `pkg/api/handler.go:567-571` | `"name":"ollama"` in models list | **REMOVE** entry |
| O9 | `pkg/api/handler.go:1814` | `validProviders` includes `ollama` | **REMOVE** from array |
| O10 | `pkg/api/batch_handlers.go:122,234` | `case ... "ollama" ...` valid-provider switch | **REMOVE** `ollama` (and `llamacpp`) arm |
| O11 | `pkg/coordination/multi_llm.go:278-283` | Ollama discovery (`OLLAMA_ENABLED`/`OLLAMA_MODEL`) | **REMOVE** the Ollama provider block + `disableLocalLLMs` gate |
| O12 | `pkg/challenge_runner/provider_challenges.go:32` | `ollama`/`llamacpp` in challenge provider set | **REMOVE** both |
| O13 | `pkg/distributed/coordinator.go:231` | `case "ollama","llamacpp":` priority arm | **REMOVE/redirect** (see R1) |

Honest-boundary FACT (§11.4.6, NOT a removal site): `pkg/storage/redis.go:303-304` is a **comment** documenting the Ollama `:`-in-model-id cache-key collision fix; the fix code stays (it is provider-agnostic length-prefixing). Do NOT remove the fix; the comment may be left or reworded.

### 1.3 SSH-remote + distributed (≈8 coordination sites, FACT)

- `cmd/unified-translator/main.go:304-305,692-694,627` — `case "ssh"` translate branch + `-ssh-*` flags + key-check arm. **Redirect or remove per R1.**
- `pkg/distributed/coordinator.go:206-245` — provider discovery (parses worker `providers` incl. `ollama`/`llamacpp`) + `getPriorityForProvider` local arm (`:231`). **Redirect/remove per R1.**
- `pkg/distributed/pairing.go:37` — `LocalLLMs []string` ("ollama, llamacpp models") worker field. **REMOVE** field per R1.
- `pkg/grpc/proto/translator.pb.go:199` + `pkg/grpc/{core_translator.go,server.go}` — provider enum comment lists `ssh`/`ollama`/`llamacpp`. **REGENERATE proto** after enum edit (do NOT hand-edit `.pb.go`).
- Worker configs `internal/working/config.worker.llamacpp.json` + `config.worker.ollama.json` — both `"default_provider":"llamacpp"` with `model_path`/`n_gpu_layers`/`gguf` (FACT). **DELETE** (local-only fixtures). `config.distributed*.json` reviewed per R1.
- `tests/websocket_monitoring_test.go:768` — `Type:"ssh-llamacpp"` worker fixture. **REWRITE/REMOVE** per R1.

**version_manager is GENERIC (FACT):** `pkg/distributed/version_manager.go` grep for `llama|ollama|gguf|local-model` returned EMPTY — it is binary-version-sync/rollback machinery, NOT llama-bound. It survives if distributed mode is redirected (R1 = redirect); it becomes Obsolete only if distributed mode is removed wholesale.

### 1.4 Test cluster (remove/rewrite under §11.4.43/§11.4.115)

llama: `llamacpp_test.go`, `llamacpp_provider_test.go`, `llamacpp_buildargs_defect_test.go` + arms in `llm_test.go`, `provider_clients_test.go`, `integration_test.go`, `cmd/unified-translator/ssh_command_test.go`, `cmd/cli/main_test.go`, `cmd/markdown-translator/main_test.go`.
ollama: `ollama_test.go`, `ollama_options_defect_test.go` + arms in `llm_test.go`, `provider_clients_test.go`, `cmd/cli/main_test.go`, `cmd/markdown-translator/main_test.go`, `cmd/translate-ssh/main_test.go`, `pkg/coordination/multi_llm_test.go`+`multi_llm_fault_test.go`, `pkg/distributed/coordinator_test.go`+`coordinator_translate_race_test.go`+`pairing_query_success_test.go`+`p6_bench_stress_test.go`, `pkg/translator/universal_test.go`, `pkg/storage/storage_w2{0,1}_cachekey_*_test.go` (cache-key tests — KEEP, rephrase ollama example only), `test/unit/coordination_test.go` (Ollama discovery tests).

---

## 2. Per-component redirect-to-bridge map

For each component currently able to select a local provider, the exact route is: replace the `NewLLMTranslator(config)` / `executeLlamaCppTranslation` / `executeSSHTranslation` provisioning with `bridge.BestTranslator(ctx, task)` (returns the strongest verified API model + fallback chain). Default provider becomes the bridge's strongest verified model, NOT a hardcoded provider string.

| Component | Current local hook | Redirect |
|---|---|---|
| `cmd/unified-translator` | `executeProviderTranslation` switch `ssh`/`llamacpp` (`:303-311`); default `-provider openai` | Default path → `bridge.BestTranslator`; delete `case "ssh"`/`case "llamacpp"`; `executeAPITranslation` becomes the bridge-backed path |
| `cmd/server` + `pkg/api/handler.go` | `createTranslator` (handler.go:722-762); provider list/`validProviders` | `createTranslator` → `bridge.BestTranslator`; drop ollama/llamacpp from lists |
| `grpc-server` (`pkg/grpc/*`) | provider-string routing incl. ssh/local | route via bridge; regenerate proto enum |
| `preparation-translator` | `llm.NewLLMTranslator` (main.go:175), default deepseek | swap to `bridge.BestTranslator` (keeps API-only already; no local path today) |
| `cmd/markdown-translator` | `llm.NewLLMTranslator` (`:314`), default `deepseek`, `case "llamacpp"` (`:286`) | `bridge.BestTranslator`; remove `llamacpp` arm |
| `cmd/ebook-translator` | `PROVIDER:llamacpp` detection (`:321`) | `bridge.BestTranslator`; remove llamacpp detection arm |
| `cmd/cli` | `-provider ... ollama/llamacpp`, `-disable-local-llms` | `bridge.BestTranslator`; remove local flags/help/examples |
| `cmd/translator` | `LlamaConfig` (`:36`) | `bridge.BestTranslator`; remove `LlamaConfig` |
| `pkg/api/batch_handlers.go` | valid-provider switch (`:122,234`) | drop ollama/llamacpp; route via bridge |
| `pkg/coordination/multi_llm.go` | Ollama discovery arm (`:278-283`) | remove Ollama block; verified providers only |
| `pkg/distributed/coordinator.go` | local-provider priority/discovery | per R1 (remove or redirect to verified API) |

**`pkg/models` vs `pkg/hardware` (§11.4.124 dead-code FACT):**
- `pkg/models` is imported by `cmd/server/main.go:17`, `pkg/api/handler.go:11`, `pkg/security/user_auth.go:12` — **NOT dead** after local removal. KEEP.
- `pkg/hardware` is imported ONLY by `pkg/translator/llm/llamacpp.go` (non-test). After llama removal it is a dead-code candidate — but **investigate via `git log -S`/`-G` + check reflection/build-tag/codegen refs BEFORE deletion** (§11.4.124); separate removal-only commit citing git-history evidence. Do NOT delete on sight.

---

## 3. §11.4.145 8-angle impact GO/NO-GO

| Angle | Finding (FACT/evidence) | Verdict |
|---|---|---|
| 1. Correctness | Removing local providers removes valid factory branches; `NewLLMTranslatorWithConfig` `case ProviderOllama/ProviderLlamaCpp` deleted → unknown-provider must error cleanly (existing default arm). Bridge returns a real verified client. | **GO** |
| 2. Regression — defaults | `config.json default_provider:"openai"` (API) — unaffected. **BUT** worker configs `config.worker.{llamacpp,ollama}.json` default to `llamacpp` (FACT) — these break post-removal → DELETE them. `markdown-translator` default `deepseek` (API) OK. `preparation` API-only OK. Each component's default MUST become the bridge strongest model. | **GO (with worker-config delete)** |
| 3. Latent/contract | Factory switch callers, `validProviders` arrays (handler.go:1814, batch_handlers.go:122/234), proto enum (`translator.pb.go:199`) all enumerate the local providers — every consumer must be edited in lockstep or a request naming `ollama`/`llamacpp` falls through to a now-error path. `LlamaCppProviderConfig` type referenced by `cmd/translate-ssh` + `cmd/translator` — removing the type breaks their compile (handled by their own removal/redirect). | **GO (lockstep edit required)** |
| 4. Security | Net-positive: removes subprocess `exec` of `llama-cli` (`llamacpp.go`) + remote binary upload/install over SSH (`translate-ssh` uploads+runs `install_llamacpp.sh`) — reduces RCE/supply-chain surface. Bridge keeps keys in-env only (§11.4.10). | **GO (improves)** |
| 5. Performance | Removes on-device GPU/CPU inference option; all inference becomes network API calls (latency + rate-limit dependent). Bridge has score-descending fallback chain (mitigation). No regression for the already-API default path. | **GO** |
| 6. Host/data safety | Removes host subprocess spawning + remote-host binary install (§11.4.133-adjacent risk reduction). No data-loss path. | **GO (improves)** |
| 7. Cross-feature | `config.json`/worker configs, `pkg/distributed` version_manager (generic, survives), cache-key fix in `redis.go` (KEEP — provider-agnostic). `OLLAMA_ENABLED` env contract removed (coordination + tests). | **GO** |
| 8. Business-logic | Tests/Challenges DO assume local providers: `test/unit/coordination_test.go` Ollama-discovery tests, `tests/websocket_monitoring_test.go` ssh-llamacpp fixture, llamacpp/ollama unit tests. §11.4.50 determinism: bridge needs `*_API_KEY` → **offline/no-key CI has NO translate path after local removal** (R2). | **CONDITIONAL — R2** |

**Overall: GO with two flagged risks (R1, R2) requiring operator input before Phase "remove".**

### Unmitigated risks needing operator input

- **R1 — distributed/SSH = whole end-user capability (§11.4.122).** `cmd/translate-ssh`, `cmd/ssh-translation`, distributed mode, and `-provider ssh` are a distinct shipped capability (remote-worker translation). Removing them is a §11.4.122 component removal → MUST be operator-confirmed. Choice: **(A)** remove entirely (Obsolete `feature-removed`) vs **(B)** keep distributed mode but redirect workers to call the verified API (version_manager + ssh_pool survive, only the llama-exec worker body is replaced). FACT: both SSH binaries are llama-ONLY today, so (A) deletes them; (B) requires building an API-worker body that does not exist yet.
- **R2 — offline/no-key CI determinism (§11.4.3 + §11.4.50 + §11.4.98).** Removing local fallback removes the only translate path that needs no API key. Deep research (§11.4.150, sources below) confirms the correct API-only pattern: tests that need a real model run only when keys are present and **SKIP-with-reason** otherwise (§11.4.3) — never a silent re-introduced local fallback (which would re-violate the operator decision). Confirm: the integration/e2e/Challenge suites adopt SKIP-when-no-key; unit tests use the existing `pkg/translator/llm/mock.go` (mocks permitted in unit tests only per §11.4.27).

---

## 4. §11.4.150 deep-research (≥2 cited latest-source angles)

**Angle A — API-only architecture + graceful degradation (no local fallback):** Current (2025–2026) production practice for API-only LLM systems replaces *local* fallback with an **intelligent routing layer**: provider/model fallback chains + circuit breakers + token-bucket rate-limiting, returning a user-friendly degraded message rather than crashing. This is exactly the bridge's score-descending fallback chain (`buildFallbackChain`, bridge-doc §4.2) — i.e. removing local runtimes and relying on a verified-provider fallback chain is the documented best practice, NOT a missing-feature risk. ([sitepoint Hybrid Cloud-Local LLM Architecture Guide 2026](https://www.sitepoint.com/hybrid-cloudlocal-llm-the-complete-architecture-guide-2026/), [getmaxim LLM Gateways 2026](https://www.getmaxim.ai/articles/top-5-llm-gateways-in-2025-the-definitive-guide-for-production-ai-applications/), [truefoundry rate-limiting AI agents](https://www.truefoundry.com/blog/rate-limiting-ai-agents-preventing-llm-api-exhaustion))

**Angle B — CI testing of LLM-dependent code without keys/offline (the R2 risk):** The documented pattern is NOT a local model fallback but a **test mode that skips the external provider call** while exercising everything else, plus **record/replay** for deterministic real-behaviour without per-run API spend, plus **keys via CI secrets/env only**. Maps directly to: SKIP-with-reason when no key (§11.4.3), `mock.go` for unit contracts (§11.4.27), and real-provider integration tests gated on key presence. The honest handling of R2 is therefore SKIP-with-reason — confirmed by external precedent, not a silent local fallback. ([dev.to testing LLM integrations in CI without burning tokens](https://dev.to/akarshc/how-to-test-llm-integrations-in-ci-without-burning-tokens-1ibh), [Red Hat integration testing fast-moving AI backend](https://developers.redhat.com/articles/2026/05/27/how-we-built-integration-testing-fast-moving-ai-backend))

Conclusion: no bigger hidden problem — the design (verified-provider fallback chain + SKIP-when-no-key CI) matches current best practice. The only genuine trade-off is "no offline inference," which IS the operator's explicit decision.

---

## 5. Obsolete list (§11.4.90 — operator-gated removal)

| Binary/asset | FACT | Reason | Gate |
|---|---|---|---|
| `cmd/translate-ssh` | llama-only (no API translate path; grep empty) | `superseded-by-design-change` | R1(A) |
| `cmd/ssh-translation` | llama-only (`Type:"llamacpp"` hardwired) | `superseded-by-design-change` | R1(A) |
| `internal/working/config.worker.llamacpp.json` | `default_provider:llamacpp`+gguf | `feature-removed` | always (local fixture) |
| `internal/working/config.worker.ollama.json` | `default_provider:llamacpp`+gguf | `feature-removed` | always |
| `internal/working/config.distributed*.json` | distributed-worker fixtures | `feature-removed` | R1(A) |
| `pkg/translator/llm/llamacpp.go`, `llamacpp_provider.go`, `ollama.go` | impl files | `superseded-by-design-change` | always (after §11.4.124 git-history) |
| `pkg/hardware` | imported only by `llamacpp.go` | dead-code candidate | §11.4.124 investigate-first |
| `internal/scripts/{setup_llama.py,test_llama.py,install_llamacpp.sh,translate_llm_only.py}` | uploaded by translate-ssh | `feature-removed` | R1(A) |

`pkg/models`, `pkg/distributed/version_manager.go`, `ssh_pool.go`, `pkg/storage/redis.go` cache-key fix: **NOT Obsolete** (live non-local consumers).

---

## 6. Gate plan — `CM-NO-LOCAL-RUNTIME` (§1.1 paired-mutation)

Generalises the bridge-doc `CM-NO-LLAMACPP` to cover all three runtimes:
- **Pre-build gate `CM-NO-LOCAL-RUNTIME`:** grep the authoritative source tree (`cmd/` + `pkg/` + `internal/`, EXCLUDING submodules `llms_verifier/ helix_qa/ vision_engine/`, EXCLUDING removal-test files + the `mock.go` unit seam) for the forbidden set: `llama.cpp` `llamacpp` `llama-cli` `ProviderLlamaCpp` `ProviderOllama` `OllamaClient` `:11434` `OLLAMA_ENABLED` `LlamaCppProviderConfig`, and (if R1=A) `executeSSHTranslation`/`translate-ssh`. Any NEW occurrence → FAIL.
- **Paired §1.1 mutation:** re-introduce one forbidden usage (e.g. a `ProviderOllama` const) → assert gate FAILs; restore → assert PASS.
- **Per-component RED-first redirect tests (§11.4.115):** for each redirected component, a `RED_MODE=1` test reproducing "component still provisions a local client" on the pre-fix artifact, flipped `RED_MODE=0` to the standing guard asserting it provisions via `bridge.BestTranslator` (verified API). Registered into the §11.4.135 regression suite.
- **No-key SKIP guard (§11.4.3):** a test asserting that with NO `*_API_KEY` present every translate entry point returns an honest SKIP-with-reason / "API key required" error — NEVER a local fallback (proves R2 handled honestly; paired mutation re-adds a local fallback → gate FAILs).
- **Worker-config gate:** assert no tracked config carries `"default_provider":"llamacpp"` / `"ollama"` / `gguf` / `model_path`.

---

## 7. Phased ordered implementation checklist (the implementation stream executes this)

**Pre-req:** `pkg/bridge` exists with `BestTranslator` (parallel stream, bridge-doc Phases 0–2). Do not start Phase R-3 until `bridge.Open`+`BestTranslator` build green.

- **Phase R-0 — Operator gate (BEFORE any removal).** AskUserQuestion R1 (distributed/SSH remove-vs-redirect) + R2 confirm (SKIP-when-no-key, no silent fallback) + confirm D-ollama=REMOVE. (§11.4.122/§11.4.66/§11.4.101).
- **Phase R-1 — Redirect components (additive, locals still present, builds green).** Route the 9 components (+coordination) through `bridge.BestTranslator`; make it the default provisioning path. Each component independently buildable. §11.4.115 RED-first redirect test per component.
- **Phase R-2 — Remove Ollama (§11.4.124 git-history first; separate commit).** O1–O13: delete `ollama.go`, const/ValidModels/warn-arm/factory-branch, lists (handler/batch), coordination arm, challenge set, distributed arm; remove `-disable-local-llms`/`OLLAMA_ENABLED` contract; rewrite/remove ollama tests; reword `redis.go` comment.
- **Phase R-3 — Remove llama.cpp (§11.4.124 + §11.4.122).** Bridge-doc §2 sites #1–#7,#10–#17,#19 + `LlamaCppProviderConfig` type; regenerate proto; delete worker configs; investigate `pkg/hardware` orphaning (separate removal-only commit with git-history evidence).
- **Phase R-4 — SSH/distributed per R1.** (A) Obsolete `cmd/translate-ssh`+`cmd/ssh-translation`+distributed fixtures + the `internal/scripts/*llama*` set + `-provider ssh` + distributed local arms; OR (B) redirect workers to verified API (build API-worker body; keep version_manager/ssh_pool). Track each removal as §11.4.90 Obsolete.
- **Phase R-5 — Gate + full retest.** Add `CM-NO-LOCAL-RUNTIME` + paired mutation + no-key SKIP guard + worker-config gate. `config.json` already API-default (no change); update `Documentation/AGENTS.md`, README, `.golangci.yml` if needed, the §11.4.131 session-resumption file. Full-suite retest (§11.4.40). docs/qa/<run-id>/ transcript for the redirected translate path (§11.4.83).

---

## 8. Catalogue-check (§11.4.74) + cross-refs

- `Catalogue-Check: reuse digital.vasic.translator/pkg/bridge` (parallel stream) — every redirect calls `BestTranslator`, no reimplementation.
- `Catalogue-Check: extend vasic-digital/LLMsVerifier` — verified-model source unchanged.
- Composes with bridge-doc §2 (llama.cpp sites), §4 (bridge API), §7 (`CM-NO-LLAMACPP` → generalised here to `CM-NO-LOCAL-RUNTIME`), §9 (D1/D2/D-ollama → R1/R2 + firm D-ollama=REMOVE here).

## Sources verified 2026-06-15
- https://www.sitepoint.com/hybrid-cloudlocal-llm-the-complete-architecture-guide-2026/
- https://www.getmaxim.ai/articles/top-5-llm-gateways-in-2025-the-definitive-guide-for-production-ai-applications/
- https://www.truefoundry.com/blog/rate-limiting-ai-agents-preventing-llm-api-exhaustion
- https://dev.to/akarshc/how-to-test-llm-integrations-in-ci-without-burning-tokens-1ibh
- https://developers.redhat.com/articles/2026/05/27/how-we-built-integration-testing-fast-moving-ai-backend
