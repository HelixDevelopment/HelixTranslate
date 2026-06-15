# HelixTranslate — Constitution

> **Status:** Active. This document is the project's authoritative
> rule set. When a rule here conflicts with `CLAUDE.md`, `AGENTS.md`,
> or any guide, the Constitution wins.

## INHERITED FROM constitution/Constitution.md

This constitution **extends** the Helix Universal Constitution at
`constitution/Constitution.md`. All clauses there apply unless
explicitly overridden below with an explicit `Override §X.Y`
section. The universal clauses are authoritative for any topic this
file does not cover; the project-specific rules below (CONST-033 …
CONST-036 and the Definition of Done) extend them and never weaken
any universal clause. When this file disagrees with the constitution
submodule, the constitution wins.

## Mission

See README.md.

## Mandatory Standards

1. **Reproducibility:** every change is reproducible from a clean
   clone (`git clone <repo> && <project bootstrap>`); no hidden steps.
2. **Tests track behavior, not code:** test what the user-visible
   behavior is, not what the implementation looks like.
3. **No silent skips, no silent mocks above unit tests.**
4. **Conventional Commits** for all commits.
5. **SSH-only for git operations** (`git@…`); HTTPS prohibited.

## Numbered Rules

<!-- Rules are numbered CONST-NNN. New rules append. Removed rules
     keep their number with a "**Retired:** …" line. -->

<!-- BEGIN host-power-management addendum (CONST-033) -->

### CONST-033 — Host Power Management is Forbidden

**Status:** Mandatory. Non-negotiable. Applies to every project,
submodule, container entry point, build script, test, challenge, and
systemd unit shipped from this repository.

**Rule:** No code in this repository may invoke a host-level power-
state transition (suspend, hibernate, hybrid-sleep, suspend-then-
hibernate, poweroff, halt, reboot, kexec) on the host machine. This
includes — but is not limited to:

