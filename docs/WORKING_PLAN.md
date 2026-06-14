# HelixTranslate — Working Plan: Unfinished Items & Known Issues

**Revision:** 1
**Last modified:** 2026-06-14T18:10:00Z
**Authority:** operator mandate 2026-06-14 ("what is left unfinished and what are (known) issues … nothing left unfinished with zero issues when we finally finish")
**Scope:** the complete, honest (§11.4.6) inventory of everything not-done + every known issue, structured as a subagent-driven execution plan. Each item carries: WHAT, WHY-OPEN/BLOCKER, EVIDENCE, SUBAGENT-TASK, ACCEPTANCE (rock-solid proof per §11.4.123, no bluff).

**Baseline at authoring:** HEAD `5b0dd17`; `go build ./...` clean; full `go test ./... -p 1` GREEN (57 ok / 0 FAIL at HEAD daa2f70; HEAD 5b0dd17 adds only llm.go fixes — `pkg/distributed/TestSSHPool_cleanup` is a confirmed full-sweep-load flake, passes `-count=3` isolated, §11.4.7). 27 genuine mutation-proven bug fixes landed this session across 10 parallel subagent waves; the main-module product bug-hunt is **saturated** per the §11.4.118 completeness-critic audit.

Status legend: `OPEN` · `OPERATOR-BLOCKED` (needs credentials/decision) · `DESIGN` (needs architecture decision) · `AUTONOMOUS` (a subagent can close it now) · `DONE`.

---

## P0 — Release blockers (must close before any release tag)

### P0.1 — Version string inconsistency `[AUTONOMOUS]`
- **WHAT:** Authoritative version is ambiguous. `VERSION`=`2.3.0`; `cmd/grpc-server` & `cmd/unified-translator` `appVersion`=`3.0.0`; `cmd/translator` `appVersion`=`2.1.0`; CLAUDE.md says treat `VERSION` (2.3.0) as authoritative, Makefile references 3.0.0.
- **EVIDENCE:** `cat VERSION`=2.3.0; `grep appVersion cmd/*/main.go` → 3.0.0/3.0.0/2.1.0.
- **SUBAGENT-TASK:** Reconcile every binary's `appVersion` to the single authoritative `VERSION` value (read `VERSION` at build time or a shared `pkg/version` constant); remove the hardcoded divergent literals. RED test asserting each `appVersion == VERSION`.
- **ACCEPTANCE:** all `appVersion` literals equal `VERSION`; mutation (revert one) → test FAILs. **NOTE:** the authoritative number (2.3.0 vs 3.0.0) is an **operator decision** — see P1.0; the subagent wires the single-source-of-truth, operator picks the value.

### P0.2 — Full §11.4.40 release retest not yet run `[OPERATOR-BLOCKED on devices/topology]`
- **WHAT:** Release requires the complete §11.4.40 7-step retest (pre-build sweep, post-build sweep, on-device cycle, meta-test mutation sweep, Challenge bank sweep, Issues/Fixed audit, CONTINUATION sync) on a clean baseline. We have run repeated full `go test ./... -p 1` sweeps (green) but not the full 7-step ritual.
- **SUBAGENT-TASK:** none yet — gated on P4 (gates), P6 (Challenges/HelixQA), and the §11.4.151 release-prefix tagging. Sequenced AFTER P4/P6.
- **ACCEPTANCE:** all 7 steps GREEN with captured evidence + operator confirmation + §11.4.151 `helix_translate-<version>` tag pushed to all upstreams (no force §11.4.113).

### P0.3 — No §11.4.151 prefixed release tag yet `[OPERATOR-BLOCKED]`
- **WHAT:** No release tag exists. Per §11.4.151 the tag must be `helix_translate-<version>` (prefix from `HELIX_RELEASE_PREFIX` env or lowercased root dir name) on main repo + every owned submodule.
- **ACCEPTANCE:** prefixed tag created post-P0.2, fanned to all upstreams.

---

## P1 — Operator / credential-gated (cannot be closed autonomously; need the operator)

