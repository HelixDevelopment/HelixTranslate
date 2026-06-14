# HelixTranslate — Working Plan: Unfinished Items & Known Issues

**Revision:** 2
**Last modified:** 2026-06-15T04:45:00Z
**Authority:** operator mandate 2026-06-14/15 ("what is left unfinished and what are (known) issues … nothing left unfinished with zero issues when we finally finish")
**Scope:** the complete, honest (§11.4.6) inventory of everything not-done + every known issue, structured as a subagent-driven execution plan (§11.4.70). Each item carries: WHAT, STATUS, EVIDENCE, SUBAGENT-TASK, ACCEPTANCE (rock-solid proof §11.4.123, no bluff), and PARALLEL-SAFE (disjoint scope for §11.4.58/§11.4.119).

**Baseline at authoring:** main HEAD `e4c96f6`; `go build ./...` clean; full `go test ./... -p 1` GREEN at quiescent checkpoints; **38 genuine, reproduce-first, mutation-proven bug fixes** landed this session (waves 1–12) across the main module + 8 owned submodules, all committed + pushed (submodules → own mirrors, main → both upstreams, gitlink pointers synced). The high-yield product bug-hunt has reached the §11.4.118 **completeness boundary** — recent waves return clean non-findings on already-hardened/rule-correct packages.

Status legend: `OPEN` · `OPERATOR-BLOCKED` (needs credentials/decision) · `DESIGN` (needs architecture decision) · `AUTONOMOUS` (a subagent can close it now) · `DONE`.

---

## P0 — Release blockers (must close before any release tag)

### P0.1 — Version single-source-of-truth `[DONE — value still operator-gated, see P1.0]`
- **WHAT:** `pkg/version/app.go` now holds `const AppVersion = "2.3.0"`; the 5 binaries were reconciled to it this session. `VERSION`=`2.3.0` is authoritative per CLAUDE.md; `Makefile` still references `3.0.0`.
- **REMAINING:** the *number* (2.3.0 vs 3.0.0) is P1.0; the Makefile literal should track the chosen value.
- **ACCEPTANCE:** all `appVersion` literals == `pkg/version.AppVersion` (DONE); Makefile reconciled once P1.0 decided.

### P0.2 — Full §11.4.40 release retest not yet run `[OPEN — sequenced last]`
- **WHAT:** Release requires the complete §11.4.40 7-step retest (pre-build sweep, post-build sweep, on-device/topology cycle, meta-test mutation sweep, Challenge bank sweep, Issues/Fixed audit, CONTINUATION sync) on a clean baseline. Repeated full `go test ./... -p 1` sweeps are GREEN, but the full 7-step ritual has not been run end-to-end.
- **SUBAGENT-TASK:** sequenced AFTER P1.0 (version) + P4 (gate suite) + P6 (Challenges). Then run all 7 steps with captured evidence.
- **ACCEPTANCE:** all 7 steps GREEN with captured evidence + operator confirmation.

### P0.3 — No §11.4.151 prefixed release tag yet `[OPERATOR-BLOCKED on P0.2 + P1.0]`
- **WHAT:** No release tag exists. Per §11.4.151 it must be `helix_translate-<version>` (prefix from `HELIX_RELEASE_PREFIX` env, else lowercased root dir) on main repo + every owned submodule, fanned to all upstreams (no force §11.4.113).
- **ACCEPTANCE:** prefixed tag created post-P0.2, pushed to all upstreams across main + owned submodules.

---

## P1 — Operator / credential-gated (cannot be closed autonomously)

### P1.0 — Decide the authoritative version number `[OPERATOR-BLOCKED]`
- **WHAT:** Next version = `2.3.x` or `3.0.0`? CLAUDE.md says `VERSION` (2.3.0) wins; `Makefile` says `3.0.0`. **Recommendation (§11.4.101 evidence-backed):** `2.3.0` per the CLAUDE.md authoritative-VERSION rule, unless you intend a major bump.

### P1.1 — Provider credentials absent/invalid `[OPERATOR-BLOCKED]`
- **OPENAI_API_KEY** absent → allowlist stale (missing gpt-4o-mini/4.1/o-series). **ANTHROPIC_API_KEY** absent → allowlist very stale (missing 3.5/3.7/4.x). **GEMINI_API_KEY** invalid (live `/models` "API Key not found"). **ZHIPU** out of balance (error 1113) → allowlist stale (live glm-4.5…5.1) but translation/response-shape unverifiable.
- **UNBLOCK:** operator adds/refreshes keys / recharges Zhipu → then the proven **deepseek pattern** applies (live `/models` → verify-translate + verify response-content shape → additive allowlist update + RED-proven gate). Template: deepseek fix in pkg/translator/llm/llm.go.

