# Independent Code Review — Bridge Phase-2 No-Local-Runtime Redirect (P-1 → R-1)

**Revision:** 1
**Last modified:** 2026-06-15T21:01:10Z
**Reviewer:** independent code-review agent (structurally separate from the author, §11.4.142 / §11.4.125)
**Authority:** §11.4.142 (every change reviewed), §11.4.125 (review gate before build), §11.4.134 (iterate-until-GO), §11.4.69 (no silent fallback)
**Scope:** parent repo `/Volumes/T7/Projects/helix_translate`, commits `e5307ce..35cb12c` reviewed as one logical change.

---

## Verdict: **GO** (no BLOCKING findings)

The no-local-runtime redirect is correct, the R2 no-silent-fallback mandate holds on every redirected path, the new tests genuinely catch mis-wiring, the nil-default seams are behaviour-preserving, the deferrals (R-3/R-4) are correctly scoped, and the 3 reported failures are git-proven pre-existing and unrelated. **R-2/R-3 removals may proceed.** Three NIT-level hardening items are listed (none gate the build).

Build green at HEAD (`go build ./cmd/... ./pkg/...` → BUILD_OK). `go vet` clean on all touched packages. All redirect test suites pass at HEAD:
```
ok  pkg/bridge        ok  pkg/coordination   ok  pkg/preparation   ok  pkg/verification
ok  pkg/api (Bridge)  ok  pkg/grpc (Bridge)  ok  cmd/unified-translator (Bridge)
```

---

## Commits reviewed
- P-1 `e5307ce` — `bridge.BestClient` adapter + `Invoke` extract-method refactor
- P-1.5 `86816a4` — `ProviderDiverse*` API + injectable `EnsembleTranslatorFactory` seams
- R-1 `9f6be23` / `0713bec` / `3cd4a77` / `0a3b607` / `7f843fc` / `15cab94` / `35cb12c` — adapters + ~10 single-site + 3 ensemble-site redirects + R2 honest no-key error

---

## Dimension 1 — Correctness — **GO**

- **P-1 `Invoke` refactor is behaviour-preserving (FACT).** `git show e5307ce` shows a pure extract-method: `resolveBest` (bridge.go:382) is the verbatim prior inline selection→resolution→marker logic; `realClientBuild` (bridge.go:313) is the verbatim prior `NewOpenAIClient` call. `Invoke` (bridge.go:425) and the new `BestClient` (bridge.go:413) both route through the single `clientBuild` construction path — exactly ONE construction path, as documented.
- **Bridge opened once per binary, not per-request.** api `Handler.bridgeFor` (handler.go:~790) and grpc `CoreTranslatorImpl.bridgeFor` (core_translator.go) both lazy-open via `sync.Once` (`bridgeOnce.Do`) caching `bridge`+`bridgeErr`; cli/server/markdown/preparation mains `bridge.Open(...)` once in `main`. No per-request leak.
- **`sync.Once` lazy handlers are thread-safe.** `bridge`/`bridgeErr` are written only inside `Once.Do` and read only after `Do` returns — the standard safe `sync.Once` memo pattern. No data race.
- **Provider-diverse adapter genuinely gives one-per-provider, no collapse (FACT).** `ProviderDiverseModels` (bridge.go:455-480) dedups via a `seen map[providerID]bool` over the score-descending `ListVerified`, keeping first-per-provider (= strongest per provider). `FallbackOrder` re-numbered 1..N. `ProviderDiverseClients`/`Translators` build through the same whitelist-immune `clientBuild` seam. Mutation-proven by `TestBridge_ProviderDiverseClients_OnePerProviderRankedByScore` (RED FAILs `got 3, want 5`).
- **`clientTranslator.Translate` threads `contextStr` verbatim** (bridge.go:554-555) — no argument loss. Sole ensemble consumer is `polisher.go:563 translator.Translate(ctx, prompt, location)`, matching the thin surface (FACT).

## Dimension 2 — R2 no-silent-fallback (core mandate) — **GO**

