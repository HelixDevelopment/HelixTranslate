# meta_test_no_fakes_beyond_unit.sh — companion guide

**Revision:** 1
**Last modified:** 2026-06-14T00:00:00Z

## Overview

Paired §1.1 mutation proof for the `CM-NO-FAKES-BEYOND-UNIT` gate in
`scripts/pre_build_verification.sh`. PROVES the gate is not a bluff
(§1.1 / §11.4.27): a non-unit (build-tagged) Go test importing a mock/stub path
must FAIL the gate, while a unit test importing a mock must stay PASS.

## Prerequisites

- `bash`, `git`, `grep`, `head` on `PATH`.

## Usage

```bash
bash scripts/testing/meta_test_no_fakes_beyond_unit.sh
```

Exit codes: `0` gate proven sound · `1` gate is a bluff · `2` harness error.

## Internal behaviour

Builds a disposable git repo under `mktemp`, pointed at the gate via
`PBV_REPO_ROOT`; tmp dir removed via `trap` on every exit. Cases exercised:

1. baseline: a UNIT test (no build tag) importing `…/mocks` stays PASS — proves
   no false positive on units (§11.4.27 permits mocks in unit tests only).
2. an `//go:build integration` test importing `…/mocks` → FAIL → remove → PASS.
3. an `//go:build e2e` test importing `…/stubs` → FAIL → remove → PASS.

## Edge cases / honest limitations

- The gate (and therefore this proof) is line-grep based, not AST-based; it
  detects import-path string literals. Mocks referenced indirectly are not
  caught. See `docs/scripts/pre_build_verification.sh.md` for the full
  limitation note.

## Related scripts

- `scripts/pre_build_verification.sh` — the gate under test.

**Last verified:** 2026-06-14 (META-TEST RESULT: PASS).
