#!/usr/bin/env bash
#
# ============================================================================
# commit_all.sh — project commit-wrapper for helix_translate (§11.4.22)
# ============================================================================
#
# Purpose:
#   The single authorised commit + push entry point for the main repo. It
#   encodes — mechanically — the exact git discipline the Helix Constitution
#   this project inherits requires, so the conductor never has to remember it
#   by hand:
#     * §11.4.113  ABSOLUTE no-force-push   — refuses --force / --force-with-lease / +<ref>.
#     * §2.1       Multi-upstream push      — pushes to ALL distinct push URLs.
#     * §11.4.71   Fetch-before-push + FF   — fetches first; STOPS on non-fast-forward
#                  / §11.4.113                (never auto-merges, never force-pushes).
#     * §11.4.84   Working-tree quiescence  — aborts if mutation markers are staged/present.
#     * §11.4.30   No `git add -A`          — requires explicit pathspecs; refuses none.
#                                             NEVER stages the `helix_qa` submodule.
#     * §11.4.88   Background push          — --sync-push (default) or detached push;
#                                             commit lock released right after commit.
#     * §11.4.67   Target-shell parseable   — passes `bash -n` AND `sh -n`.
#
# Usage:
#   scripts/commit_all.sh -m "<message>" <pathspec> [<pathspec>...]
#   scripts/commit_all.sh -m "<message>" --background <pathspec>...
#
# Inputs (arguments / flags):
#   -m, --message <msg>   Commit message (required).
#   --sync-push           Push synchronously and report push exit code (DEFAULT).
#   --background          Push detached (nohup &+disown); orchestrator exit = COMMIT result.
#   --no-push             Commit only; do not push (used by tests / staging-only flows).
#   <pathspec>...         Explicit paths to stage. At least one required (§11.4.30).
#
# Environment:
#   COMMIT_ALL_REPO_ROOT  Override repo root (tests point this at a throwaway repo).
#   COMMIT_ALL_REMOTE     Restrict push to a single named remote (tests use a local bare).
#   COMMIT_ALL_NO_VERIFY_OK   (unset) — hooks are never skipped by this wrapper.
#
# Outputs:
#   stdout  Human-readable progress + a final SUMMARY line.
#   Exit codes:
#     0  commit (and, in --sync-push, push) succeeded
#     2  usage error / forbidden flag / no pathspec / mutation residue / non-FF
#     3  nothing to commit (informational — mirrors §11.4.22 doc-sync convention)
#     1  unexpected git failure
#
# Side-effects:
#   Acquires an flock-style lock at "<repo>/.git/.commit_all.lock", stages the
#   given pathspecs, creates ONE commit, releases the lock immediately, then
#   fetches + fast-forward-pushes to every distinct push URL.
#
# Dependencies:
#   bash (>=3.2 compatible constructs), git. `flock` used when present; falls
#   back to an atomic mkdir lock on hosts without it (macOS).
# ============================================================================

set -euo pipefail

# ----------------------------------------------------------------------------
# Constants
# ----------------------------------------------------------------------------
PROG="commit_all.sh"
LOCK_NAME=".commit_all.lock"
# §11.4.30 — submodule that MUST NEVER be staged by this wrapper.
FORBIDDEN_PATHSPEC="helix_qa"
# §11.4.84 — mutation residue markers. A staged/working-tree hit aborts the commit.
# Each alternative wraps ONE character in a regex char-class (e.g. a trailing
# [D]) so this DEFINITION line never self-matches the scan when commit_all.sh is
# itself the staged pathspec — the wrapper that defines the markers must not flag
# its own definition (§11.4.120 false-positive). A char-class does NOT change
# what the regex matches (the class contains exactly the one literal character),
# while the bracketed text here is not itself a match. No comment in this file
# may contain a literal marker verbatim, or it would re-introduce a self-match.
MUTATION_MARKERS='MUTATE[D] for paired|// alway[s] pass|# alway[s] pass|_mutate[d]_|<<<<<<[<] |^======[=]$|>>>>>>[>] '

# ----------------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------------
die() {
  # $1 = exit code, $2.. = message
  code="$1"; shift
  printf '%s: ERROR: %s\n' "$PROG" "$*" >&2
  exit "$code"
}

info() { printf '%s: %s\n' "$PROG" "$*"; }

usage() {
  cat >&2 <<'EOF'
Usage: commit_all.sh -m "<message>" [--sync-push|--background|--no-push] <pathspec> [<pathspec>...]
  Refuses --force / --force-with-lease / +<ref> (§11.4.113).
  Refuses no pathspec and `-A` / `--all` (§11.4.30).
EOF
  exit 2
}