Every redirected default path returns an honest hard error on no-key and CANNOT reach a local runtime:
- **`bridge.Open` is the choke point** (bridge.go:98). No `*_API_KEY` → `bootstrapInProcess` returns the honest error `"no provider API keys set ... (local llama.cpp fallback is not permitted)"` (bridge.go:160-164). No code path constructs llama.cpp/Ollama from `Open`.
- **api** `createTranslator` (handler.go:790) routes everything except `provider=="distributed"` through `bridgeFor()`; the old `llm.NewLLMTranslator(config)` is deleted. `bridgeErr` is cached and surfaced, never masked.
- **grpc** `executeTranslation` switch (core_translator.go:255) — `ssh`→SSH (R-3 deferred), `llamacpp`→explicit operator-chosen local (R-3 deferred), **`default`→`executeAPITranslation`→bridge**. The deleted `llm.NewLLMTranslator` is replaced by `ct.bridgeFor()`. Local llama.cpp reachable ONLY by explicit `ProviderConfig.Type=="llamacpp"` (operator choice, not silent fallback).
- **cli** (main.go) — bridge opened once; multi-LLM via `NewMultiLLMTranslatorWrapperWithFactory(..., b.EnsembleFactory(task))`, single fallback via `b.BestTranslatorFunc(task)(ctx)`; no-key → `return fmt.Errorf("LLMsVerifier bridge unavailable (no local-runtime fallback): %w", ...)`.
- **server** (main.go) — `NewMultiLLMCoordinatorWithFactory(..., b.EnsembleFactory(...))`; no-key → `log.Fatalf("LLMsVerifier bridge unavailable (no local-runtime fallback)")`.
- **markdown-translator** (main.go) — the old `createTranslator` `case "llamacpp"` no-key local arm is DELETED; replaced by `createTranslator(ctx, b, task)`→`b.BestTranslatorFunc`. No-key → `log.Fatalf`.
- **preparation-translator** (main.go) — base translator now `b.BestTranslator(ctx, task)` instead of hardcoded `deepseek` `NewLLMTranslator`; no-key → fatal.
- **unified-translator** (main.go:488) — `executeAPITranslation` default branch → `bridgeTranslator`→`bridge.Open`+`BestTranslator`; `mock` is an explicit offline seam, `ssh`/`llamacpp` arms untouched (R-3/R-4). No-key → `"bridge translation unavailable (no local-runtime fallback)"`.

The remaining `llamacpp`/`Ollama` constructors in the tree (`cmd/unified-translator:307/448`, `pkg/grpc/core_translator.go:259`, `pkg/translator/llm/llamacpp*.go`, `cmd/translate-ssh`) are all reachable ONLY via an EXPLICIT operator provider selection (R-3/R-4 deferred scope) — never from a redirected default. **No silent-fallback path found.**

## Dimension 3 — Test genuineness (§1.1 / §11.4.1) — **GO**

Independent adversarial sub-review (full transcript on file) examined all redirect tests, ran each offline, and verified all six `RED_MODE` polarities genuinely FAIL on the broken branch. Findings:
- **`pkg/bridge/bridge_test.go` — GENUINE.** `invokeDispatch`/`clientBuild` are real package vars on the single production routing path; guards independently recompute the oracle via `BestModel` and assert captured `(providerMarker, modelID)` equals it. `TestBridge_Invoke_RoutesToBestModel` RED proven FAIL (`routed to "deepseek-chat", want "gpt-4o"`). The `BestClient` provider-name pair deliberately uses a **deepseek** strongest model so the bare `*OpenAIClient` `"openai"` would FAIL — load-bearing against the exact hardcoded-openai defect. `TestOpen_NoKeys_HonestHardError` deterministic via empty `Getenv`, asserts the R2 error text + that any "llama" mention is framed "not permitted".
- **`pkg/api` / `pkg/grpc` `bridge_redirect_test.go` — GENUINE.** Inject a fake `bridgeTranslatorFactory`, assert the fake's `BRIDGE:`/`GetName()=="bridge-verified"` output is actually returned (not the old `NewLLMTranslator` path) AND lang pair threaded. No-key tests run the REAL `bridge.Open` with empty `Getenv` → assert `"no provider API keys"`. Distributed-arm test installs a factory that `t.Fatal`s if reached.
- **`cmd/unified-translator/bridge_redirect_test.go` — GENUINE.** The mock-seam IS load-bearing: removing `case "mock"` drops it to the bridge default → real `os.Getenv` → mis-route FAILs the `err==nil` assertion.
- **coordination/preparation/verification `*_factory_test.go` — GENUINE.** RED polarity (factory not injected / set on a discarded struct) proven FAIL on all three; GREEN asserts the factory is reached with the run's ctx (sentinel) and yields the exact injected provider instances.

No tautology that would pass on a wrong route was found.

## Dimension 4 — nil-default seam behaviour-preservation — **GO**