### P1.2 — ~30 other provider allowlists not audited `[OPERATOR-BLOCKED on keys]`
- **WHAT:** qwen/groq/cohere/mistral/xai/replicate/cerebras/cloudflare/siliconflow/hyperbolic/togetherai/sambanova/kimi/novita/nlpcloud/upstage/sarvam/modal/publicai/nia/vulavula allowlists unverified vs live models.
- **UNBLOCK:** funded keys per provider → deepseek pattern per provider (verify-translate + string-content shape before adding).

---

## P2 — Design decisions (need architecture/operator input; not blind autonomous fixes)

### P2.1 — Inert CLI flags `[DESIGN]`
- `cmd/unified-translator` parses `-chunk-size`/`-workers`/`-concurrency`/`-verify` but never consumes them (chunking is automatic + correct); `-monitoring` start is a print-only stub. DECISION: wire to real behavior OR remove (removal needs §11.4.122 operator confirm). Then RED→GREEN.

### P2.2 — Inert config fields `[DESIGN]`
- `DOCXConfig.MinTextLength`/`IgnoreStyles` + `PDFConfig.MinTextLength` declared but unconsumed. DECISION: wire (changes output, needs care+tests) or drop.

### P2.3 — Verifier `MinScoreThreshold` scale inconsistency `[DESIGN]`
- `pkg/api` handler compares threshold vs raw 0-100 `OverallScore`; the adapter compares vs normalized 0-10. Contracts contradict. **NOTE:** wave-7 fixed the adapter's `GetPreferences`-drops-all + empty-cache bugs, but the cross-layer scale contract itself is still a design call — declare canonical scale (0-100 or 0-10) before wiring either path (fixing one side blindly breaks the other's test, §11.4.120).

### P2.4 — Reasoning-model structured-`content` support `[DESIGN]`
- OpenAI-compatible clients assume `content` is a STRING; some reasoning models return a structured LIST (verified Mistral `magistral-medium-latest`; likely glm-5/deepseek-reasoner). Adding structured-content handling unlocks them. Non-trivial; design+tests.

### P2.5 — Markdown not a first-class CLI input `[DESIGN]`
- `.md` input detected as TXT (works, structure not preserved). First-class markdown input is an enhancement.

### P2.6 — cmd/translator intermediate-markdown download-dir inconsistency `[OPEN, needs live SSH]`
- Intermediate `.md` downloads to `Dir(InputFile)` vs `Dir(OutputFile)` across paths; reproducible only under live SSH. SUBAGENT-TASK: reproduce via §11.4.76 Containers submodule (boot SSH worker container) → RED → fix → GREEN. Gated on containerized SSH infra.

---

## P3 — Dead / unwired code (§11.4.124 investigate-before-remove)

### P3.1 — `pkg/hash` package `[AUTONOMOUS — investigate]` · PARALLEL-SAFE
- **WHAT:** `pkg/hash` has ZERO importers across the module. §11.4.124: must NOT be removed without git-history investigation (where/how wired, when it died, hidden-reference check) + §11.4.122 operator confirm (shipped package).
- **SUBAGENT-TASK:** `git log --follow`/`-S` pickaxe/blame → capture FACT → either wire it in properly + add tests, OR (proven-unneeded) remove in its own descriptive commit citing git evidence + operator confirm.
- **NOTE:** an untracked compiled `hash` binary + `workable-items` binary sit at repo root (build artifacts, §11.4.30 — never commit; candidate for `.gitignore` entry, see P4.4).

---

## P4 — Governance / constitution-compliance (partially landed this campaign)

