# Bridge Phase-2 (No-Local-Runtime) — Execution Plan, Verified Against Current Tree

**Revision:** 1
**Last modified:** 2026-06-15T00:00:00Z
**Status:** DESIGN / READY-FOR-EXECUTION (read-only survey produced this; no production source edited). FACT-cited per §11.4.6.
**Authority:** Operator mandate 2026-06-15 — "Do not use any local running models using llama.cpp, only LLMsVerifier obtained strongest models and created bridge … for direct access to model capabilities!" Locked decisions: **D1** remove local runtimes entirely · **D2** forbid ALL local runtimes (llama.cpp + Ollama) · **R1** Obsolete the SSH/distributed worker path (§11.4.90) · **R2** require API keys everywhere (no offline/local fallback) · **D4/D5** top-1 + in-process fallback chain. Bridge facade = `pkg/bridge` (`BestTranslator`/`Invoke`/`BestModel`/`ListVerified`).
**Supersedes the estimates in:** `docs/design/LOCAL_RUNTIME_REMOVAL.md` (Rev 1) — this doc carries the verified-against-current-tree counts/paths. The companion `docs/design/LLMSVERIFIER_BRIDGE.md` covers the bridge build itself.
**Verified against tree at:** HEAD `a5860b2` (2026-06-15). Every file:line below was confirmed in this read-only pass.

---

## 0. TL;DR for the conductor

