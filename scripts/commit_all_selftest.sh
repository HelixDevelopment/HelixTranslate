#!/usr/bin/env bash
#
# ============================================================================
# commit_all_selftest.sh — hermetic anti-bluff test for commit_all.sh (§11.4.27, §11.4.85)
# ============================================================================
#
# Purpose:
#   Prove commit_all.sh REALLY enforces its constitutional guards, against a
#   THROWAWAY git repo + a local bare remote, both created under a temp dir.
#   NO real project remote is ever touched and NO real push is ever performed.
#
# Usage:
#   scripts/commit_all_selftest.sh
#
# Inputs:    none.
# Outputs:   per-case PASS/FAIL lines; final SELFTEST SUMMARY; exit 0 iff all PASS.
# Side-effects: creates + removes a temp dir under $TMPDIR (trap cleanup on exit).
# Dependencies: bash, git, the sibling commit_all.sh.
# ============================================================================

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
WRAPPER="$HERE/commit_all.sh"
[ -x "$WRAPPER" ] || { echo "FATAL: $WRAPPER not executable"; exit 1; }

WORK="$(mktemp -d "${TMPDIR:-/tmp}/commit_all_selftest.XXXXXX")"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT

PASS=0
FAIL=0
ok()   { printf '  PASS  %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  FAIL  %s\n' "$*"; FAIL=$((FAIL + 1)); }

# Assert a wrapper invocation exits with an expected code (and optionally that
# its output matches a pattern). Runs against the throwaway repo via env vars.
run_wrapper() {
  # echoes nothing; sets RC and OUT globals. `set +e` so a non-zero wrapper
  # exit is captured, not propagated through the harness's `set -e`.
  set +e
  OUT="$(COMMIT_ALL_REPO_ROOT="$REPO" COMMIT_ALL_REMOTE="origin" "$WRAPPER" "$@" 2>&1)"
  RC=$?
  set -e
  return 0
}

# ----------------------------------------------------------------------------
# Build throwaway repo + bare local remote
# ----------------------------------------------------------------------------
BARE="$WORK/remote.git"
REPO="$WORK/repo"
git init --quiet --bare "$BARE"
git init --quiet "$REPO"
git -C "$REPO" config user.email "selftest@example.invalid"
git -C "$REPO" config user.name  "selftest"
git -C "$REPO" config commit.gpgsign false
git -C "$REPO" remote add origin "$BARE"
# Seed an initial commit so a branch + remote ref exist.
echo "seed" > "$REPO/seed.txt"
git -C "$REPO" add seed.txt
git -C "$REPO" commit --quiet -m "seed"
git -C "$REPO" branch -M main
git -C "$REPO" push --quiet origin main
echo "== throwaway repo: $REPO  bare remote: $BARE =="

# ----------------------------------------------------------------------------
# Case 1: happy path — explicit pathspec, ff-push to local bare succeeds, commit lands
# ----------------------------------------------------------------------------
echo "feature-A" > "$REPO/a.txt"
run_wrapper -m "feat: add a.txt" a.txt
if [ "$RC" -eq 0 ] && printf '%s' "$OUT" | grep -q 'push=ok'; then
  ok "case1 happy-path: exit 0 + push=ok"
else
  bad "case1 happy-path: rc=$RC out=$OUT"
fi
# commit-landed assertion: HEAD subject matches AND bare remote advanced to it
LOCAL_HEAD="$(git -C "$REPO" rev-parse HEAD)"
BARE_MAIN="$(git -C "$BARE" rev-parse refs/heads/main)"
if [ "$(git -C "$REPO" log -1 --pretty=%s)" = "feat: add a.txt" ]; then
  ok "case1 commit-landed: HEAD subject correct"
else
  bad "case1 commit-landed: subject=$(git -C "$REPO" log -1 --pretty=%s)"
fi
if [ "$LOCAL_HEAD" = "$BARE_MAIN" ]; then
  ok "case1 ff-push-to-local-bare: bare remote main == local HEAD ($BARE_MAIN)"
else
  bad "case1 ff-push: local=$LOCAL_HEAD bare=$BARE_MAIN"
fi

# ----------------------------------------------------------------------------
# Case 2: --force is rejected (§11.4.113), exit 2, nothing committed
# ----------------------------------------------------------------------------
echo "x" > "$REPO/b.txt"
run_wrapper -m "msg" --force b.txt
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q '11.4.113'; then
  ok "case2 force-rejected: exit 2 citing §11.4.113"
else
  bad "case2 force-rejected: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
# Case 3: -A / add-all is refused (§11.4.30), exit 2
# ----------------------------------------------------------------------------
run_wrapper -m "msg" -A
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q '11.4.30'; then
  ok "case3 add-A-refused: exit 2 citing §11.4.30"
else
  bad "case3 add-A-refused: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
