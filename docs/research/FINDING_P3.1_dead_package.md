# Finding — P3.1 §11.4.124 investigation of the DEAD `pkg/hash` package

**Revision:** 1
**Last modified:** 2026-06-14T00:00:00Z
**Item:** WORKING_PLAN.md P3.1
**Status:** Investigated — operator decision required (do NOT remove without §11.4.122 confirmation)
**Type:** Task

## Summary (FACT)

`pkg/hash/codebase_hash.go` (~393 LOC) is a **standalone `package main` command misplaced
under `pkg/`** — NOT an importable library. It declares `package main` with its own
`func main()`. A `package main` can never be imported by any other Go code, so its
"zero importers" status is structural and permanent, not a mistakenly-deleted call-site.

It is a **functional duplicate** of the real, fully-wired codebase hasher that already
ships in `pkg/version/hasher.go` (`CodebaseHasher`), which is imported across many
binaries and packages.

## Git-history evidence (where wired / when died / hidden-ref check)

- **Origin / where wired:** never wired anywhere. `git log --all --follow --diff-filter=A`
  shows the file was first (and only ever) added at its current path `pkg/hash/codebase_hash.go`
  in a single bulk commit `05a1aab "Auto-commit"`. It never lived at the repo root or under
  `cmd/`, and no prior call-site to it has ever existed in history (no `import ".../pkg/hash"`
  in any commit — `git log -S 'pkg/hash'` shows only later docs/plan commits referencing the
  string, never a Go import).
- **When/how it became dead:** it was never alive. It was committed already-orphaned as a
  `package main` under `pkg/`. There is no deleted call-site to restore.
- **Hidden-reference check (reflection/build-tags/codegen/scripts/config):** none.
  - No `//go:build`, no `// +build`, no `//go:generate` in the file.
  - No real reference in any `*.go`, `*.sh`, `Makefile`, `*.mk` (the only `pkg/hash` string
    hits outside the package itself are inside scraped GitHub HTML under
    `docs/research/helix_qa/.../pkg_tree.json` — a saved web page, not a code reference).

## Duplicate analysis — `pkg/hash` vs `pkg/version/hasher.go`

| | `pkg/hash/codebase_hash.go` (DEAD) | `pkg/version/hasher.go` (LIVE) |
|---|---|---|
| package | `main` (cannot be imported) | `version` (library) |
| type | `CodebaseHashGenerator` | `CodebaseHasher` |
| wired in | nothing | `cmd/ssh-translation`, `cmd/cli`, `cmd/translator`, `cmd/ebook-translator`, `pkg/sshworker`, `test/integration` |
| first added | `05a1aab` (Auto-commit, 2025-later) | `5dc7076` (local snapshot, 2025-11-23) — **predates** the dead one |
| purpose | sha256 every codebase file → combined hash for version sync | sha256 every codebase file → combined hash for version sync |

Both compute a sha256-of-codebase hash for distributed-worker version synchronisation.
`pkg/version.CodebaseHasher` is the canonical, wired, used implementation. `pkg/hash` is an
earlier/parallel abandoned variant that was never reachable because it is `package main`.

## Build / vet status

- `go build ./pkg/hash/` → exit 0 (compiles as a standalone command).
- `go vet ./pkg/hash/` → exit 0.
- No `_test.go` files in `pkg/hash/`.

## Decision (§11.4.124)

- **NOT wired in.** There is no library API to wire — it is `package main` with no
  importable symbols, and the capability it provides is already delivered (and used) by
  `pkg/version.CodebaseHasher`. There is no mistakenly-deleted call-site to restore and no
  never-completed wiring to finish.
- **NOT removed.** Removal of a shipped component is operator-gated per §11.4.122 — out of
  scope for this subagent. Captured here for the operator to decide.

## Recommendation for the operator

`pkg/hash` is a genuine duplicate of `pkg/version.CodebaseHasher` and is structurally
unreachable (`package main` under `pkg/`). Two defensible options:

- **[A] (Recommended) Remove `pkg/hash/`** as a duplicate/abandoned command in its own
  descriptive removal commit citing this git-history evidence, tracked as
  `Obsolete (→ Fixed.md)` reason `duplicate-of` superseded by `pkg/version.CodebaseHasher`
  (§11.4.90). Captured proof of genuine non-need: zero importers (structural —
  `package main`), no hidden refs, functional duplicate of a live wired hasher.
- **[B] Relocate to `cmd/`** if a standalone CLI codebase-hash tool is wanted — move
  `pkg/hash/codebase_hash.go` to e.g. `cmd/codebase-hash/main.go` so the misplaced
  command lives where commands belong. (Still leaves two hasher implementations to
  maintain; option A is cleaner.)