- **Bridge EXISTS and builds:** `pkg/bridge/bridge.go` (+ `bridge_test.go`, `export_test.go`, `util.go`, `data/`). `BestTranslator`/`Invoke`/`BestModel`/`ListVerified` are implemented. `cmd/model-bridge` already imports it. **The Phase-0 prerequisite (bridge present) is MET.**
- **Drop-in verdict: GO, with ONE tracked prerequisite gap (P-1, below).** `bridge.BestTranslator` returns `translator.Translator` backed by the SAME `*llm.LLMTranslator` that `llm.NewLLMTranslator` returns today — so the 14 high-level `NewLLMTranslator` call sites redirect cleanly. The gap: two consumers depend on the narrower `llm.LLMClient` interface (which has `GetProviderName()`); `translator.Translator` does NOT expose `GetProviderName()`. Those two need an adapter (P-1).
- **Verified site counts (vs the draft's estimates):**
  | Class | Draft estimate | **Verified (this pass)** |
  |---|---|---|
  | `NewLLMTranslator` non-test call sites | 9 components | **14 call sites** across 12 files |
  | llama.cpp non-test source files | ≈25 sites | **16 files** (≈25 lines) |
  | Ollama non-test source files | ≈13 sites | **~12 files** (token spread; O1–O13 hold, see §1.2) |
  | local-runtime worker configs | 5 | **8 configs** |
  | local-runtime helper scripts | 4 | **15 scripts** in `internal/scripts/` |
  | SSH binaries with API path | "llama-only" | **CONFIRMED llama-only** (grep for `NewLLMTranslator`/`VerifiedFactory`/`bridge.` returned EMPTY) |
- **8-angle verdict: GO with conditions** (R1 operator-confirm distributed/SSH removal = already granted via R1; R2 no-key → honest SKIP, no silent fallback — the bridge already hard-errors with no keys, §1.4 below).

---

## 1. Verified current inventory (FACT, file:line)

### 1.1 The 14 `NewLLMTranslator` redirect call sites (THE primary redirect surface)

`llm.NewLLMTranslator(config) (*LLMTranslator, error)` is defined `pkg/translator/llm/llm.go:217`. Non-test callers (verified this pass):

| # | Site | Component | Current default |
|---|---|---|---|
| 1 | `cmd/cli/main.go:350` | cmd/cli | provider flag (incl. ollama/llamacpp) |
| 2 | `cmd/markdown-translator/main.go:314` | markdown-translator | `deepseek` (API) |
| 3 | `cmd/preparation-translator/main.go:175` | preparation-translator | resolves `*_API_KEY` (API-only already) |
| 4 | `cmd/unified-translator/main.go:469` | unified-translator | API path |
| 5 | `cmd/unified-translator/main.go:513` | unified-translator | second provisioning site |
| 6 | `pkg/api/batch_handlers.go:123` | api batch | provider switch |
| 7 | `pkg/api/batch_handlers.go:235` | api batch | provider switch |
| 8 | `pkg/api/handler.go:778` | api `createTranslator` | provider list |
| 9 | `pkg/coordination/multi_llm.go:171` | coordination | per-provider |
| 10 | `pkg/distributed/coordinator.go:297` | distributed | local translator (R1) |
| 11 | `pkg/distributed/coordinator.go:302` | distributed | local translator fallback (R1) |
| 12 | `pkg/grpc/core_translator.go:320` | grpc | provider routing |
| 13 | `pkg/grpc/core_translator.go:361` | grpc | provider routing (2nd) |
| 14 | `pkg/preparation/coordinator.go:98` | preparation pkg | per-provider |
| 15 | `pkg/verification/polisher.go:124` | verification | per-provider |

> Draft missed #3, #5, #11, #13, #14, #15. (15 rows; #4/#5 same binary, #6/#7 same file, #10/#11 same file, #12/#13 same file → 12 files, 15 call sites.)
> `pkg/challenge_runner/provider_challenges.go:51` references the STRING `TestNewLLMTranslator` in a `go test -run` — NOT a call site; leave or update test name only.

### 1.2 llama.cpp — 16 non-test files (FACT)

`cmd/cli/main.go`, `cmd/ebook-translator/ebook_translator.go`, `cmd/markdown-translator/main.go`, `cmd/ssh-translation/main.go`, `cmd/translate-ssh/main.go`, `cmd/unified-translator/main.go`, `pkg/api/batch_handlers.go`, `pkg/api/handler.go`, `pkg/challenge_runner/provider_challenges.go`, `pkg/distributed/coordinator.go`, `pkg/distributed/pairing.go`, `pkg/grpc/core_translator.go`, `pkg/grpc/proto/translator.pb.go` (regen, do NOT hand-edit), `pkg/translator/llm/llamacpp_provider.go`, `pkg/translator/llm/llamacpp.go`, `pkg/translator/llm/llm.go`.

unified-translator local flag/branch lines (FACT): `case "ssh"` `:304`, `case "llamacpp"` `:306`, `executeSSHTranslation` `:314`, `executeLlamaCppTranslation` `:447`, hardcoded gguf `:391`, `-ssh-host/-ssh-user/-ssh-password/-ssh-port` `:627-630`, `-llama-binary/-llama-model` `:634-635`, `case "ssh"` key-check `:692`, `case "llamacpp"` key-check `:697-699`, help text `:764-772`.

### 1.3 Ollama — O1–O13 (FACT, verified)

| ID | Site (file:line) | Action |
|---|---|---|
| O1 | `pkg/translator/llm/ollama.go` (whole file, `OllamaClient`, `:11434` `/api/generate`) | DELETE (§11.4.124 git-history first) |
| O2 | `pkg/translator/llm/llm.go:34` `ProviderOllama` const | REMOVE |
| O3 | `pkg/translator/llm/llm.go:70` `ValidModels[ProviderOllama]` | REMOVE |
| O4 | `pkg/translator/llm/llm.go` "Ollama/LlamaCpp custom-model warn" arm | REMOVE (both gone) |
| O5 | `pkg/translator/llm/llm.go` `case ProviderOllama:` factory branch (`NewLLMTranslatorWithConfig`, ~`:251-312`) | REMOVE; unknown → error |
| O6 | `cmd/unified-translator/main.go:702` `case "mock", "ollama":` no-key arm | REMOVE `ollama`; `case "ollama"` → error/redirect |
| O7 | `cmd/cli/main.go` `ollama` help + `-disable-local-llms` flag | REMOVE refs; flag removed |
| O8 | `pkg/api/handler.go` `"name":"ollama"` models-list entry | REMOVE |
| O9 | `pkg/api/handler.go` `validProviders` array | REMOVE `ollama` (and `llamacpp`) |
| O10 | `pkg/api/batch_handlers.go` valid-provider switch | REMOVE `ollama`/`llamacpp` arms |
| O11 | `pkg/coordination/multi_llm.go:280-283` Ollama discovery (`OLLAMA_ENABLED`/`OLLAMA_MODEL`) + `disableLocalLLMs` gate (`:62,94,111,139,152`) | REMOVE the whole Ollama block + the `disableLocalLLMs` field+gates |
| O12 | `pkg/challenge_runner/provider_challenges.go` `ollama`/`llamacpp` in challenge set | REMOVE both |
| O13 | `pkg/distributed/coordinator.go:231` `case "ollama","llamacpp":` priority arm | REMOVE/redirect per R1 |

Honest-boundary FACT (§11.4.6 — NOT a removal site): `pkg/storage/redis.go` Ollama reference is a COMMENT documenting the `:`-in-model-id cache-key fix; the fix code is provider-agnostic — KEEP it; reword the comment only.

### 1.4 SSH-remote + distributed (R1) — FACT

- **`cmd/translate-ssh/main.go` + `cmd/ssh-translation/main.go` are llama-ONLY** — grep for `NewLLMTranslator`/`VerifiedFactory`/`executeAPI`/`bridge.` in both returned EMPTY (verified). `ssh-translation` hardwires `Type:"llamacpp"`, `BinaryPath:"/usr/local/bin/llama.cpp"`. → Obsolete (R1).
- `pkg/distributed/coordinator.go` — local arms: `getPriorityForProvider` `case "ollama","llamacpp"` `:231`; local translator provisioning `:297` + fallback `:302`; worker-provider parse comments `:206,215`. Redirect or remove per R1.
- `pkg/distributed/pairing.go:37` — `LocalLLMs []string` ("ollama, llamacpp models") worker field. REMOVE per R1.
- `pkg/grpc/proto/translator.pb.go` provider enum comment lists ssh/ollama/llamacpp → REGENERATE proto (never hand-edit `.pb.go`).
- `version_manager.go` / `ssh_pool.go` are GENERIC (no llama/ollama/gguf refs) — survive if R1=redirect; Obsolete only if R1=wholesale-remove distributed mode.

### 1.5 Worker configs — 8 (FACT, draft listed 5)

`config_ollama.json`, `config.distributed.json`, `config.distributed.thinker.json`, `config.distributed.test.json`, `config.worker.json`, `config.production.template.json`, `config.worker.llamacpp.json`, `config.worker.ollama.json` — all under `internal/working/`, all carry a local-provider/`gguf`/`model_path`/`11434`/`ollama` marker. Audit each: DELETE the worker/distributed local fixtures; for shared templates (e.g. `config.production.template.json`) STRIP the local arm rather than delete.

### 1.6 Helper scripts — 15 (FACT, draft listed 4)

`internal/scripts/`: `batch_translate_llamacpp.sh`, `batch_translate_markdown_llamacpp.sh`, `check_llamacpp.sh`, `install_llamacpp.sh`, `llm_translation.sh`, `monitor_llamacpp_translation.sh`, `setup_llama.py`, `test_distributed_llamacpp.sh`, `test_llama.py`, `translate_llm_only.py`, `translate_markdown_llamacpp_production.sh`, `translate_markdown_llamacpp.sh`, `translate_markdown_multillm.sh`, `translate_with_llama.py`, `translator-ssh-llama`. All `feature-removed` (R1 — local-only tooling).

### 1.7 Test cluster (rewrite/remove under §11.4.43/§11.4.115)

llama: `llamacpp_test.go`, `llamacpp_provider_test.go`, `llamacpp_buildargs_defect_test.go` + arms in `llm_test.go`, `provider_clients_test.go`, `integration_test.go`, `cmd/unified-translator/ssh_command_test.go`, `cmd/cli/main_test.go`, `cmd/markdown-translator/main_test.go`. ollama: `ollama_test.go`, `ollama_options_defect_test.go` + arms in the above + `cmd/translate-ssh/main_test.go`, `pkg/coordination/multi_llm_test.go`/`multi_llm_fault_test.go`, `pkg/distributed/*_test.go`, `pkg/translator/universal_test.go`, `test/unit/coordination_test.go`. KEEP `pkg/storage/*_cachekey_*_test.go` (cache-key tests — reword ollama example only).

---

## 2. Drop-in confirmation + the ONE prerequisite gap

**Interfaces (FACT):**
- `llm.LLMClient` (`pkg/translator/llm/llm.go:211`): `Translate(ctx,text,prompt)(string,error)` + `GetProviderName() string`.
- `translator.Translator` (`pkg/translator/translator.go:36`): `Translate(ctx,text,context)(string,error)` + `TranslateWithProgress(...)` + `GetStats()` + `GetName()`. **No `GetProviderName()`.**
- `bridge.BestTranslator(ctx, task) (translator.Translator, []string, error)` — backed by `VerifiedFactory.CreateTranslatorWithFallback → *llm.LLMTranslator` (`pkg/translator/llm/verified_factory.go:108`). `*LLMTranslator` is the SAME concrete type `NewLLMTranslator` returns (`llm.go:217`).

**Verdict — the 15 `NewLLMTranslator` call sites: clean drop-in.** They consume the high-level `Translate`/`TranslateWithProgress` surface (the `Translator` contract), which `bridge.BestTranslator` satisfies via the identical `*LLMTranslator`. Replace `llm.NewLLMTranslator(config)` with `bridge.BestTranslator(ctx, task)` (default provider becomes the strongest verified model, not a hardcoded string).

**Prerequisite gap P-1 (the ONLY hand-wave-free blocker) — two `llm.LLMClient` consumers:**
- `pkg/markdown/simple_workflow.go:22` — field `LLMProvider llm.LLMClient`.
- `pkg/modelsbridge/bridge.go:29,35` — wraps `llm.LLMClient`, calls `GetProviderName()` (`:43,103`).

`bridge.BestTranslator` returns `translator.Translator`, which lacks `GetProviderName()` — so it is NOT assignable to an `llm.LLMClient` field. **Resolution (tracked work item before R-1 touches these two):** add a tiny adapter exposing `Translate(ctx,text,prompt)` + `GetProviderName()` over the bridge — source the provider name from `bridge.BestModel(ctx,task).ProviderID`. Either (a) `bridge` exposes a `BestClient(ctx,task) (llm.LLMClient, error)` helper, or (b) a thin local adapter in each consumer wraps `BestTranslator`+`BestModel`. Recommended: **(a)** one `bridge.BestClient` helper (single, tested, reused) — keeps the two consumers a clean drop-in and avoids duplicate adapters. This is a small additive bridge method; it is a PREREQUISITE for redirecting those two files, not a blocker for the other 13 sites.

---

## 3. Contention map + parallelization

Removal/redirect steps and the files each touches, ordered so they don't collide (§11.4.58/§11.4.119 — one owner per file). Disjoint-file steps parallelize; same-file steps serialize.

| Step | Files (exclusive owner) | Parallel-safe with |
|---|---|---|
| P-1 (bridge `BestClient` adapter) | `pkg/bridge/bridge.go` (+ test) | everything (additive, no removal) |
| R-1a redirect cmd binaries | `cmd/cli/main.go`, `cmd/markdown-translator/main.go`, `cmd/preparation-translator/main.go`, `cmd/unified-translator/main.go`, `cmd/ebook-translator/ebook_translator.go` | R-1b, R-1c (disjoint files) |
| R-1b redirect api/grpc | `pkg/api/handler.go`, `pkg/api/batch_handlers.go`, `pkg/grpc/core_translator.go` | R-1a, R-1c |
| R-1c redirect pkg coordinators | `pkg/coordination/multi_llm.go`, `pkg/preparation/coordinator.go`, `pkg/verification/polisher.go` | R-1a, R-1b |
| R-1d redirect LLMClient consumers (needs P-1) | `pkg/markdown/simple_workflow.go`, `pkg/modelsbridge/bridge.go` | after P-1; disjoint from R-1a/b/c |
| R-2 remove Ollama | `pkg/translator/llm/{ollama.go,llm.go}`, lists in handler/batch (SHARED with R-1b → SERIALIZE after R-1b), `provider_challenges.go`, `multi_llm.go` (SHARED with R-1c → SERIALIZE after R-1c) | NOT parallel with R-1b/R-1c on shared files |
| R-3 remove llama.cpp | `llamacpp.go`, `llamacpp_provider.go`, `llm.go` (SHARED with R-2 → SERIALIZE), proto regen, `pkg/hardware` (§11.4.124 investigate-first, separate commit) | serialize with R-2 on `llm.go` |
| R-4 SSH/distributed (R1) | `cmd/translate-ssh`, `cmd/ssh-translation`, `pkg/distributed/{coordinator.go,pairing.go}`, worker configs, scripts | distributed files disjoint from R-2/R-3 except `coordinator.go` (already touched in R-1c/R-2 — serialize) |
| R-5 gate + retest | new gate script, docs, `.golangci.yml`, session-resumption file | last |

**Serialization hot files (MUST be single-owner across steps):** `pkg/translator/llm/llm.go` (R-2 + R-3), `pkg/api/handler.go` (R-1b + R-2 lists), `pkg/coordination/multi_llm.go` (R-1c + R-2/O11), `pkg/distributed/coordinator.go` (R-1c + R-2/O13 + R-4). Assign each to ONE agent across its steps, or sequence them.

**What each removal BREAKS** (must be edited in lockstep): the factory switch `case ProviderOllama/ProviderLlamaCpp` (`llm.go`), `validProviders` arrays (`handler.go`, `batch_handlers.go`), proto enum (`translator.pb.go` → regen), `LlamaCppProviderConfig` type (referenced by `cmd/translate-ssh`, `cmd/translator` `:36` — both removed/redirected in R-4), the §11.4.153 feature-ledger rows for llama.cpp/Ollama/SSH (→ §11.4.90 Obsolete, §5), and the test cluster (§1.7).

---

## 4. `CM-NO-LOCAL-RUNTIME` gate + paired §1.1 mutation

**Pre-build gate `CM-NO-LOCAL-RUNTIME`** (generalises bridge-doc `CM-NO-LLAMACPP`):
- **Scans** `cmd/ + pkg/ + internal/` `*.go` and `internal/working/*.json`, **EXCLUDING** submodule trees (`llms_verifier/`, `helix_qa/`, `vision_engine/`), the removal-history test files, and the `pkg/translator/llm/mock.go` unit seam (mocks permitted in unit tests only, §11.4.27).
- **FAILs** on any occurrence of the forbidden set:
  `llama.cpp` · `llamacpp` · `llama-cli` · `ProviderLlamaCpp` · `ProviderOllama` · `OllamaClient` · `LlamaCppProviderConfig` · `:11434` / `11434` · `OLLAMA_ENABLED` · `OLLAMA_MODEL` · `-disable-local-llms` · (R1=A) `executeSSHTranslation` / `translate-ssh` / `ssh-translation` / `LocalLLMs`.
- **JSON config sub-assert:** no tracked `internal/working/*.json` carries `"default_provider":"llamacpp"` | `"ollama"` | `gguf` | `model_path` | `n_gpu_layers` | `11434`.
- **Paired §1.1 mutation:** re-introduce one forbidden token (e.g. re-add the `ProviderOllama` const in `llm.go`) → assert gate FAILs; restore → assert PASS. (gate-code = separate work item.)

**No-key SKIP guard (R2, §11.4.3):** a test asserting that with NO `*_API_KEY` set, every translate entry point returns an honest hard error / SKIP-with-reason — NEVER a local fallback. The bridge ALREADY enforces this (`bootstrapInProcess` returns "no provider API keys set … (local llama.cpp fallback is not permitted)", `bridge.go:158-163`) — the guard pins that behaviour. Paired mutation: re-add a local fallback in any redirected entry point → guard FAILs.

**Per-component §11.4.115 RED-first redirect tests:** per redirected component, `RED_MODE=1` reproduces "component still provisions a local client" on the pre-fix artifact; flipped `RED_MODE=0` is the standing guard asserting it provisions via `bridge.BestTranslator`. Register into the §11.4.135 regression suite.

---

## 5. R2 — require-keys-everywhere map (honest hard-fail, §11.4.69)

Every entry point that must hard-fail (no silent local fallback) when no API key + no verified model is available:

| Entry point | Current no-key behaviour | Post-R2 behaviour |
|---|---|---|
| `pkg/bridge` `Open`/`bootstrapInProcess` | already hard-errors (`bridge.go:158-163`) | unchanged — this IS the canonical honest path |
| `cmd/unified-translator/main.go:702` `case "mock","ollama"` | falls back to mock/ollama no-key arm | drop `ollama`; redirect default to bridge → bridge hard-errors if no key |
| `cmd/cli/main.go:350` | provider+`-disable-local-llms` | bridge path → hard-error if no key; remove local flags |
| `cmd/markdown-translator/main.go:314` | `deepseek` default needs key | bridge path → hard-error if no key |
| `cmd/preparation-translator/main.go:175` | resolves `*_API_KEY`, empty = honest | bridge path → hard-error if no key (already honest) |
| `cmd/ebook-translator/ebook_translator.go` | `PROVIDER:llamacpp` detection | bridge path; remove llamacpp detection |
| `pkg/api/handler.go:778` `createTranslator` | provider list | bridge path → 4xx/honest error if no key |
| `pkg/api/batch_handlers.go:123,235` | provider switch | bridge path → honest error |
| `pkg/grpc/core_translator.go:320,361` | provider routing | bridge path → gRPC error status if no key |
| `pkg/coordination/multi_llm.go:171` | per-provider + Ollama discovery | bridge path; remove Ollama discovery + `disableLocalLLMs` |
| `pkg/preparation/coordinator.go:98`, `pkg/verification/polisher.go:124` | per-provider | bridge path → honest error |
| `pkg/distributed/coordinator.go:297,302` | local translator provisioning | R1: remove (A) or redirect worker body to verified API (B) |

CI with no keys ⇒ integration/e2e/Challenge suites SKIP-with-reason (§11.4.3); unit tests use `mock.go` (§11.4.27). No re-introduced silent local fallback (would re-violate the mandate).

---

## 6. §11.4.122 / §11.4.90 — Obsolete entries to create (operator-authorized via D1/D2/R1)

Each removal is an operator-authorized capability drop (D1 remove local runtimes; D2 forbid both; R1 Obsolete SSH/distributed). Create these §11.4.90 Obsolete tracker entries (status `Obsolete (→ Fixed.md)`), each with `Obsolete-Details` (Since / Reason / Superseding-item=`pkg/bridge` / Triple-check evidence = this doc's §1):

| Item | Reason | Gate |
|---|---|---|
| `cmd/translate-ssh` (llama-only, grep-empty for API path) | `superseded-by-design-change` | R1 |
| `cmd/ssh-translation` (`Type:"llamacpp"` hardwired) | `superseded-by-design-change` | R1 |
| `pkg/translator/llm/llamacpp.go`, `llamacpp_provider.go` | `superseded-by-design-change` | always (after §11.4.124 git-history) |
| `pkg/translator/llm/ollama.go` | `superseded-by-design-change` | always (after §11.4.124 git-history) |
| 8 worker/distributed configs (§1.5) | `feature-removed` | worker/distributed = always; templates = strip-local-arm |
| 15 `internal/scripts/*llama*`/`*llm*`/`multillm`/`translator-ssh-llama` (§1.6) | `feature-removed` | R1 |
| `pkg/hardware` (imported only by `llamacpp.go`) | dead-code candidate | §11.4.124 investigate-FIRST, separate removal-only commit citing git-history |
| feature-ledger rows (§11.4.153) for llama.cpp / Ollama / SSH-worker | `feature-removed` | with their removal commits |

**NOT Obsolete (live non-local consumers — KEEP):** `pkg/models` (imported by `cmd/server`, `handler.go`, `user_auth.go`), `pkg/distributed/version_manager.go` + `ssh_pool.go` (generic; Obsolete only if R1=wholesale distributed removal), `pkg/storage/redis.go` cache-key fix (provider-agnostic), `cmd/model-bridge` (the bridge CLI).

---

## 7. Ordered execution checklist (the implementation stream runs this)

**Pre-req MET:** `pkg/bridge` exists with `BestTranslator`/`Invoke`/`BestModel`/`ListVerified` (verified). 

- **P-0 — Operator gate.** R1 (distributed/SSH remove vs redirect-worker-to-API) + R2 confirm (no-key → SKIP, no silent fallback) — both already granted via D1/D2/R1; confirm R1=A (remove) vs R1=B (redirect) before R-4. (§11.4.66/§11.4.101/§11.4.122.)
- **P-1 — Add `bridge.BestClient(ctx,task) (llm.LLMClient, error)` adapter** (additive, no removal). Tested. Unblocks R-1d. (Closes the only drop-in gap.)
- **R-1 — Redirect (additive; locals still present; builds stay green).** R-1a cmd binaries · R-1b api/grpc · R-1c pkg coordinators (parallel — disjoint files) → then R-1d the two `LLMClient` consumers (after P-1). Each component: §11.4.115 RED-first redirect test.
- **R-2 — Remove Ollama** (§11.4.124 git-history first; separate commit). O1–O13; reword `redis.go` comment; rewrite/remove ollama tests. Serialize on `llm.go`/`handler.go`/`multi_llm.go`/`coordinator.go`.
- **R-3 — Remove llama.cpp** (§11.4.124 + §11.4.122). 16 files; regenerate proto; delete `LlamaCppProviderConfig`; investigate `pkg/hardware` orphaning (separate removal-only commit with git-history evidence).
- **R-4 — SSH/distributed per R1.** (A) Obsolete the 2 SSH binaries + distributed local arms + 8 configs + 15 scripts + `-provider ssh`; OR (B) keep distributed, redirect worker body to verified API (build the API-worker body — does not exist today). Track each as §11.4.90 Obsolete.
- **R-5 — Gate + full retest.** Add `CM-NO-LOCAL-RUNTIME` + paired mutation + no-key SKIP guard + worker-config JSON gate. Update `Documentation/AGENTS.md`, README, `.golangci.yml`, the §11.4.131 session-resumption file. Full-suite retest (§11.4.40). `docs/qa/<run-id>/` transcript for the bridge-backed translate path (§11.4.83).

---

## 8. §11.4.145 8-angle impact GO/NO-GO (re-confirmed)

| Angle | Finding (FACT) | Verdict |
|---|---|---|
| 1 Correctness | Bridge returns the same `*LLMTranslator`; deleted factory branches → unknown-provider errors via existing default arm. | GO |
| 2 Regression-defaults | `config.json default_provider:"openai"` unaffected; 8 worker/distributed configs default local → delete/strip (§1.5). | GO (with config delete) |
| 3 Latent/contract | factory switch, `validProviders`, proto enum, `LlamaCppProviderConfig` consumers all need lockstep edit (§3). | GO (lockstep) |
| 4 Security | removes `exec` of llama-cli + SSH remote-binary install/upload — net RCE/supply-chain reduction. | GO (improves) |
| 5 Performance | all inference → network API; bridge fallback chain mitigates. | GO |
| 6 Host/data safety | removes host subprocess + remote install (§11.4.133-adjacent). | GO (improves) |
| 7 Cross-feature | version_manager generic (survives R1=B); redis cache-key fix kept; `OLLAMA_ENABLED` contract removed. | GO |
| 8 Business-logic | tests assume locals (§1.7); no-key CI → SKIP (R2, bridge already hard-errors). | GO (R2 honest) |

**Overall: GO.** R1 choice (A vs B) is the only operator gate before R-4; both pre-authorized in principle.

## Sources verified 2026-06-15 (§11.4.150 — carried from the companion draft)
- https://www.sitepoint.com/hybrid-cloudlocal-llm-the-complete-architecture-guide-2026/
- https://www.getmaxim.ai/articles/top-5-llm-gateways-in-2025-the-definitive-guide-for-production-ai-applications/
- https://dev.to/akarshc/how-to-test-llm-integrations-in-ci-without-burning-tokens-1ibh
- https://developers.redhat.com/articles/2026/05/27/how-we-built-integration-testing-fast-moving-ai-backend
