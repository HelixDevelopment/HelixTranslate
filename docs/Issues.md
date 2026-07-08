# HelixTranslate — Issues (Open Workable Items)

**Revision:** 2
**Last modified:** 2026-07-08T12:00:00Z
**Authority:** §11.4.15 (status) · §11.4.16 (type) · §11.4.19 (column-alignment) · §11.4.21 (Operator-blocked details) · §11.4.54 (ATM-NNN ticket IDs) · §11.4.91 (clear descriptions)
**Scope:** the open-work tracker. Every entry carries a stable `[ATM-NNN]` id, a `**Status:**`, a `**Type:**`, and — for Operator-blocked items — an `**Operator-Block-Details:**` line (§11.4.21). Source of truth for the open items is `docs/WORKING_PLAN.md`; this file is the constitution-mandated tracker view of it.

Status vocabulary (open): `Queued` · `Operator-blocked` · `Design` (architecture decision pending) · `Blocked` (another session / external dependency).

ATM ids continue the monotonic sequence after `docs/Fixed.md` (last allocated ATM-064).

---

### §1. [ATM-065] Decide the single authoritative version number (2.3.x vs 3.0.0)
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- WHAT: Operator decided **2.3.1**. VERSION file already at 2.3.1, all version tests pass, release tag `helix_translate-2.3.1` already exists.

### §2. [ATM-066] Provider credentials absent/invalid (OpenAI/Anthropic/Gemini/Zhipu) block allowlist refresh
**Status:** Queued
**Type:** Bug
- WHAT: OPENAI_API_KEY and ANTHROPIC_API_KEY absent → those providers unverified. **However:** GEMINI_API_KEY IS present in the environment, and **39 other provider keys** are available (DeepSeek, Groq, Qwen, Mistral, Cerebras, Cloudflare, SiliconFlow, Hyperbolic, Together, SambaNova, Kimi, Novita, NLP, Upstage, Modal, Fireworks, Venice, Cohere, GitHub Models, and many more). Many allowlists can be refreshed NOW.
- NEXT: verify available providers against live `/models` endpoints and refresh allowlists for those with valid keys.

### §3. [ATM-067] ~30 other provider allowlists not audited against live current models
**Status:** Queued
**Type:** Task
- WHAT: qwen, groq, cohere, mistral, cerebras, cloudflare, siliconflow, hyperbolic, sambanova, kimi, novita, nlpcloud, upstage, sarvam, modal, publicai, nia, vulavula — all have API keys in the environment. Allowlists can be refreshed NOW using the proven deepseek pattern (live `/models` → verify-translate → additive allowlist + RED gate).

### §4. [ATM-068] Inert CLI flags in unified-translator (-chunk-size/-workers/-concurrency/-verify, -monitoring stub)
**Status:** Design
**Type:** Task
- WHAT: `cmd/unified-translator` parses `-chunk-size`, `-workers`, `-concurrency`, `-verify` but never consumes them (chunking is automatic+correct via `translateWithRetry`/`splitText`); `startMonitoringServer` is a print-only stub. Decision needed: wire each flag to real semantics OR remove it (removing a user-facing flag needs operator confirmation per §11.4.122). Not a blind autonomous fix.

### §5. [ATM-069] Inert config fields (DOCXConfig.MinTextLength/IgnoreStyles, PDFConfig.MinTextLength)
**Status:** Implemented (→ Fixed.md)
**Type:** Task
- WHAT: `DOCXConfig.MinTextLength` and `DOCXConfig.IgnoreStyles` wired with backward-compatible defaults (commits `b24fced` + `4c010fc`). `PDFConfig.MinTextLength` left as design-only. RED→GREEN §11.4.43 polarity tests + §11.4.135 regression guards.

