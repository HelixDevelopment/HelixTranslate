# pre_build_verification.sh — companion guide

**Revision:** 2
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
| `CM-SCRIPT-TARGET-SHELL-PARSEABLE` | §11.4.67 | Every tracked `*.sh` under `scripts/` + `scripts/testing/` parses cleanly under `bash -n`; any script declaring an **sh-family shebang** (`#!/bin/sh`, `#!/usr/bin/env sh`) or **no recognised shell shebang** must ALSO parse under `sh -n`. A script with an honest **bash** shebang is only required to be bash-parseable (invoked via its shebang it never runs under `sh`), so its bash-only constructs (`mapfile`, `< <(...)`, `[[ ]]`, arrays) are legitimate and must NOT be false-FAILed. |
| `CM-VERSION-SINGLE-SOURCE` | P0.1 / `a36030e` | No `cmd/*/main.go` declares a hardcoded semver version literal (e.g. `appVersion = "3.0.0"`); every binary's version derives from `pkg/version.AppVersion` (== authoritative `VERSION` file). A fast grep complement of the Go test `TestNoBinaryDeclaresDivergentVersionLiteral`. |

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
- **CM-SCRIPT-TARGET-SHELL-PARSEABLE** is faithful to §11.4.67's "every shell
  script that may be invoked under a target shell OTHER than the one in its
  shebang MUST parse cleanly under that target shell." It does NOT force `sh -n`
  on an honest-`bash` script, because such a script invoked via its shebang only
  ever runs under bash — doing so would itself be a §11.4.1 FAIL-bluff (the
  project's `scripts/demo-all.sh` is exactly this case: genuine bash with
  `mapfile` + process substitution, correctly PASSes). It requires `bash` AND
  `sh` on `PATH`; if either is absent it returns `2` (harness error), never a
  false PASS. Scope is the tracked `scripts/` tree (incl. `scripts/testing/`).
- **CM-VERSION-SINGLE-SOURCE** is grep-based: it matches a Go assignment of a
  3-part semver string literal to an identifier named `(app)version`. A version
  built at runtime (`fmt.Sprintf`) or held in a differently-named identifier is
  NOT caught; a compliant `appVersion = version.AppVersion` reference and a
  2-part XML/EPUB attr (`version="1.0"`) are correctly NOT flagged. It and the
  Go test `TestNoBinaryDeclaresDivergentVersionLiteral` together are the
  high-value probe, not an AST-grade proof.

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
- `scripts/testing/meta_test_script_target_shell_parseable.sh` — paired §1.1
  mutation proof for `CM-SCRIPT-TARGET-SHELL-PARSEABLE`.
- `scripts/testing/meta_test_version_single_source.sh` — paired §1.1 mutation
  proof for `CM-VERSION-SINGLE-SOURCE`.
- `scripts/testing/meta_test_constitution_inheritance.sh` — the pre-existing
  inheritance meta-test (sibling discipline).
- `scripts/commit_all.sh` — the authorised commit + push wrapper.

**Last verified:** 2026-06-14 (all 4 gates PASS on current tree; all four paired
mutation tests PASS — every gate FAILs on a real violation and PASSes when
restored, and the two false-FAIL negatives confirm honest-bash scripts and the
compliant `version.AppVersion` form are never wrongly flagged).