- `systemctl {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}`
- `loginctl {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}`
- `pm-{suspend,hibernate,suspend-hybrid}`
- `shutdown {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}`
- DBus calls to `org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}`
- DBus calls to `org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to any value other than `'nothing'` or `'blank'`

**Why:** The host runs mission-critical parallel CLI-agent and
container workloads. On 2026-04-26 18:23:43 the host was auto-
suspended by the GDM greeter's idle policy mid-session, killing
HelixAgent and 41 dependent services. Recurring memory-pressure
SIGKILLs of `user@1000.service` (perceived as "logged out") have the
same outcome. Auto-suspend, hibernate, and any power-state transition
are unsafe for this host.

**Defence in depth (mandatory artifacts in every project):**
1. `scripts/host-power-management/install-host-suspend-guard.sh` —
   privileged installer, manual prereq, run once per host with sudo.
   Masks `sleep.target`, `suspend.target`, `hibernate.target`,
   `hybrid-sleep.target`; writes `AllowSuspend=no` drop-in; sets
   logind `IdleAction=ignore` and `HandleLidSwitch=ignore`.
2. `scripts/host-power-management/user_session_no_suspend_bootstrap.sh` —
   per-user, no-sudo defensive layer. Idempotent. Safe to source from
   `start.sh` / `setup.sh` / `bootstrap.sh`.
3. `scripts/host-power-management/check-no-suspend-calls.sh` —
   static scanner. Exits non-zero on any forbidden invocation.
4. `pkg/challenge_runner/scripts/host_no_auto_suspend_challenge.sh` — asserts
   the running host's state matches layer-1 masking.
5. `pkg/challenge_runner/scripts/no_suspend_calls_challenge.sh` — wraps the
   scanner as a challenge that runs in CI / `run_all_challenges.sh`.

**Enforcement:** Every project's CI / `run_all_challenges.sh`
equivalent MUST run both challenges (host state + source tree). A
violation in either channel blocks merge. Adding files to the
scanner's `EXCLUDE_PATHS` requires an explicit justification comment
identifying the non-host context.

**See also:** `docs/HOST_POWER_MANAGEMENT.md` for full background and
runbook.

<!-- END host-power-management addendum (CONST-033) -->

<!-- BEGIN llmsverifier-single-source-of-truth addendum (CONST-034) -->

### CONST-034 — LLMsVerifier is the Single Source of Truth for All LLM Models

**Status:** Mandatory. Non-negotiable. Applies to all translation workflows, API endpoints, CLI commands, distributed workers, and WebSocket sessions.

**Rule:** LLMsVerifier SHALL be the EXCLUSIVE and ONLY source of truth for all LLM models available to HelixTranslate. No model may be used in HelixTranslate that does not originate from LLMsVerifier and pass its verification gate.

**Requirements:**
1. **F-SSOT-001:** Only models verified by LLMsVerifier are eligible for use.
2. **F-SSOT-002:** HelixTranslate MUST NOT maintain any independent model registry, hardcoded model lists, or fallback model definitions outside of LLMsVerifier-provided data.
3. **F-SSOT-003:** All model metadata (name, ID, capabilities, token limits, pricing, features) SHALL be retrieved from LLMsVerifier at runtime or build time.
4. **F-SSOT-004:** Model information SHALL be cached locally with configurable TTL, but cache invalidation MUST trigger re-fetch from LLMsVerifier.
5. **F-SSOT-005:** If LLMsVerifier is unreachable, HelixTranslate SHALL either use expired cache with warning or enter degraded mode with explicit user notification. No unauthorized models SHALL be used.
6. **F-GATE-001:** ONLY models that have PASSED LLMsVerifier validation, verification, and scoring pipeline SHALL be presented to end users.
7. **F-GATE-002:** Models MUST satisfy ALL three criteria: (a) validation passed, (b) verification passed, (c) positive scoring (> 0 overall score).
8. **F-GATE-003:** The scoring algorithm: `Overall Score = (Responsiveness x 0.30) + (Code Capability x 0.25) + (Feature Richness x 0.25) + (Reliability x 0.20)`. Models with score <= 0 SHALL be rejected.
9. **F-GATE-004:** Models MUST have `VerificationStatus == "verified"` AND `CanSeeCode == true` to be eligible.
10. **F-GATE-005:** Models MUST have `AffirmativeResponse == true` to be eligible.

**Enforcement:** Any code change that introduces a hardcoded model list, bypasses the verification gate, or uses an unverified model constitutes a Constitution violation and MUST be reverted.

<!-- END llmsverifier-single-source-of-truth addendum (CONST-034) -->

<!-- BEGIN anti-bluff-testing addendum (CONST-035) -->

### CONST-035 — Anti-Bluff Testing Constitution (MANDATORY)

**Status:** Mandatory. Non-negotiable. Applies to every test, challenge, HelixQA bank entry, and CI pipeline in this repository and all submodules.

**§35.1 The Problem (Historical Mandate)**

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used!"

This is the **worst possible outcome**: green tests + broken features. Every rule below exists to prevent this.

**§35.2 The Six Anti-Bluff Rules**

Every test, Challenge, and HelixQA bank entry MUST:

1. **Assert on a concrete end-user-visible outcome** — translated text content, DB row, downloadable file, visible dashboard element, API response body. NOT just "no error" or "200 OK" or "exit code 0".

2. **Run against the real system** — mocks ONLY in unit tests (`go test -short`). ALL other test types MUST use real services, real databases, real LLM providers (or documented `SKIP-OK: #<ticket>`).

3. **Include a matching negative assertion** — test MUST fail when the feature is broken. E.g., if testing "English→Serbian translation", also assert the output is NOT in English.

4. **Emit copy-pasteable evidence** — response body snippet, screenshot filename, DB row dump, log excerpt, video timestamp.

5. **Verify "fails when feature is removed"** — deliberately break the feature (comment out implementation, change API key to invalid), re-run test, test MUST FAIL.

6. **No blind shells** — no `&& echo PASS`, no `|| true`, no `tee` exit-code laundering, no `test -f file && echo "PASS"` without checking file content.

**§35.3 HelixQA-Specific Anti-Bluff Rules**