### P1.0 — Decide the authoritative version number `[OPERATOR-BLOCKED]`
- **WHAT:** Is the next version 2.3.x or 3.0.0? CLAUDE.md says VERSION (2.3.0) wins; Makefile/binaries say 3.0.0. Operator must pick.

### P1.1 — Provider credentials absent/invalid `[OPERATOR-BLOCKED]`
- **OPENAI_API_KEY** absent → OpenAI provider unverified; allowlist stale (`gpt-3.5-turbo/gpt-4/gpt-4-turbo/gpt-4o` only — missing gpt-4o-mini/4.1/o-series).
- **ANTHROPIC_API_KEY** absent → Anthropic unverified; allowlist very stale (only claude-3 2024 models; missing 3.5/3.7/4.x).
- **GEMINI_API_KEY** invalid ("API Key not found", re-confirmed via live `/models`) → gemini non-functional.
- **ZHIPU** account out of balance (error 1113 "余额不足") → allowlist stale (live `/models` = `glm-4.5/4.5-air/4.6/4.7/5/5-turbo/5.1`; our allowlist has none of them) but **cannot verify translation or response-shape** → fix blocked.
- **UNBLOCK:** operator adds/refreshes the keys / recharges Zhipu. Then the **deepseek pattern** applies (proven this session): live `/models` → verify-translate + verify string-content shape → additive allowlist update + RED-proven gate guard.
- **EVIDENCE:** live `/models` probes captured this session; deepseek fix `0fd1a34` is the template.