### P4.1 — Workable-items tracker constellation `[PARTIAL — DONE-scaffold, ongoing sync]`
- **DONE:** `docs/Issues.md` (10 open ATM tickets), `docs/Fixed.md`, `docs/Issues_Summary.md`, `docs/Fixed_Summary.md`, `docs/CONTINUATION.md`, the §11.4.93 Go `cmd/workable-items` tool + tracked `docs/workable_items.db` (§11.4.95) all exist (created this campaign).
- **REMAINING:** the 38 campaign fixes are recorded narratively in CONTINUATION rev 52 but NOT all migrated into `Fixed.md`/the DB with per-fix ATM IDs; procedure docs `docs/procedures/issues/*.md` (§11.4.63) + per-item Reopens/Status docs not yet present.
- **SUBAGENT-TASK:** migrate the 38 fixes into Fixed.md + workable_items.db with ATM IDs + Status/Type; add the 5 procedure docs. PARALLEL-SAFE (docs-only).

### P4.2 — Pre-build CM-* gate suite `[PARTIAL — 8 gates landed, more mandated]`
- **DONE:** `scripts/pre_build_verification.sh` with 8 gates (CM-GITIGNORE-PRECOMMIT-AUDIT, CM-NO-FAKES-BEYOND-UNIT, CM-SCRIPT-TARGET-SHELL-PARSEABLE, CM-VERSION-SINGLE-SOURCE, CM-TRACKER-DOCS-PRESENT, CM-ATM-TICKET-IDS, CM-DOC-SIBLING-SYNC, CM-NO-FORCE-PUSH-ABSOLUTE), each with a paired §1.1 mutation, all green.
- **REMAINING:** the constitution references dozens more CM-* gates (regression-guard-registered §11.4.135, multi-pass-evaluation, sink-evidence-per-feature, etc.). SUBAGENT-TASK: implement the next highest-value gates each with a paired mutation. PARALLEL-SAFE (scripts-only).

### P4.3 — §11.4.65 universal markdown export `[AUTONOMOUS — verify]` · PARALLEL-SAFE
- The doc-sibling guard (CM-DOC-SIBLING-SYNC) is green; commit wrapper auto-syncs. SUBAGENT-TASK: full audit that every tracked non-source `.md` (incl. this WORKING_PLAN.md) has fresh `.html`+`.pdf` siblings.

### P4.4 — `.gitignore` build-artifact hygiene `[AUTONOMOUS]` · PARALLEL-SAFE
- **WHAT:** untracked compiled binaries `hash`, `workable-items` (and tracked rebuilt `unified-translator`/`preparation-translator`/`translate-ssh`) sit at repo root. §11.4.30 forbids committing build artifacts. SUBAGENT-TASK: add `.gitignore` entries for the compiled root binaries + a §11.4.77 regen note (they rebuild from `cmd/`); leave the tracked ones for an operator decision (removing a tracked binary is a visible change). PARALLEL-SAFE.

### P4.5 — commit_all.sh companion doc (§11.4.18) `[AUTONOMOUS, minor]` · PARALLEL-SAFE
- This session patched `scripts/commit_all.sh` (§11.4.120 guard reconcile). §11.4.18 wants a `docs/scripts/commit_all.md` companion; none exists. SUBAGENT-TASK: author the companion doc (Purpose/Usage/guard behavior) if the project adopts §11.4.18, else record a deliberate deviation.

---

## P5 — Owned submodules (§11.4.28 equal-codebase) `[DONE this campaign]`

### P5.1 — Submodule bug-hunt `[DONE — 20 mutation-proven fixes, pushed]`
- **DONE:** llm_provider ×5, llms_verifier ×4, doc_processor ×2, llm_orchestrator ×2, docs_chain ×2, challenges ×2, containers ×2 — all reproduce-first + mutation-proven, committed + pushed to their own mirrors, parent gitlink pointers synced (commit `ebea2d3`+`e21c779`). vision_engine's gocv bug was independently fixed upstream (`64b7d08`) → integrated, not double-fixed.
- **REMAINING:** `helix_qa` — NOT hunted; another session is actively working it (`m` working tree, §11.4.119 single-owner). `security` + `constitution` submodules — not hunted (constitution is governance; security submodule is a candidate next pass). SUBAGENT-TASK (after operator confirms no collision): hunt the `security` submodule.

### P5.2 — §11.4.147 misplaced sibling clone cleanup `[OPERATOR-BLOCKED]`
- **WHAT:** a separate sibling clone `/Volumes/T7/Projects/llm_orchestrator` (OUTSIDE the parent tree) holds a STALE, superseded, uncommitted RoundRobinSelector fix (`M pkg/agent/multi_pool.go` + new test) — a subagent wrote it there by mistake; the correct fix was re-applied + committed in the submodule (`4e08b93`). The sibling's uncommitted change is now obsolete.
- **EVIDENCE:** `git -C /Volumes/T7/Projects/llm_orchestrator status` shows the stale M/?? entries.
- **SUBAGENT-TASK:** none — operator decides whether to `git checkout -- .` / discard that sibling clone's working tree (safe: the fix is already in the real submodule). Flagged so it is not mistaken for live work.