- `MultiLLMCoordinator.translatorFactory` is consumed at exactly ONE branch — `initializeLLMInstances` line 141 `if c.translatorFactory != nil { initializeFromFactory(); return }`. When nil (un-redirected callers, `config.TranslatorFactory` zero value), control falls through to the unchanged `discoverProviders + NewLLMTranslator loop`. Docstring claim "byte-for-byte identical to before" is FACT-accurate.
- Original `NewMultiLLMCoordinator` (line 97) retained with unchanged signature; new `NewMultiLLMCoordinatorWithFactory` (line 129) is additive. The same additive nil-default pattern applies to preparation/verification per the P-1.5 + R-1c diffs and their `_NilFactoryUnchanged` companion tests.

## Dimension 5 — regression blast-radius — **GO**

- Changed signatures (`createTranslator`, `executeAPITranslation`, the `*WithFactory` constructors) verified: callers compile (build green) and the original no-factory constructors are preserved, so un-redirected callers are unbroken.
- **The 3 reported failures are git-proven PRE-EXISTING and unrelated** (independent sub-review, full transcript on file). `git diff --stat 49d94f2..35cb12c` is EMPTY for `internal/verifier/`, `test/distributed/`, `pkg/distributed/`, and the SSH test — the redirect touched none of them, so it definitionally cannot have caused them.
  - **`TestAffirmativeResponseIsHardGate`** — `internal/verifier/pipeline_affirmative_gate_test.go`. The default suite PASSES (`ok ... 0.194s`); it FAILs ONLY under explicit `RED_MODE=1` (by-design §11.4.115 reproduction). The GREEN-default polarity was set by `97a8afd`, an ancestor of baseline `49d94f2`. **CORRECTION to author's framing:** it is NOT "in default-RED mode" — default is GREEN/PASS. Not a regression; just imprecise wording in the handoff.
  - **`test/distributed` race** — `manager_test.go:587-588` unguarded `worker.ActiveJobs++` in the test's own mock; byte-identical at baseline; surfaces only under `-race`.
  - **`test/integration` SSH** — `TestSSHTranslationIntegration` PASSES at HEAD via an in-process mock SSH server; only the e2e variants topology-SKIP. Not a failure at HEAD.

## Dimension 6 — deferred R-1d correctness — **GO**

- `modelsbridge.New` — **zero callers** in `pkg/`/`cmd/`/`internal/` (grep empty). No live consumer left broken; deferring to R-4 is correct.
- markdown `LLMProvider` (`llm.LLMClient` consumer) — the only non-test construction site is `cmd/translate-ssh/main.go:231` which sets it to `nil` ("Will be created remotely"). Non-nil only in the SSH-worker path (R-3/R-4 deferred). No consumer silently broken; deferral correct.

## Dimension 7 — safety — **GO**

- No host/data regression: changes are in-process translator wiring only; no destructive ops, no power-state/host-session calls.
- No secret leakage: `ModelInfo` is key-free (§11.4.10); keys read in-memory via `ProviderResolver`/`SetKeyResolver` and never returned/logged. `bridge_test.go` no-key tests use an empty `Getenv` override (no real keys touched).
- Build green at HEAD; vet clean. History is sequential additive commits (P-1 prerequisite → P-1.5 seams → R-1c threading → per-binary redirects), each replacing a `NewLLMTranslator` site with a bridge call while keeping the original constructors — consistent with green-at-each-commit.

---

## NIT-level hardening (non-blocking — do NOT gate R-2/R-3)

1. **`bridge_test.go:436` `TestBridge_ProviderDiverseTranslators...`** asserts only `len==3` + non-nil; add provider-identity + order assertions (or a `RED_MODE` polarity) to match its `...Clients` sibling. Currently covered transitively via the shared `ProviderDiverseModels` path, so not blocking.
2. **`cmd/unified-translator/bridge_redirect_test.go:34`** — make the mock-seam guard *positively* assert the bridge was not reached (inject a `bridgeOpener` that `t.Fatal`s, mirroring api's distributed-arm test), and soften the env-dependent "live network attempt" docstring.
3. **Docstring trims** — `TestBridge_NumericProviderID_HTTPPath` exercises resolution, not dispatch ("end-to-end at the bridge layer" overstates); and the handoff note for `TestAffirmativeResponseIsHardGate` should be corrected from "default-RED" to "GREEN-default; RED only under RED_MODE=1".

---

## §11.4.134 iterate-until-GO

No BLOCKING findings → the review loop terminates at a clean **GO** this pass. The 3 NITs are tracked here for the author but do not re-arm the loop (they introduce no defect and no false-result/bluff capability). The author may address them opportunistically; they are not preconditions for R-2/R-3 removals.
