# meta_test_gitignore_precommit_audit.sh — companion guide

**Revision:** 1
**Last modified:** 2026-06-14T00:00:00Z

## Overview

Paired §1.1 mutation proof for the `CM-GITIGNORE-PRECOMMIT-AUDIT` gate in
`scripts/pre_build_verification.sh`. A gate without a paired mutation is itself
a Constitution violation: this harness PROVES the gate is not a bluff
(§1.1 / §11.4.30) by introducing a real violation of every forbidden class and
asserting the gate flips to FAIL, then removing it and asserting PASS.

## Prerequisites

- `bash`, `git`, `grep` on `PATH`.

## Usage

```bash
bash scripts/testing/meta_test_gitignore_precommit_audit.sh
```

Exit codes: `0` gate proven sound · `1` gate is a bluff on ≥1 class · `2`
harness error.

## Internal behaviour

Builds a disposable, COMPLIANT git repo under `mktemp` (legit source +
anchoring `.gitignore` + allow-listed `.env.example`) and points the gate at it
via `PBV_REPO_ROOT`. The real project working tree is NEVER mutated; the tmp
dir is removed on every exit path via a `trap`. Mutations exercised:

1. track `build/api-server` (declared build output) → FAIL → restore → PASS
2. track root `api-server` binary (anchoring regression) → FAIL → restore → PASS
3. track real `.env` secret file → FAIL → restore → PASS
4. track `api_keys.json` → FAIL → restore → PASS
5. negative: allow-listed `.env.example` stays PASS (no false positive)

## Edge cases

- Uses `git add -f` against the disposable repo to stage files its own
  `.gitignore` would normally exclude — that is intentional: the test simulates
  the failure mode where a forbidden file slips into version control despite the
  ignore line.

## Related scripts

- `scripts/pre_build_verification.sh` — the gate under test.

**Last verified:** 2026-06-14 (META-TEST RESULT: PASS).
