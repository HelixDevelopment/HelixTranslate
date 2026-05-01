# Security — Constitution

> **Status:** Active. This document is the project's authoritative
> rule set. When a rule here conflicts with `CLAUDE.md`, `AGENTS.md`,
> or any guide, the Constitution wins.

## Mission

Security provides security primitives for the Helix ecosystem,
including SSRF protection, input validation, and secure transport
utilities.

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

### CONST-034 — Security is Non-Negotiable

**Status:** Mandatory.

**Rule:** All security features MUST have tests that verify both
positive (allowed) and negative (blocked) cases. A security control
that cannot be shown to block an attack is not a security control.

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