# §11.4.113 — reject any force-push / history-rewrite token anywhere in argv.
reject_force_tokens() {
  for _arg in "$@"; do
    case "$_arg" in
      --force|--force-with-lease|--force-with-lease=*|-f)
        die 2 "§11.4.113 ABSOLUTE no-force-push: refusing forbidden flag '$_arg'"
        ;;
      +*)
        # A leading '+' refspec (e.g. +main:main) is a forced push refspec.
        die 2 "§11.4.113 ABSOLUTE no-force-push: refusing forced refspec '$_arg'"
        ;;
    esac
  done
}

# §11.4.30 — reject add-everything tokens; pathspecs must be explicit.
reject_add_all_tokens() {
  for _arg in "$@"; do
    case "$_arg" in
      -A|--all|.|:/|':(top)')
        die 2 "§11.4.30 NEVER 'git add -A': refusing add-everything token '$_arg' — pass explicit pathspecs"
        ;;
    esac
  done
}

# ----------------------------------------------------------------------------
# Argument parsing
# ----------------------------------------------------------------------------
MESSAGE=""
PUSH_MODE="sync"   # sync | background | none
PATHSPECS=()

# Guard the whole argv BEFORE interpreting it, so a forbidden token can never
# slip through option parsing.
reject_force_tokens "$@"

while [ "$#" -gt 0 ]; do
  case "$1" in
    -m|--message)
      [ "$#" -ge 2 ] || die 2 "missing argument for $1"
      MESSAGE="$2"; shift 2 ;;
    --sync-push)   PUSH_MODE="sync"; shift ;;
    --background)  PUSH_MODE="background"; shift ;;
    --no-push)     PUSH_MODE="none"; shift ;;
    -A|--all)
      # §11.4.30 — explicit, cited rejection of add-everything (caught here so the
      # message cites the rule rather than the generic unknown-flag branch).
      die 2 "§11.4.30 NEVER 'git add -A': refusing add-everything flag '$1' — pass explicit pathspecs" ;;
    -h|--help)     usage ;;
    --)            shift; while [ "$#" -gt 0 ]; do PATHSPECS+=("$1"); shift; done ;;
    -*)
      # Any unknown dash-flag is rejected rather than passed to git blindly.
      die 2 "unknown flag '$1'" ;;
    *)
      PATHSPECS+=("$1"); shift ;;
  esac
done

[ -n "$MESSAGE" ] || die 2 "commit message required (-m \"<message>\")"

# §11.4.30 — at least one explicit pathspec, and none of the add-everything tokens.
if [ "${#PATHSPECS[@]}" -eq 0 ]; then
  die 2 "§11.4.30 refusing to run with no pathspec — staging is explicit-only"
fi
reject_add_all_tokens "${PATHSPECS[@]}"