- Bank entries declare **executable actions** (never prose).
- Each entry declares **concrete success predicates**: `assertBodyContains: 'translated text'`, `assertVisible: 'Translation Complete'`.
- **Stagnation guard** — frame N+1 identical to N for >10 seconds = FAIL.
- Vision-model `verified=true` with empty/tautological reasoning = `INCONCLUSIVE` (not PASS).
- `IsBlankScreenshot()` must pass before any vision analysis.

**§35.4 Functional Probe Floor**

- TCP-open is the FLOOR, not the ceiling.
- PostgreSQL → `SELECT 1` returns `1`.
- Redis → `PING` returns `PONG`.
- API → `GET /health` returns `{"status": "healthy"}` with non-empty body.
- Translation → actual translated text in response, not just session_id.

**§35.5 Evidence Requirements**

- Every PASS must carry positive evidence captured during execution.
- No metadata-only PASS, no configuration-only PASS, no "absence-of-error" PASS.
- Evidence types: API response bodies, downloaded files, database query results, browser console output, screenshots, video frames.

**§35.6 Bluff Taxonomy (FORBIDDEN Patterns)**

| Bluff Type | Example | Why It's Wrong |
|-----------|---------|---------------|
| **Wrapper bluff** | Test asserts function returns, but caller ignores return value | Passes but feature unused |
| **Contract bluff** | System advertises capability but rejects it in dispatch | Advertises what it can't do |
| **Structural bluff** | Checks file exists but doesn't verify content | File exists but is empty/corrupt |
| **Comment bluff** | Code comment promises behavior code doesn't have | Tests comment, not code |
| **Skip bluff** | `t.Skip("not running yet")` without `SKIP-OK` marker | Hides broken test |

**§35.7 Mutation Testing (Mandatory)**

- Every challenge MUST have a paired mutation test.
- Mutation deliberately breaks the feature → the challenge MUST then FAIL.
- A challenge without a paired mutation = BLUFF challenge = Constitution violation.

**§35.8 Audit Ritual**

Every Full-QA cycle MUST:
1. Pick 5 random tests + 5 random challenges.
2. Comment out the target implementation.
3. Re-run tests/challenges.
4. Confirm they FAIL.
5. Restore implementation.
6. Document results in session report.

**§35.9 User Mandate (NON-NEGOTIABLE)**

The bar is NOT "tests pass" but **"users can use the feature."**
A translation that "completes" but produces garbage text is a FAIL.
An API that returns 200 but with wrong content is a FAIL.
A dashboard that "loads" but shows no data is a FAIL.

<!-- END anti-bluff-testing addendum (CONST-035) -->

<!-- BEGIN helixqa-mandate addendum (CONST-036) -->

### CONST-036 — HelixQA is the Sole Authorized QA Tool

**Status:** Mandatory. Non-negotiable.

**Rule:** All automated UI/UX testing of the Web Dashboard and API endpoints MUST use HelixQA. No custom Playwright scripts, no curl-based test harnesses outside HelixQA banks.

**Requirements:**
1. **HelixQA-only for Web Dashboard and API testing.**
2. **Vision-driven only.** Screenshot → LLM analysis → action decision. No hardcoded selectors, no sleep timers.
3. **Universal Solution Principle.** When HelixQA cannot interact with a HelixTranslate UI element, the fix MUST be implemented in HelixQA itself, never by adding test hooks to HelixTranslate.
4. **Live log monitoring.** Every session streams API logs, gRPC logs, translation logs.
5. **Screen-state tracking.** Frame N vs N+1. Stagnation >10s = critical failure.
6. **Executable actions in banks**, never prose.
7. **Video mandatory for Web Dashboard sessions.** Screenshots at every step.
8. **Evidence validation.** Post-translation must contain actual translated text, not placeholder.

<!-- END helixqa-mandate addendum (CONST-036) -->

## Definition of Done

A change is done when:

1. The code change is committed.
2. All project-level tests pass on a clean clone.
3. All challenges in `pkg/challenge_runner/scripts/` pass on the running host.
4. Governance docs (`CONSTITUTION.md`, `AGENTS.md`, `CLAUDE.md`) are
   coherent with the change.
5. All HelixQA banks pass for all configured platforms.
6. Anti-bluff audit (CONST-035 §35.8) passes — random tests confirmed to fail when feature removed.
7. Every modified/added feature has a registered challenge.
8. Evidence from the most recent QA session is archived in `docs/reports/qa-sessions/`.

