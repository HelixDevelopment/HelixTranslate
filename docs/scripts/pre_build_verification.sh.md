# pre_build_verification.sh — companion guide

**Revision:** 1
**Last modified:** 2026-06-14T00:00:00Z

## Overview

`scripts/pre_build_verification.sh` is the project's pre-build gate-suite
runner (WORKING_PLAN item P4.2). It runs a small set of GENUINE,
mechanically-checkable `CM-*` gates BEFORE every build, per the inherited
Constitution's **MANDATORY TESTING CONSTRAINTS** ("Pre-build / pre-merge
verification — BEFORE every build"). It is the first real `CM-*` gate runner
for this project; before P4.2 only the constitution-inheritance meta-test
existed.

Each gate fails ONLY on a real, user-visible repository-hygiene violation —
never on a script-internal bug (§11.4.1) — and each is backed by a PAIRED
§1.1 mutation test under `scripts/testing/` that proves the gate is not a
bluff.

## Implemented gates

| Gate ID | Anchor | What it asserts |
|---|---|---|
| `CM-GITIGNORE-PRECOMMIT-AUDIT` | §11.4.30 | No tracked file matches the forbidden build-artifact / secret classes the project's `.gitignore` declares. Includes the **cmd/api-server anchoring regression guard**: the prebuilt root `api-server` binary must stay anchored as `/api-server` so the bare token never re-hides the SOURCE directory. |
| `CM-NO-FAKES-BEYOND-UNIT` | §11.4.27 | Non-unit Go test files (build-tagged `integration`/`e2e`/`stress`/`performance`/`security`) do not import a mock/stub/fake package path. Mocks are permitted in unit tests only. |

## Prerequisites

- `sh` (or `bash`) and `git` on `PATH`. `grep`, `head`.
- Run from anywhere inside the repo; the script resolves the git toplevel.
- No `pandoc` / `weasyprint` / `go` required — the suite is git-index + grep
  only, and runs in well under a second (no build, no network).

## Usage examples

```bash
# Run the whole suite (CI / pre-build hook):
scripts/pre_build_verification.sh

# List gate ids:
scripts/pre_build_verification.sh --list

# Run a single gate:
scripts/pre_build_verification.sh --gate CM-GITIGNORE-PRECOMMIT-AUDIT
```

Exit codes: `0` all selected gates passed · `1` at least one real violation ·
`2` usage / harness error (unknown gate, not a git repo).

## Edge cases & honest limitations (§11.4.6)

- **CM-GITIGNORE-PRECOMMIT-AUDIT** mirrors the classes this project's
  `.gitignore` declares (the named `build/*` outputs, real `.env` files,
  `config_with_keys.json` / `api_keys.json` / `secrets.*` / `keys.*`). It does
  NOT flag every conceivable build artifact (e.g. arbitrary `*.exe`), because
  the project's declared policy is class-specific and the working tree is
  compliant with that declared policy. Widening the forbidden set is future
  work and would require the tree to be brought into compliance first.
- `.env.example` is explicitly allow-listed (the project's `.gitignore` carries
  `!.env.example`) and must NOT be flagged.
- **CM-NO-FAKES-BEYOND-UNIT** is line-grep based, not AST-based. It matches
  import-path string literals ending in `/mock(s)`, `/stub(s)`, `/fake(s)` in
  files carrying a non-unit build tag in their first 15 lines. A mock referenced
  indirectly (a re-exported type from a differently-named package) is NOT
  caught. Untagged `_test.go` files are treated as unit tests (mocks allowed),
  matching the convention that integration/e2e suites carry build tags. This is
  a high-value cheap probe, not a complete proof; AST-grade enforcement is
  tracked as future work.

## Internal behaviour

The script is read-only over the git index + working tree (no writes, no
build, no network). It is written in POSIX-portable shell (no arrays, no
`[[ ]]`, no process substitution) and parses cleanly under both `bash -n` and
`sh -n` per §11.4.67.

The mutation tests build disposable git repos under `mktemp` and point the gate
at them via the `PBV_REPO_ROOT` env var, so the real working tree is never
mutated.

## Related scripts

- `scripts/testing/meta_test_gitignore_precommit_audit.sh` — paired §1.1
  mutation proof for `CM-GITIGNORE-PRECOMMIT-AUDIT`.
- `scripts/testing/meta_test_no_fakes_beyond_unit.sh` — paired §1.1 mutation
  proof for `CM-NO-FAKES-BEYOND-UNIT`.
- `scripts/testing/meta_test_constitution_inheritance.sh` — the pre-existing
  inheritance meta-test (sibling discipline).
- `scripts/commit_all.sh` — the authorised commit + push wrapper.

**Last verified:** 2026-06-14 (suite PASS on current tree; both paired mutation
tests PASS — every gate FAILs on a real violation and PASSes when restored).