# Case 4: no pathspec is refused (§11.4.30), exit 2
# ----------------------------------------------------------------------------
run_wrapper -m "msg"
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q 'no pathspec'; then
  ok "case4 no-pathspec-refused: exit 2"
else
  bad "case4 no-pathspec-refused: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
# Case 5: helix_qa pathspec is refused (§11.4.30 gotcha), exit 2
# ----------------------------------------------------------------------------
run_wrapper -m "msg" helix_qa
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q 'helix_qa'; then
  ok "case5 helix_qa-refused: exit 2"
else
  bad "case5 helix_qa-refused: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
# Case 6: +refspec forced push is refused (§11.4.113), exit 2
# ----------------------------------------------------------------------------
run_wrapper -m "msg" "+main:main"
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q '11.4.113'; then
  ok "case6 plus-refspec-refused: exit 2 citing §11.4.113"
else
  bad "case6 plus-refspec-refused: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
# Case 7: mutation residue aborts (§11.4.84), exit 2
# ----------------------------------------------------------------------------
printf 'func verify() { return true } // always pass\n' > "$REPO/mutated.go"
run_wrapper -m "msg" mutated.go
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q '11.4.84'; then
  ok "case7 mutation-residue-abort: exit 2 citing §11.4.84"
else
  bad "case7 mutation-residue-abort: rc=$RC out=$OUT"
fi
rm -f "$REPO/mutated.go"

# ----------------------------------------------------------------------------
# Case 8: fetch-before-push attempted + non-FF STOP (§11.4.71/§11.4.113)
#   Advance the bare remote behind the wrapper's back so local is non-FF.
# ----------------------------------------------------------------------------
OTHER="$WORK/other"
git clone --quiet "$BARE" "$OTHER"
git -C "$OTHER" config user.email "o@example.invalid"
git -C "$OTHER" config user.name  "o"
git -C "$OTHER" config commit.gpgsign false
echo "divergent" > "$OTHER/c.txt"
git -C "$OTHER" add c.txt
git -C "$OTHER" commit --quiet -m "divergent upstream"
git -C "$OTHER" push --quiet origin main
# Now local REPO main is behind bare main. A new local commit => non-FF.
echo "local-only" > "$REPO/d.txt"
run_wrapper -m "feat: local d.txt (should hit non-FF)" d.txt
if [ "$RC" -eq 2 ] && printf '%s' "$OUT" | grep -q 'non-fast-forward'; then
  ok "case8 fetch-before-push + non-FF STOP: exit 2, no force"
else
  bad "case8 non-FF STOP: rc=$RC out=$OUT"
fi
# Confirm fetch was actually attempted (commit is durable locally; bare unchanged by us)
if printf '%s' "$OUT" | grep -q 'fetch-before-push'; then
  ok "case8 fetch-before-push attempted (log present)"
else
  bad "case8 fetch-before-push log missing: $OUT"
fi
# Confirm the local commit DID land (commit precedes push; durability per §11.4.88)
if [ "$(git -C "$REPO" log -1 --pretty=%s)" = "feat: local d.txt (should hit non-FF)" ]; then
  ok "case8 commit-durable-before-push: local commit landed despite non-FF push STOP"
else
  bad "case8 commit-durable: subject=$(git -C "$REPO" log -1 --pretty=%s)"
fi

# ----------------------------------------------------------------------------
# Case 9: --no-push commits but does not push (staging-only path), exit 0
# ----------------------------------------------------------------------------
NP="$WORK/nopush"; NPBARE="$WORK/nopush.git"
git init --quiet --bare "$NPBARE"
git init --quiet "$NP"
git -C "$NP" config user.email s@x.invalid; git -C "$NP" config user.name s; git -C "$NP" config commit.gpgsign false
git -C "$NP" remote add origin "$NPBARE"
echo seed > "$NP/seed.txt"; git -C "$NP" add seed.txt; git -C "$NP" commit --quiet -m seed; git -C "$NP" branch -M main
echo more > "$NP/e.txt"
set +e
OUT="$(COMMIT_ALL_REPO_ROOT="$NP" COMMIT_ALL_REMOTE="origin" "$WRAPPER" -m "feat: e.txt no push" --no-push e.txt 2>&1)"; RC=$?
set -e
if [ "$RC" -eq 0 ] && printf '%s' "$OUT" | grep -q 'push=skipped' \
   && [ "$(git -C "$NP" log -1 --pretty=%s)" = "feat: e.txt no push" ] \
   && ! git -C "$NPBARE" rev-parse --verify --quiet refs/heads/main >/dev/null 2>&1; then
  ok "case9 no-push: commit landed, bare untouched"
else
  bad "case9 no-push: rc=$RC out=$OUT"
fi

# ----------------------------------------------------------------------------
echo "== SELFTEST SUMMARY: PASS=$PASS FAIL=$FAIL =="
[ "$FAIL" -eq 0 ] || exit 1
exit 0
