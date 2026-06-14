#!/usr/bin/env bash
#
# ============================================================================
# meta_test_no_force_push_absolute.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-NO-FORCE-PUSH-ABSOLUTE (in scripts/pre_build_verification.sh) is
#   NOT a bluff gate (§1.1). In a throwaway git repo seeded with a clean script
#   under scripts/ that does a normal `git push` (no force), assert the gate
#   PASSes; then inject real §11.4.113 force-push invocations one at a time and
#   assert the gate FLIPS to FAIL, restoring to PASS after each:
#     Mut1 — `git push --force`                          (forbidden flag)
#     Mut2 — `git push --force-with-lease`               (forbidden flag)
#     Mut3 — `git push origin +main:main`                (forced refspec)
#     Mut4 — a COMMENT + a die-string naming --force      (GUARD, must NOT FAIL)
#
#   The Mut4 negative case is the load-bearing anti-false-positive proof: the
#   project's own commit_all.sh §11.4.113 GUARD names these tokens in comments,
#   case-arms, and die/echo refusal strings — none is an invocation, and the
#   gate MUST NOT trip on them (a §11.4.1 FAIL-bluff would).
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, pointed at via PBV_REPO_ROOT. The
#   tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_no_force_push_absolute.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, git. Cross-ref: scripts/pre_build_verification.sh
#   docs/scripts/pre_build_verification.sh.md (companion guide, §11.4.18).
# Parseability (§11.4.67): honest bash shebang; passes bash -n.
# ============================================================================

set -u

SELF_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)
[ -z "$ROOT" ] && ROOT=$(CDPATH= cd -- "$SELF_DIR/../.." && pwd)
GATE="$ROOT/scripts/pre_build_verification.sh"
[ -f "$GATE" ] || { echo "harness error: gate runner not found at $GATE" >&2; exit 2; }

TMP=$(mktemp -d 2>/dev/null) || { echo "harness error: mktemp failed" >&2; exit 2; }
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT INT TERM

RC=0

git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/scripts"

# A clean script: normal push, no force.
seed_clean() {
  cat > "$TMP/scripts/deploy.sh" <<'EOS'
#!/usr/bin/env bash
set -eu
git push origin main
EOS
}

commit_all() { git -C "$TMP" add -A >/dev/null 2>&1; git -C "$TMP" commit -qm x >/dev/null 2>&1; }

seed_clean
commit_all

run_gate() { PBV_REPO_ROOT="$TMP" bash "$GATE" --gate CM-NO-FORCE-PUSH-ABSOLUTE >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

echo "=== META-TEST: CM-NO-FORCE-PUSH-ABSOLUTE (paired §1.1 mutation) ==="
echo "baseline — clean 'git push origin main', must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — inject 'git push --force'"
cat > "$TMP/scripts/deploy.sh" <<'EOS'
#!/usr/bin/env bash
git push --force origin main
EOS
commit_all
expect 1 "Mut1 mutated (--force), gate must FAIL"
seed_clean; commit_all
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — inject 'git push --force-with-lease'"
cat > "$TMP/scripts/deploy.sh" <<'EOS'
#!/usr/bin/env bash
git push --force-with-lease origin main
EOS
commit_all
expect 1 "Mut2 mutated (--force-with-lease), gate must FAIL"
seed_clean; commit_all
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Mut3 — inject a forced refspec 'git push origin +main:main'"
cat > "$TMP/scripts/deploy.sh" <<'EOS'
#!/usr/bin/env bash
git push origin +main:main
EOS
commit_all
expect 1 "Mut3 mutated (+refspec), gate must FAIL"
seed_clean; commit_all
expect 0 "Mut3 restored, gate must PASS"
echo

echo "Mut4 — a GUARD script naming --force in a comment + die string (must NOT FAIL)"
cat > "$TMP/scripts/guard.sh" <<'EOS'
#!/usr/bin/env bash
# §11.4.113 ABSOLUTE no-force-push — this script REFUSES git push --force.
for arg in "$@"; do
  case "$arg" in
    --force|--force-with-lease|-f)
      die "refusing forbidden git push --force flag '$arg'" ;;
  esac
done
git push origin main
EOS
commit_all
expect 0 "Mut4 GUARD (comment+case-arm+die string), gate must still PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-NO-FORCE-PUSH-ABSOLUTE catches --force, --force-with-lease, AND a +refspec push, without false-FAILing the commit_all.sh-class force-push GUARD"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
