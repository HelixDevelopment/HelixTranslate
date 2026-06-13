# commit_all.sh — companion guide

**Revision:** 1
**Last modified:** 2026-06-13T00:00:00Z

## Overview

`scripts/commit_all.sh` is the project's **single authorised commit + push entry
point** for the main repo, mandated by Constitution §11.4.22 (document-sync /
official commit wrapper) and the "MANDATORY COMMIT & PUSH CONSTRAINTS" section
of `CLAUDE.md` ("NEVER use `git add`, `git commit`, or `git push` directly").

It mechanically encodes the git discipline the inherited Helix Constitution
requires, so the conductor never enforces it by hand. Companion to §11.4.18
(script documentation) and validated by `scripts/commit_all_selftest.sh`
(§11.4.27 / §11.4.85 anti-bluff).

## Prerequisites

- `bash` and `git` on `PATH`.
- A configured git identity (`user.name`, `user.email`).
- Remotes configured (this repo's remotes resolve to two distinct push URLs:
  `git@github.com:milos85vasic/Translator.git` and
  `git@github.com:HelixDevelopment/HelixTranslate.git`).
- `flock` is used when present; on macOS (no `flock`) an atomic `mkdir` lock is
  used instead.

## Usage examples

```bash
# Commit explicit paths and push synchronously (default) to every distinct push URL:
scripts/commit_all.sh -m "feat(fb2): preserve tail text in nested elements" \
    pkg/fb2/parser.go pkg/ebook/fb2_parser.go

# Commit + detached background push (releases the commit lock immediately,
# orchestrator exit reports COMMIT success per §11.4.88):
scripts/commit_all.sh -m "docs: update architecture notes" --background docs/ARCHITECTURE.md

# Commit only, no push (staging-only / offline):
scripts/commit_all.sh -m "wip: local checkpoint" --no-push pkg/translator/llm/openai.go
```

### Flags

| Flag | Meaning |
|---|---|
| `-m, --message <msg>` | Commit message (**required**). |
| `--sync-push` | Push synchronously, report push exit code (**default**). |
| `--background` | Detached push (`nohup &`+`disown`); exit reports the COMMIT result. |
| `--no-push` | Commit only; do not push. |
| `<pathspec>...` | **Explicit** paths to stage. At least one required. |

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Commit succeeded (and, in `--sync-push`, push succeeded). |
| `2` | Usage error / forbidden flag / no pathspec / mutation residue / non-fast-forward STOP. |
| `3` | Nothing staged to commit (informational, mirrors §11.4.22). |
| `1` | Unexpected git failure (commit failed, or a sync push partially failed). |

## Constitutional discipline encoded

| Rule | Mechanism in the script |
|---|---|
| **§11.4.113** absolute no-force-push | `reject_force_tokens` (run on the whole argv first) refuses `--force`, `--force-with-lease[=…]`, `-f`, and any leading-`+` refspec. The push step passes an explicit `<branch>:<branch>` refspec with **no** `--force`. |
| **§2.1** multi-upstream push | `collect_push_remotes` enumerates every remote's push URLs and **de-duplicates by URL**, so each physical repo is pushed exactly once (the 5 remote/URL entries collapse to the 2 distinct repos). |
| **§11.4.71 / §11.4.113** fetch-before-push + fast-forward-only | `push_one` runs `git fetch <remote>` first; if the remote-tracking branch is **not** an ancestor of `HEAD`, it **STOPs** (exit 2) and tells the operator to integrate manually — it never auto-merges and never force-pushes. |
| **§11.4.84** working-tree quiescence | `quiescence_check` greps the staged scope for mutation markers (`MUTATED for paired`, `// always pass`, `_mutated_`, conflict markers) and ABORTs (exit 2) on any unaccounted hit. |
| **§11.4.30** never `git add -A`; never `helix_qa` | Requires ≥1 explicit pathspec (refuses none); explicitly rejects `-A` / `--all` / `.` / `:/`; and refuses the `helix_qa` submodule path in any spelling. |
| **§11.4.88** background push + early lock release | The `.git/.commit_all.lock` is released the instant `git commit` returns 0 (before any push). `--background` detaches the push via `nohup … & ; disown`, logging to `.git/commit_all.push.<ts>.log`. |
| **§11.4.67** target-shell parseable | `#!/usr/bin/env bash`, `set -euo pipefail`, passes both `bash -n` and `sh -n`. |

## Edge cases

- **Nothing to commit** → exit 3 (not an error); the lock is released, no push.
- **Non-fast-forward** on any remote → STOP (exit 2); the local commit is already
  durable. Integrate with `git fetch` + merge-onto-latest-`<branch>` and re-run.
- **Partial sync-push failure** → exit 1, but the SUMMARY notes the commit is
  durable locally.
- **Hooks** are never skipped — the wrapper never passes `--no-verify`.

## Internal behaviour

1. Guard argv for force tokens (§11.4.113) **before** option parsing.
2. Parse flags + pathspecs; enforce §11.4.30 (explicit-only, no `helix_qa`).
3. Locate repo root (override via `COMMIT_ALL_REPO_ROOT`), verify it is a git repo.
4. §11.4.84 quiescence scan of the staged scope.
5. Acquire the commit lock, stage the explicit pathspecs, create one commit.
6. Release the lock immediately (§11.4.88).
7. Sync or background push, fast-forward-only, to every distinct push URL (§2.1).

### Test / environment overrides

- `COMMIT_ALL_REPO_ROOT` — point at a throwaway repo (used by the self-test).
- `COMMIT_ALL_REMOTE` — restrict pushes to one named remote (self-test uses a
  local bare remote so **no real remote is ever touched**).

## Related scripts

- `scripts/commit_all_selftest.sh` — hermetic anti-bluff test (builds a throwaway
  repo + local bare remote under `$TMPDIR`, asserts every guard fires; **no real
  push**).
- `scripts/no-silent-skips.sh`, `scripts/testing/meta_test_constitution_inheritance.sh`
  — sibling Definition-of-Done / constitution-inheritance enforcement scripts.

## Last verified

2026-06-13 — `bash -n` + `sh -n` clean; self-test `PASS=13 FAIL=0`.
