#!/usr/bin/env bash
#
# ============================================================================
# meta_test_doc_sibling_sync.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-DOC-SIBLING-SYNC (in scripts/pre_build_verification.sh) is NOT a
#   bluff gate (§1.1). In a throwaway git repo seeded with an in-scope .md that
#   carries BOTH a tracked .html and a tracked .pdf sibling, each sibling fresh
#   (mtime >= the .md), assert the gate PASSes; then introduce real §11.4.65
#   violations one at a time and assert the gate FLIPS to FAIL, restoring to
#   PASS after each:
#     Mut1 — delete (untrack) the .html sibling          (missing export)
#     Mut2 — delete (untrack) the .pdf sibling           (missing export)
#     Mut3 — backdate a sibling so the .md is NEWER       (stale export)
#     Mut4 — an EXCLUDED-tree .md without siblings        (must NOT FAIL)
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, pointed at via PBV_REPO_ROOT. The
#   tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_doc_sibling_sync.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, git, grep, touch. Cross-ref: scripts/pre_build_verification.sh
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
mkdir -p "$TMP/docs"

# Seed a fresh .md + .html + .pdf triple at a given relpath base (no extension).
# Siblings are touched AFTER the .md so they are not older (fresh).
seed_triple() { # <relpath-base-without-ext> <title>
  printf '# %s\n\nbody\n' "$2" > "$TMP/$1.md"
  printf '<html><body>%s</body></html>\n' "$2" > "$TMP/$1.html"
  printf '%%PDF-1.4 fake\n' > "$TMP/$1.pdf"
  # Ensure siblings are >= .md (touch them one second forward).
  touch "$TMP/$1.html" "$TMP/$1.pdf"
}

commit_all() { git -C "$TMP" add -A >/dev/null 2>&1; git -C "$TMP" commit -qm x >/dev/null 2>&1; }

# Baseline: one in-scope doc with fresh tracked siblings.
seed_triple docs/Sample "Sample"
commit_all

run_gate() { PBV_REPO_ROOT="$TMP" bash "$GATE" --gate CM-DOC-SIBLING-SYNC >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

echo "=== META-TEST: CM-DOC-SIBLING-SYNC (paired §1.1 mutation) ==="
echo "baseline — in-scope .md with fresh tracked .html + .pdf, must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — remove the tracked .html sibling (missing export)"
git -C "$TMP" rm -q docs/Sample.html >/dev/null 2>&1
expect 1 "Mut1 mutated (no .html), gate must FAIL"
seed_triple docs/Sample "Sample"; commit_all
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — remove the tracked .pdf sibling (missing export)"
git -C "$TMP" rm -q docs/Sample.pdf >/dev/null 2>&1
expect 1 "Mut2 mutated (no .pdf), gate must FAIL"
seed_triple docs/Sample "Sample"; commit_all
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Mut3 — backdate the .html sibling so the .md is NEWER (stale export)"
# Make .html clearly older than .md (both tracked).
touch -t 200001010000 "$TMP/docs/Sample.html"
touch "$TMP/docs/Sample.md"
expect 1 "Mut3 mutated (stale .html), gate must FAIL"
touch "$TMP/docs/Sample.html" "$TMP/docs/Sample.pdf"   # refresh siblings >= .md
expect 0 "Mut3 restored, gate must PASS"
echo

echo "Mut4 — an EXCLUDED owned-submodule .md WITHOUT siblings must NOT FAIL"
mkdir -p "$TMP/helix_qa/docs"
printf '# Excluded\n\nbody\n' > "$TMP/helix_qa/docs/Thing.md"
commit_all
expect 0 "Mut4 excluded-tree .md without siblings, gate must still PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-DOC-SIBLING-SYNC catches a missing .html, a missing .pdf, AND a stale sibling, without false-FAILing on excluded owned-submodule docs"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
