# Models — Constitution

> **Status:** Active. This document is the project's authoritative
> rule set. When a rule here conflicts with `CLAUDE.md`, `AGENTS.md`,
> or any guide, the Constitution wins.

## Mission

Models provides shared data types for the Helix ecosystem. It is a
lightweight module consumed by HelixTranslate, HelixQA, LLMsVerifier,
and other Helix components.

## Mandatory Standards

1. **Reproducibility:** every change is reproducible from a clean
   clone; no hidden steps.
2. **Tests track behavior, not code:** test what the user-visible
   behavior is, not what the implementation looks like.
3. **No silent skips, no silent mocks above unit tests.**
4. **Conventional Commits** for all commits.
5. **SSH-only for git operations** (`git@…`); HTTPS prohibited.

## Numbered Rules

### CONST-033 — Host Power Management is Forbidden

**Status:** Mandatory. Non-negotiable.

**Rule:** No code in this repository may invoke a host-level power-
state transition (suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, kexec) on the host machine.

### CONST-034 — Types are Canonical

**Status:** Mandatory.

**Rule:** All types in this module are canonical for the Helix
ecosystem. Changes MUST be backward-compatible or coordinated
across all consuming modules.

### CONST-035 — Anti-Bluff Testing Constitution (MANDATORY)

**Status:** Mandatory. Non-negotiable.

**§35.1 The Problem (Historical Mandate)**

> "We had been in position that all tests do execute with success and
> all Challenges as well, but in reality the most of the features does
> not work and can't be used!"

**§35.2 The Six Anti-Bluff Rules**

Every test MUST:

1. **Assert on a concrete end-user-visible outcome.**
2. **Run against the real system** — mocks ONLY in unit tests.
3. **Include a matching negative assertion.**
4. **Emit copy-pasteable evidence.**
5. **Verify "fails when feature is removed"** via mutation testing.
6. **No blind shells** — no `&& echo PASS`, no `|| true`.

### CONST-036 — HelixQA is the Sole Authorized QA Tool

**Status:** Mandatory.

**Rule:** HelixQA is the only authorized tool for automated
UI/UX/API testing in modules that have UI/API surfaces.

## CONST-035 — Anti-Bluff Operative Rule (MANDATORY)

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case."

**The operative rule:** Execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product.

- A green test or challenge for a feature that does not actually work is a **BLUFF** and is **FORBIDDEN**.
- Every test must assert concrete user-visible outcomes, not just internal state.
- Every challenge must run real code and verify real behavior; grep/file-existence checks are **NOT sufficient**.
- Mutation testing is **MANDATORY**: deliberately break the feature → the test/challenge **MUST then FAIL**.
- The bar for shipping is **NOT "tests pass"** but **"users can use the feature."**
- No false-success results are tolerable.