# §11.4.30 — never stage the helix_qa submodule, however it is spelled.
for _ps in "${PATHSPECS[@]}"; do
  case "$_ps" in
    "$FORBIDDEN_PATHSPEC"|"$FORBIDDEN_PATHSPEC"/*|*/"$FORBIDDEN_PATHSPEC"|*/"$FORBIDDEN_PATHSPEC"/*)
      die 2 "§11.4.30 refusing to stage forbidden submodule path '$_ps' ($FORBIDDEN_PATHSPEC)" ;;
  esac
done

# ----------------------------------------------------------------------------
# Locate repo root
# ----------------------------------------------------------------------------
REPO_ROOT="${COMMIT_ALL_REPO_ROOT:-$(cd "$(dirname "$0")/.." 2>/dev/null && pwd)}"
[ -n "$REPO_ROOT" ] || die 1 "cannot determine repo root"
cd "$REPO_ROOT" || die 1 "cannot cd to repo root '$REPO_ROOT'"
git rev-parse --git-dir >/dev/null 2>&1 || die 1 "not a git repository: $REPO_ROOT"

GIT_DIR="$(git rev-parse --git-dir)"
LOCK_PATH="$GIT_DIR/$LOCK_NAME"

# ----------------------------------------------------------------------------
# §11.4.84 — working-tree quiescence: refuse mutation residue.
# ----------------------------------------------------------------------------
quiescence_check() {
  # Scan ONLY the files we are about to stage (plus already-staged content),
  # so unrelated working-tree noise never blocks an explicit, scoped commit.
  _hits=""
  for _ps in "${PATHSPECS[@]}"; do
    # -I (ignore-binary): mutation markers are a TEXT/source concern. Without it a
    # committed binary artifact (e.g. the .pdf/.html exports §11.4.65 produces) whose
    # bytes happen to contain a marker like "=======" would false-positive-abort
    # (W14b review finding — the text-only self-test missed this).
    if [ -f "$_ps" ]; then
      if LC_ALL=C grep -InE "$MUTATION_MARKERS" "$_ps" >/dev/null 2>&1; then
        _hits="$_hits $_ps"
      fi
    elif [ -d "$_ps" ]; then
      # §11.4.120: a submodule path is a gitlink — staging it records ONLY the
      # submodule's HEAD SHA, never its working-tree files. Recursively scanning
      # the submodule tree here is a false-positive (the submodule's own
      # quiescence is enforced by ITS own scoped commit). A submodule pathspec
      # is exactly one ls-files entry with mode 160000; skip it. Ordinary
      # directories (>1 entry, or non-gitlink mode) are still scanned in full.
      _ls="$(git ls-files -s -- "$_ps" 2>/dev/null)"
      _nlines="$(printf '%s\n' "$_ls" | sed '/^$/d' | wc -l | tr -d ' ')"
      _mode1="$(printf '%s\n' "$_ls" | awk 'NR==1{print $1}')"
      if [ "$_nlines" = "1" ] && [ "$_mode1" = "160000" ]; then
        continue
      fi
      if LC_ALL=C grep -rInE "$MUTATION_MARKERS" "$_ps" >/dev/null 2>&1; then
        _found="$(LC_ALL=C grep -rIlE "$MUTATION_MARKERS" "$_ps" 2>/dev/null | tr '\n' ' ')"
        _hits="$_hits $_found"
      fi
    fi
  done
  if [ -n "${_hits// /}" ]; then
    die 2 "§11.4.84 mutation residue detected in staged scope:${_hits} — ABORT (account for it or clean it first)"
  fi
}
quiescence_check

# ----------------------------------------------------------------------------
# Lock acquisition (released immediately after commit per §11.4.88)
# ----------------------------------------------------------------------------
LOCK_FD=""
LOCK_DIR=""
acquire_lock() {
  if command -v flock >/dev/null 2>&1; then
    exec 9>"$LOCK_PATH"
    LOCK_FD=9
    flock -n 9 || die 1 "another commit_all.sh holds the lock ($LOCK_PATH)"
  else
    # Atomic mkdir fallback (macOS has no flock).
    LOCK_DIR="${LOCK_PATH}.d"
    mkdir "$LOCK_DIR" 2>/dev/null || die 1 "another commit_all.sh holds the lock ($LOCK_DIR)"
  fi
}
release_lock() {
  if [ -n "$LOCK_FD" ]; then
    eval "exec ${LOCK_FD}>&-" 2>/dev/null || true
    LOCK_FD=""
  fi
  if [ -n "$LOCK_DIR" ] && [ -d "$LOCK_DIR" ]; then
    rmdir "$LOCK_DIR" 2>/dev/null || true
    LOCK_DIR=""
  fi
}
trap release_lock EXIT

acquire_lock

# ----------------------------------------------------------------------------
# Stage + commit
# ----------------------------------------------------------------------------
info "staging ${#PATHSPECS[@]} explicit pathspec(s)"
git add -- "${PATHSPECS[@]}"

if git diff --cached --quiet; then
  info "nothing staged to commit (exit 3, informational)"
  release_lock
  exit 3
fi

info "committing"
# Hooks are NEVER skipped here (no --no-verify). git commit must succeed.
if ! git commit -m "$MESSAGE"; then
  die 1 "git commit failed"
fi
COMMIT_SHA="$(git rev-parse HEAD)"
info "commit landed: $COMMIT_SHA"

# §11.4.88 — release the lock the instant the commit is durable on local disk,
# BEFORE any (potentially slow) push.
release_lock
trap - EXIT

if [ "$PUSH_MODE" = "none" ]; then
  printf '%s: SUMMARY commit=%s push=skipped\n' "$PROG" "$COMMIT_SHA"
  exit 0
fi

# ----------------------------------------------------------------------------
# Determine distinct push URLs (§2.1) — dedup so each physical repo pushed once.
# ----------------------------------------------------------------------------
collect_push_remotes() {
  # Emit "remote<TAB>url" for one representative remote per distinct push URL.
  # If COMMIT_ALL_REMOTE is set, restrict to it (tests use a local bare remote).
  if [ -n "${COMMIT_ALL_REMOTE:-}" ]; then
    _url="$(git remote get-url --push "$COMMIT_ALL_REMOTE" 2>/dev/null | head -1)"
    [ -n "$_url" ] || die 1 "COMMIT_ALL_REMOTE='$COMMIT_ALL_REMOTE' has no push URL"
    printf '%s\t%s\n' "$COMMIT_ALL_REMOTE" "$_url"
    return 0
  fi
  _seen=""
  for _r in $(git remote); do
    # A remote may carry multiple push URLs; iterate each.
    while IFS= read -r _u; do
      [ -n "$_u" ] || continue
      case " $_seen " in
        *" $_u "*) : ;;                       # already covered this URL
        *) _seen="$_seen $_u"; printf '%s\t%s\n' "$_r" "$_u" ;;
      esac
    done <<EOF
$(git remote get-url --push --all "$_r" 2>/dev/null)
EOF
  done
}

BRANCH="$(git rev-parse --abbrev-ref HEAD)"

# §11.4.71 / §11.4.113 — fetch-before-push, fast-forward-only push to one URL.
push_one() {
  # $1 = remote name, $2 = url
  _remote="$1"; _url="$2"
  info "fetch-before-push: $_remote ($_url)"
  if ! git fetch "$_remote" >/dev/null 2>&1; then
    info "WARN: fetch failed for $_remote — proceeding to ff-only push attempt"
  fi
  # If the remote tracking branch exists and is NOT an ancestor of HEAD's
  # upstream-merge-base, a plain push would be non-FF. We rely on git's own
  # fast-forward enforcement: NO --force is ever passed, so a non-FF push is
  # rejected by git, and we surface it as a STOP (§11.4.113 — never auto-merge).
  _remote_ref="refs/remotes/$_remote/$BRANCH"
  if git rev-parse --verify --quiet "$_remote_ref" >/dev/null 2>&1; then
    _remote_sha="$(git rev-parse "$_remote_ref")"
    if ! git merge-base --is-ancestor "$_remote_sha" HEAD 2>/dev/null; then
      die 2 "§11.4.71/§11.4.113 non-fast-forward: $_remote/$BRANCH ($_remote_sha) is not an ancestor of HEAD — STOP. Integrate manually (fetch + merge-onto-latest-$BRANCH); never force-push."
    fi
  fi
  info "fast-forward push: $_remote $BRANCH"
  # Explicit refspec, NO leading '+', NO --force.
  git push "$_remote" "$BRANCH:$BRANCH"
}

do_push_all() {
  _failed=0
  while IFS="$(printf '\t')" read -r _remote _url; do
    [ -n "$_remote" ] || continue
    if ! push_one "$_remote" "$_url"; then
      info "WARN: push to $_remote ($_url) failed"
      _failed=$((_failed + 1))
    fi
  done <<EOF
$(collect_push_remotes)
EOF
  return "$_failed"
}

if [ "$PUSH_MODE" = "background" ]; then
  # §11.4.88 — detached push; orchestrator exit reports COMMIT success.
  _log="$GIT_DIR/commit_all.push.$(date +%Y%m%dT%H%M%S).log"
  info "background push -> $_log"
  # Re-invoke ourselves in --sync-push --no-* mode for the push only would be
  # circular; instead run the push function body in a detached subshell.
  nohup bash -c '
    set -euo pipefail
    cd "$1" || exit 1
    branch="$3"
    push_one() {
      r="$1"; git fetch "$r" >/dev/null 2>&1 || true
      rr="refs/remotes/$r/$branch"
      if git rev-parse --verify --quiet "$rr" >/dev/null 2>&1; then
        rs="$(git rev-parse "$rr")"
        if ! git merge-base --is-ancestor "$rs" HEAD 2>/dev/null; then
          echo "NON-FF on $r/$branch — STOP (no force)"; return 2
        fi
      fi
      git push "$r" "$branch:$branch"
    }
    shift 3
    rc=0
    while [ "$#" -gt 0 ]; do push_one "$1" || rc=1; shift; done
    exit "$rc"
  ' _ "$REPO_ROOT" "$GIT_DIR" "$BRANCH" \
    $(collect_push_remotes | cut -f1 | tr '\n' ' ') \
    >"$_log" 2>&1 &
  disown || true
  printf '%s: SUMMARY commit=%s push=background log=%s\n' "$PROG" "$COMMIT_SHA" "$_log"
  exit 0
fi

# Synchronous push (default).
if do_push_all; then
  printf '%s: SUMMARY commit=%s push=ok\n' "$PROG" "$COMMIT_SHA"
  exit 0
else
  printf '%s: SUMMARY commit=%s push=partial-failure (commit is durable locally)\n' "$PROG" "$COMMIT_SHA"
  exit 1
fi