## See also

- `README.md` — project overview, quickstart.
- `AGENTS.md` — guidance for AI coding agents (Codex, Cursor, etc.).
- `CLAUDE.md` — guidance specifically for Claude Code.
- `docs/HOST_POWER_MANAGEMENT.md` — CONST-033 background and runbook.
- `docs/research/llms_verifer/` — LLMsVerifier integration materials.
- `docs/research/helix_qa/` — HelixQA integration materials.

## CONST-035 — Anti-Bluff Operative Rule (MANDATORY)

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case."

**The operative rule:** Execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product.

- A green test or challenge for a feature that does not actually work is a **BLUFF** and is **FORBIDDEN**.
- Every test must assert concrete user-visible outcomes, not just internal state.
- Every challenge must run real code and verify real behavior; grep/file-existence checks are **NOT sufficient**.
- Mutation testing is **MANDATORY**: deliberately break the feature → the test/challenge **MUST then FAIL**.
- The bar for shipping is **NOT "tests pass"** but **"users can use the feature."**
- No false-success results are tolerable.

## Mirrored constitution anchors (§11.4.154–§11.4.157)

Condensed consumer-side mirrors of the four universal anchors most recently
added to the constitution submodule. Canonical text lives in
`constitution/Constitution.md`; these mirrors carry the literal `11.4.N` tokens
the propagation gates (`CM-COVENANT-114-154/155/156/157-PROPAGATION`) require
(§11.4.35). Faithful condensations — no clause weakened, nothing invented.

**§11.4.154 — Window-scoped capture + fresh-corpus rotation for feature/QA recordings (User mandate, 2026-06-15).** Refines §11.4.2/.5/.107/.153 recording discipline with two capture-hygiene invariants. **(A) Window-scoped, NOT whole-screen** — every feature/QA video MUST capture ONLY the window/surface of the app/service under test (GUI window / CLI-TUI terminal pane / web tab-viewport / device-emulator frame), NEVER the whole desktop/monitor or unrelated windows (whole-desktop capture leaks operator-private content §11.4.10/.83, dilutes the §11.4.107 liveness/freeze oracle, breaks the §11.4.137 OCR/ROI oracle); target by stable identity (window id/title, device serial, browser context, tmux target) per §11.4.111, never a fixed full-screen index; platform genuinely cannot capture below whole-screen ⇒ honest §11.4.3 SKIP + tracked migration item. **(B) Fresh-corpus rotation** — when a new recording run for a scope begins, the agent's OWN prior in-scope stale recordings at the raw recording path MUST be removed FIRST so the live corpus reflects the current run (§11.4.107 not-stale + §11.4.86 roster-freshness); "remove old" = the agent's own prior recordings for the SAME scope/project ONLY, NEVER another project's/operator-authored files (uncertain ⇒ surface, don't delete §11.4.122/§9.2); committed `docs/qa/<run-id>/` evidence (§11.4.83) is the durable record, NOT rotated. Classification: universal (§11.4.17). Composes §11.4.2/.5/.10/.83/.86/.107/.111/.122/.128/.137/.153/§9.2/.6. Canonical authority: constitution submodule `Constitution.md` §11.4.154. Non-compliance is a release blocker.

**§11.4.155 — Project-name-prefixed feature/QA recording filenames (User mandate, 2026-06-15).** Every recorded video the project produces (§11.4.153 real-use, §11.4.154 window-scoped, §11.4.128 always-on device, any raw/curated artefact at the recording path + the committed `docs/qa/<run-id>/` trail §11.4.83) MUST have a filename that STARTS WITH the PROJECT-NAME prefix, ALWAYS; an unprefixed recording is a §11.4.155 violation (un-greppable + un-attributable multi-project corpus — the §11.4.151 identify-and-grep failure on the recording axis). **Prefix resolution (closed-set, deterministic — §11.4.6, IDENTICAL to §11.4.151):** (1) `HELIX_RELEASE_PREFIX` from `.env` (git-ignored §11.4.30, documented in tracked `.env.example` §11.4.77) else (2) lowercased snake_case project-root dir name §11.4.29; SAME prefix for EVERY recording in a checkout; canonical form `<PREFIX>---<feature-or-scope>---<run-id>.<ext>`; MUST equal the §11.4.151-resolved release-tag prefix (divergence is itself a violation — one project, one name). Honest boundary (§11.4.6): the prefix guarantees attribution + greppability, NOT content validity (still §11.4.107/.137/.153) and does NOT relax §11.4.154's window-scope/rotation. Classification: universal (§11.4.17). Composes §11.4.151/.128/.153/.154/.111/.83/.6/.29/.30/.35/.77/.86. Canonical authority: constitution submodule `Constitution.md` §11.4.155. Non-compliance is a release blocker.