### P1.2 — ~30 other provider allowlists not audited `[OPERATOR-BLOCKED on keys]`
- **WHAT:** qwen, groq, cohere, mistral (live shows `magistral-*` reasoning family missing — but magistral-medium returns structured-list content our client can't parse, see P2.4), xai, replicate, cerebras, cloudflare, siliconflow, hyperbolic, togetherai, sambanova, kimi, novita, nlpcloud, upstage, sarvam, modal, publicai, nia, vulavula — allowlists not verified against live current models.
- **UNBLOCK:** funded keys per provider → apply the deepseek pattern per provider (verify-translate + string-content shape before adding).

---

## P2 — Design decisions (need architecture/operator input; not blind autonomous fixes)

### P2.1 — Inert CLI flags `[DESIGN]`
- **WHAT:** `cmd/unified-translator` parses `-chunk-size`, `-workers`, `-concurrency`, `-verify` but **never consumes** them (real chunking is automatic + correct via `translateWithRetry`/`splitText`). `startMonitoringServer` is a print-only stub (the `-monitoring` flag does not start a real monitor in the unified CLI).
- **DECISION NEEDED:** wire each flag to real behavior (define semantics) OR remove it (§11.4.122 — removing a user-facing flag needs operator confirmation). Removing/altering is not a blind fix.
- **SUBAGENT-TASK (after decision):** wire or remove per operator choice, with RED→GREEN tests.

### P2.2 — Inert config fields `[DESIGN]`
- **WHAT:** `DOCXConfig.MinTextLength` + `DOCXConfig.IgnoreStyles` and `PDFConfig.MinTextLength` are declared/documented but never consumed.
- **DECISION:** wire (filter short paragraphs / honor ignore-styles) or drop. Wiring MinTextLength changes output → needs care + tests.

### P2.3 — Verifier `MinScoreThreshold` scale inconsistency `[DESIGN]`
- **WHAT:** the handler (`pkg/api/verifier_handlers.go`) compares `MinScoreThreshold` against **raw 0-100** `OverallScore`; the adapter (`internal/services/llmsverifier_score_adapter.go`) compares it against the **normalized 0-10** score. The two contracts contradict; `GetPreferences`/`GetProviderScore` currently have **no production caller**.
- **DECISION NEEDED:** declare the canonical scale (0-100 or 0-10) before either path is wired. Fixing either side blindly breaks the other's passing test (§11.4.120).

### P2.4 — Reasoning-model structured-`content` support `[DESIGN]`
- **WHAT:** OpenAI-compatible clients assume `content` is a STRING. Some reasoning models return `content` as a structured LIST (verified: Mistral `magistral-medium-latest`; likely glm-5 / deepseek-reasoner class). Our clients silently drop / can't parse it.
- **DECISION:** add structured-content handling to the OpenAI-compatible client layer → would unlock magistral/glm-5/reasoner models. Non-trivial; design + tests required.

### P2.5 — Markdown not a first-class CLI input `[DESIGN]`
- **WHAT:** `.md` input is detected as TXT and translated as plain text (works, but markdown structure not preserved). First-class markdown input is an enhancement.

### P2.6 — cmd/translator intermediate-markdown download-dir inconsistency `[OPEN, needs live SSH]`
- **WHAT:** intermediate `.md` downloads to `Dir(InputFile)` in one path vs `Dir(OutputFile)` in another; manifests only under live SSH with `-o` in a different dir. Not unit-testable without real SSH infra.
- **SUBAGENT-TASK:** reproduce via the §11.4.76 Containers submodule (boot an SSH worker container) → RED → fix → GREEN. Gated on containerized SSH test infra.

---

## P3 — Dead / unwired code (§11.4.124 investigate-before-remove)

### P3.1 — `pkg/hash` package is dead `[AUTONOMOUS — investigate]`
- **WHAT:** `pkg/hash` (393 LOC) has **ZERO importers** across the module (confirmed). Per §11.4.124 it must NOT be removed without git-history investigation establishing how/when it became dead + whether a hidden reference exists.
- **SUBAGENT-TASK:** git-history investigation (`git log --follow`, `-S` pickaxe, blame) → capture as FACT where it was wired + when it died → either (a) restore a mistakenly-deleted call-site / wire it in properly + add tests, or (b) if proven genuinely unneeded, remove it in its own descriptive commit citing the git evidence (+ operator confirm per §11.4.122 since it's a shipped package).
- **ACCEPTANCE:** captured git-history evidence + decision; if removed, separate commit citing evidence.

---

## P4 — Governance / constitution-compliance gaps (large; the project under-implements many mandates)

### P4.1 — Workable-items tracker constellation missing `[AUTONOMOUS scaffold + ongoing]`
- **WHAT:** The constitution mandates `docs/Issues.md`, `docs/Issues_Summary.md`, `docs/Fixed.md`, `docs/Fixed_Summary.md`, `docs/CONTINUATION.md` (✓ exists), per-item ATM-NNN tickets (§11.4.54), the SQLite `docs/workable_items.db` single-source-of-truth (§11.4.93/.95), procedure docs `docs/procedures/issues/*.md` (§11.4.63), Reopens/Status docs. **CONFIRMED ABSENT:** Issues/Fixed/summaries, procedures/, workable_items.db, ATM tickets. The project tracks work in `CONTINUATION.md` only.
- **SUBAGENT-TASK:** scaffold the tracker constellation: migrate this session's 27 fixes into `Fixed.md` + `Fixed_Summary.md` with ATM-NNN IDs + Status/Type columns; create empty `Issues.md`/`Issues_Summary.md`; stand up the §11.4.93 Go `cmd/workable-items` tool + DB (or document a deliberate §11.4.6 deviation). Large — split across sub-tasks.
- **ACCEPTANCE:** the doc-set exists, in sync (.md + .html + .pdf per §11.4.65), gates green.

### P4.2 — Pre-build CM-* gate suite not implemented `[AUTONOMOUS, large]`
- **WHAT:** The constitution references dozens of `CM-*` pre-build gates + paired §1.1 mutations. This project has only `scripts/testing/meta_test_constitution_inheritance.sh`; the `CM-*` gate suite + `pre_build_verification.sh` are NOT implemented.
- **SUBAGENT-TASK:** implement the highest-value gates first (anti-bluff smoke, doc-sync, regression-guard-registered, no-fakes-beyond-unit, gitignore-precommit-audit) each with a paired mutation, wired into a `scripts/pre_build_verification.sh`.
- **ACCEPTANCE:** each gate present + its paired mutation FAILs it; runnable as a suite.

### P4.3 — §11.4.65 universal markdown export audit `[AUTONOMOUS]`
- **WHAT:** all tracked non-source `.md` must have synced `.html`+`.pdf` siblings. The commit wrapper auto-syncs, but a full audit (incl. this new WORKING_PLAN.md + all docs/) is needed.
- **SUBAGENT-TASK:** run/verify `sync_all_markdown_exports`-equivalent; confirm every docs/*.md has fresh .html/.pdf.

---

## P5 — Owned submodules (§11.4.28 equal-codebase) `[OPERATOR-COORDINATION]`

### P5.1 — Submodule bug-hunt + brittle-test fixes `[BLOCKED: another session active]`
- **WHAT:** Owned submodules present: `challenges`, `containers`, `helix_qa`, `doc_processor`, `llm_orchestrator`, `llm_provider`, `vision_engine`, `llms_verifier`, `docs_chain`. They are §11.4.28 equal-codebase and have the brittle `"connection refused"` env-coupled test class (15+ sites in challenges/containers/helix_qa) — the same class fixed in the main module this session.
- **BLOCKER:** evidence shows **another session is actively working `helix_qa`** (observed `github.com/HelixDevelopment/...` `go test -race` running). Per §11.4.119 single-owner, do NOT collide.
- **SUBAGENT-TASK (after operator confirms ownership/coordination):** per-submodule bug-hunt waves (separate go.mod, separate upstreams, their own commit flow) mirroring the main-module campaign.

---

## P6 — Full test-type coverage (§11.4.25 / §11.4.27) `[AUTONOMOUS + infra-gated]`

### P6.1 — Per-feature test-type matrix + HelixQA + Challenges `[OPEN, large]`
- **WHAT:** §11.4.27 mandates 100% coverage with every test type (unit ✓, integration ✓-partial, e2e ✓-partial, security ✓-partial, stress/chaos ✓-partial via §11.4.85, plus ddos/scaling/perf/benchmark/ui/ux, Challenges bank, full HelixQA autonomous sessions). This session added unit/integration + some stress/chaos + real E2E proofs; the full matrix per feature is unfilled.
- **SUBAGENT-TASK:** build the §11.4.25 coverage ledger (feature × platform × test-type × status); fill the highest-value gaps (perf/benchmark for the translation pipeline, chaos for distributed/storage, Challenges entries per shipped feature in `tools/helixqa/banks/`).
- **ACCEPTANCE:** coverage ledger published; each shipped feature has ≥ the mandated test types with captured evidence under `docs/qa/<run-id>/` (§11.4.83).

### P6.2 — docs/qa/<run-id> evidence per shipped feature `[AUTONOMOUS]`
- **WHAT:** §11.4.83 requires a recorded e2e transcript per shipped feature. We have E2E proofs for PDF input, output-format matrix, deepseek-v4. The 27 fixes' user-visible features need per-feature evidence dirs.

---

## Confirmed GREEN / DONE this session (for completeness)
- 27 genuine mutation-proven bug fixes (waves 1-10) — see CONTINUATION.md Rev 36-41 for the itemized list with commit hashes.
- Format matrix complete: 6 inputs (FB2/EPUB/TXT/HTML/DOCX/PDF) × 5 outputs (EPUB/FB2/HTML/TXT/MD), all real-translation proven.
- PDF + DOCX input revived from license-gated-dead (MIT ledongthuc/pdf + stdlib OOXML).
- DeepSeek allowlist current (v4 models, live-`/models`-proven).
- All sweeps GREEN at quiescent checkpoints (57 ok / 0 FAIL); build+vet clean; all FF-pushed to both upstreams (no force §11.4.113).

---

## Execution order (recommended)
1. **Now (autonomous):** P0.1 version single-source wiring (pending P1.0 value) · P3.1 pkg/hash investigation · P4.3 export audit · P6.2 evidence backfill.
2. **After operator input:** P1.0 version number · P1.1/P1.2 keys → allowlist audits · P2.x design decisions.
3. **Larger autonomous tracks:** P4.1 tracker constellation · P4.2 CM-gate suite · P6.1 test-type matrix.
4. **Coordinated:** P5.1 submodules (confirm no collision).
5. **Final:** P0.2 full §11.4.40 retest → P0.3 §11.4.151 prefixed release tag.

**Nothing here is silently dropped.** Every item is tracked to a subagent task with rock-solid-proof acceptance (§11.4.123) and no-bluff verification (§11.4).