---

## P6 — Full test-type coverage (§11.4.25 / §11.4.27) `[OPEN, large]`

### P6.1 — Per-feature test-type matrix + Challenges + HelixQA `[OPEN]`
- §11.4.27 mandates 100% coverage with every test type. This campaign added unit + integration + stress/chaos (partial) + real E2E proofs; the full matrix (ddos/scaling/perf/benchmark/ui/ux, Challenges bank, full HelixQA autonomous sessions) is unfilled. SUBAGENT-TASK: build the §11.4.25 coverage ledger (feature × platform × test-type × status); fill highest-value gaps (perf/benchmark for the translation pipeline, chaos for distributed/storage, Challenges per shipped feature).

### P6.2 — docs/qa/<run-id> evidence per shipped feature `[AUTONOMOUS]` · PARALLEL-SAFE
- §11.4.83 requires a recorded e2e transcript per shipped feature. SUBAGENT-TASK: per-feature evidence dirs for the campaign's user-visible fixes.

---

## P7 — Remaining low-yield hunt surface (completeness, §11.4.118)

### P7.1 — Un-hunted main packages `[AUTONOMOUS, low yield]` · PARALLEL-SAFE
- Not yet hunted (low complexity / low yield): `pkg/language`, `pkg/models`, `pkg/modelsbridge`, `pkg/sshworker`, `pkg/api` (handlers), `pkg/deployment`, `pkg/challenge_runner`, `pkg/logger`, `pkg/version`, `pkg/batch` (coordination-batch hunted), `cmd/*` CLI entry points. SUBAGENT-TASK: continue the reproduce-first/mutation-proven hunt one disjoint package per subagent (3-wide; the host rate-limiter kills 4-wide fan-outs — use probe-then-scale).

### P7.2 — Audited CLEAN this campaign (no genuine bug; existing guards mutation-verified real, §11.4.6)
- pkg/translator (root), pkg/events, pkg/websocket, pkg/script, pkg/report, pkg/format, pkg/hardware. **Two UNCONFIRMED Windows-only leads** in pkg/hardware (`detectGPU` false-positive; `vm_stat` overflow) flagged for a future Windows-capable session — NOT fabricated fixes.

---

## Confirmed GREEN / DONE this session (for completeness)
- **38 genuine mutation-proven bug fixes** — CONTINUATION rev 52 has the itemized table with commit SHAs (main: distributed×4, ebook/fb2×4, verifier×2, verification×2, storage, security, markdown, coordination, preparation, grpc, progress; submodules: ×20).
- §11.4.120-reconciled `commit_all.sh` guard (skip gitlinks + no self-match), paired-verified.
- All sweeps GREEN at quiescent checkpoints; build+vet clean; all FF-pushed to both upstreams (no force §11.4.113); parent gitlink pointers synced.

---

## Execution order (recommended)
1. **Autonomous, parallel-safe NOW (3-wide subagent waves):** P4.1 fix-migration into Fixed.md/DB · P4.3 export audit · P4.4 .gitignore hygiene · P6.2 evidence backfill · P7.1 low-yield package hunt.
2. **After operator input:** P1.0 version number · P1.1/P1.2 keys → allowlist audits · P2.x design decisions · P5.2 sibling-clone discard.
3. **Larger autonomous tracks:** P4.2 more CM-gates · P6.1 test-type matrix + Challenges + HelixQA.
4. **Coordinated:** P5.1 `security` submodule (confirm no collision); `helix_qa` owned by another session.
5. **Final (release):** P0.2 full §11.4.40 7-step retest → P0.3 §11.4.151 `helix_translate-<version>` tag fanned to all upstreams.

**Nothing here is silently dropped.** Every item is tracked to a subagent task with rock-solid-proof acceptance (§11.4.123) and no-bluff verification (§11.4). "Zero issues when we finish" = every P0 closed, every P1/P2 operator-decided, P4–P7 driven to completion or explicitly operator-deferred.