**§11.4.156 — All CI/CD automation (GitHub Actions / GitLab pipelines / equivalents) MUST be disabled (User mandate, 2026-06-15).** Every repository this Constitution governs — main repo, this constitution submodule, every owned + nested submodule we author and push — MUST ship with ALL server-side CI/CD automation DISABLED: no push to any owned upstream may trigger a GitHub Actions run, GitLab pipeline, or equivalent (Jenkins/CircleCI/Travis/Drone/Woodpecker/Bitbucket/Azure, any `on: push`/`schedule`/`workflow_dispatch`). GENERALISES + makes ABSOLUTE the §11.4.75 Layer-5 posture across ALL governed repos; enforcement migrates to the LOCAL §11.4.75 git-hook ritual + §11.4.40 pre-tag sweep, never a remote runner. ALL hold: **(A)** zero active root-level `.github/workflows/*.yml|yaml` / `.gitlab-ci.yml` / equivalent; **(B)** "disabled" = a push triggers ZERO runs — delete OR rename to a non-trigger name (the §11.4.75 `.disabled-local-only` pattern); a live-`on:`+`if:false` workflow is NOT compliant; **(C)** scope = repos we author+push — vendored/third-party nested configs are INERT, OUT of scope (§11.4.29 vendor-exempt); **(D)** no new CI may be added; **(E)** pre-push verify the tracked workflow set is empty for authored repos. Honest boundary (§11.4.6): file-level disabling stops FILE-triggered runs, NOT provider-side server settings (org-default required workflows, branch-protection checks) — the operator turns those off. Composes §11.4.75/.29/.6/.40/.42/.109/.113/§2.1. Classification: universal (§11.4.17). The parent repo currently has NO active CI workflow files — already §11.4.156-compliant. Canonical authority: constitution submodule `Constitution.md` §11.4.156. Non-compliance is a release blocker.

**§11.4.157 — GEMINI.md maintained in lockstep with CLAUDE.md / AGENTS.md / QWEN.md (User mandate, 2026-06-15).** `GEMINI.md` is a FIRST-CLASS governance context carrier EQUAL to `CLAUDE.md`/`AGENTS.md`/`QWEN.md`, never optional/best-effort. ALL hold: **(A)** five-carrier lockstep — no governance change is complete until `GEMINI.md` carries it alongside the other three mirrors (added to the §11.4.26 propagation + cross-reference set explicitly); **(B)** no silent drift — `GEMINI.md` lagging the other mirrors' highest rule is a §11.4.157 violation (§11.4.65-class), back-fill required; **(C)** equal status — `GEMINI.md` restates the SAME literal `11.4.N` anchors the propagation gates require (§11.4.35), fleet count INCLUDES `GEMINI.md`; **(D)** consumer projects' own CLAUDE/AGENTS/QWEN/GEMINI bind too (§11.4.35). Honest boundary (§11.4.6): claiming `GEMINI.md` "in sync" while a back-fill is incomplete is itself a §11.4.157 violation. Composes §11.4.26/.35/.17/.44/.65/.140/.156. Classification: universal (§11.4.17). This parent repo now maintains `CLAUDE.md` + `AGENTS.md` + `QWEN.md` + `GEMINI.md` in lockstep alongside `CONSTITUTION.md`; every future governance edit MUST land in all carriers. Canonical authority: constitution submodule `Constitution.md` §11.4.157. Non-compliance is a release blocker.