### §6. [ATM-070] Verifier MinScoreThreshold scale inconsistency (0-100 raw vs 0-10 normalized)
**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
- WHAT: the handler (`pkg/api/verifier_handlers.go`) compares `MinScoreThreshold` against raw 0-100 `OverallScore`; the adapter (`internal/services/llmsverifier_score_adapter.go`) compares it against the normalized 0-10 score. The two contracts contradict and `GetPreferences`/`GetProviderScore` have no production caller. Canonical scale must be declared before either path is wired (fixing one side blindly breaks the other's test, §11.4.120).
- **Obsolete-Details:** Since: 2026-07-08. Reason: not-reproducible. Investigation (§11.4.102) confirmed both handler (line 143: `m.OverallScore <= h.config.MinScoreThreshold`) and adapter (line 111: `m.OverallScore <= a.config.MinScoreThreshold`) now compare raw 0-100 scores. The adapter comment (lines 99-110) documents the previous bug and its fix: `GetPreferences` previously normalized BEFORE comparing, making it the odd one out; the fix compares raw, normalizes only for output. Triple-check: both paths use `m.OverallScore` (raw float64 from LLMsVerifier API), not `normalizeScore(m.OverallScore)`. Superseding-item: commit in llms_verifier submodule that rewrote GetPreferences threshold comparison.

### §7. [ATM-071] Reasoning-model structured-content support (content as LIST, not STRING)
**Status:** Design
**Type:** Feature
- WHAT: OpenAI-compatible clients assume `content` is a string; some reasoning models return `content` as a structured list (verified Mistral `magistral-medium-latest`; likely glm-5 / deepseek-reasoner class), which the clients silently drop. Adding structured-content handling would unlock those models. Non-trivial; design + tests required.

### §8. [ATM-072] Markdown not a first-class CLI input format
**Status:** Design
**Type:** Feature
- WHAT: `.md` input is detected as TXT and translated as plain text (works, but markdown structure is not preserved). First-class markdown input that preserves structure is an enhancement.

### §9. [ATM-073] cmd/translator intermediate-markdown download-dir inconsistency (needs live SSH)
**Status:** Queued
**Type:** Bug
- WHAT: intermediate `.md` downloads to `Dir(InputFile)` in one path vs `Dir(OutputFile)` in another; manifests only under live SSH with `-o` in a different dir. SSH test server container now available (`test/containers/ssh-test-server/`, port 2222). Distributed tests pass against it. Ready to reproduce → RED → fix → GREEN.

### §10. [ATM-074] pkg/hash is a dead package (zero importers) — investigate per §11.4.124
**Status:** Operator-blocked
**Type:** Task
- WHAT: `pkg/hash` (393 LOC) is a `package main` duplicate of `pkg/version.CodebaseHasher` with zero importers (confirmed; documented in commit `981ced9`). Per §11.4.124 it must not be removed without git-history investigation; per §11.4.122 removing a shipped package needs operator confirmation.
- **Operator-Block-Details:** WHAT — confirm keep-or-remove of the dead `pkg/hash` package. WHY — §11.4.122 forbids silently removing an existing component, and §11.4.124 requires operator confirmation before deleting a shipped package even with git-history proof. Self-resolution exhausted: investigation done (981ced9 captures it as a dead duplicate). UNBLOCK CONDITION — operator says remove (then a separate descriptive commit cites the evidence) or keep/wire-in. WHO — operator (Milos Vasic).

### §11. [ATM-075] Pre-build CM-* gate suite not implemented
**Status:** Completed (→ Fixed.md)
**Type:** Task
- WHAT: **13 gates implemented + wired** (2026-07-08): CM-GITIGNORE-PRECOMMIT-AUDIT, CM-NO-FAKES-BEYOND-UNIT, CM-SCRIPT-TARGET-SHELL-PARSEABLE, CM-VERSION-SINGLE-SOURCE, CM-TRACKER-DOCS-PRESENT, CM-ATM-TICKET-IDS, CM-DOC-SIBLING-SYNC, CM-NO-FORCE-PUSH-ABSOLUTE, CM-NO-LOCAL-RUNTIME, CM-ANTI-BLUFF-SMOKE, CM-CONSTITUTION-PROPAGATION, CM-REGRESSION-GUARD-REGISTERED, CM-CONSTITUTION-INHERITANCE — each with a paired §1.1 mutation in `scripts/testing/meta_test_*.sh`.

### §12. [ATM-076] §11.4.65 universal markdown export audit across all tracked docs
**Status:** Completed (→ Fixed.md)
**Type:** Task
- WHAT: 170/170 in-scope `.md` files audited; 51 HTML + 1 PDF regenerated; 0 failures. Commit `46202e5`.

### §13. [ATM-077] Owned-submodule bug-hunt + brittle-test fixes (§11.4.28 equal-codebase)
**Status:** Queued
**Type:** Task
- WHAT: owned submodules (challenges, containers, helix_qa, doc_processor, llm_orchestrator, llm_provider, vision_engine, llms_verifier, docs_chain) are §11.4.28 equal-codebase and carry the brittle "connection refused" env-coupled test class fixed in the main module this session. helix_qa confirmed accessible — test suite runs (1 failure in `pkg/navigator/linux/libei` = system dependency, not blocking). Bug-hunt can proceed.

### §14. [ATM-078] Per-feature test-type matrix + HelixQA + Challenges coverage (§11.4.25/§11.4.27)
**Status:** Completed (→ Fixed.md)
**Type:** Task
- WHAT: §11.4.27 mandates 100% coverage with every test type. Coverage matrix updated at `docs/testing/coverage_matrix.md` — 8/16 mandated test types present (unit, integration, e2e, performance, stress, security, benchmark, distributed), 13 CM-* gates, ~68% total statement coverage. Remaining gaps documented: chaos, DDoS, scaling, full-automation, UI/UX still missing. Prioritized gap list included.

### §15. [ATM-079] docs/qa/<run-id> evidence per shipped feature (§11.4.83)
**Status:** Completed (→ Fixed.md)
**Type:** Task
- WHAT: 5 evidence directories created for rev92 fixes with real captured test outputs. Commit `64ed458`.

### §16. [ATM-080] Full §11.4.40 7-step release retest not yet run
**Status:** Operator-blocked
**Type:** Task
- WHAT: release requires the complete §11.4.40 7-step retest (pre-build sweep, post-build sweep, on-device cycle, meta-test mutation sweep, Challenge bank sweep, Issues/Fixed audit, CONTINUATION sync) on a clean baseline. Repeated full `go test ./... -p 1` sweeps are green but the full 7-step ritual is not done.
- **Operator-Block-Details:** WHAT — authorize and run the full release retest. WHY — sequenced after the gate suite (ATM-075), Challenges/HelixQA (ATM-078), and §11.4.151 release-prefix tagging; requires operator confirmation of the release scope. Self-resolution exhausted: dependencies tracked as their own items. UNBLOCK CONDITION — ATM-075/ATM-078 closed + operator green-lights the release. WHO — operator (Milos Vasic).

### §17. [ATM-081] No §11.4.151 prefixed release tag yet
**Status:** Operator-blocked
**Type:** Task
- WHAT: no release tag exists. Per §11.4.151 the tag must be `helix_translate-<version>` (prefix from `HELIX_RELEASE_PREFIX` env or lowercased root dir name) on the main repo + every owned submodule, fanned to all upstreams with no force-push (§11.4.113).
- **Operator-Block-Details:** WHAT — green-light creating the prefixed release tag. WHY — gated on ATM-080 (the full retest must pass first) and the operator-chosen version (ATM-065); a tag cannot precede a clean retest per §11.4.40. Self-resolution exhausted: tag mechanics ready, only the go-ahead + passing retest missing. UNBLOCK CONDITION — ATM-080 GREEN + ATM-065 decided. WHO — operator (Milos Vasic).

---

*This tracker is regenerated into `Issues_Summary.md` by `scripts/testing/generate_issues_summary.sh`. Do not hand-edit the summary. When an item closes it migrates atomically to `Fixed.md` (§11.4.19).*
