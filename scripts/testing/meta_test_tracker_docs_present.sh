#!/usr/bin/env bash
#
# ============================================================================
# meta_test_tracker_docs_present.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-TRACKER-DOCS-PRESENT (in scripts/pre_build_verification.sh) is NOT
#   a bluff gate (§1.1). In a throwaway git repo seeded with the four canonical
#   tracker docs — each carrying a §11.4.44 revision header — assert the gate
#   PASSes, then introduce real violations one at a time (delete a doc; strip a
#   '**Revision:**' header line; strip a '**Last modified:**' header line) and
#   assert the gate FLIPS to FAIL, restoring to PASS after each.
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, pointed at via PBV_REPO_ROOT. The
#   tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_tracker_docs_present.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, git, grep, head. Cross-ref: scripts/pre_build_verification.sh
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

# A well-formed tracker doc: H1 + §11.4.44 revision header block.
seed_doc() { # <relpath> <h1>
  printf '# %s\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n**Authority:** test\n\n---\n\n### §1. [ATM-001] seed item\n' \
    "$2" > "$TMP/$1"
}
seed_all() {
  seed_doc docs/Issues.md "Issues"
  seed_doc docs/Fixed.md "Fixed"
  seed_doc docs/Issues_Summary.md "Issues Summary"
  seed_doc docs/Fixed_Summary.md "Fixed Summary"
}
seed_all
git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" bash "$GATE" --gate CM-TRACKER-DOCS-PRESENT >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

echo "=== META-TEST: CM-TRACKER-DOCS-PRESENT (paired §1.1 mutation) ==="
echo "baseline — all 4 docs present with revision headers, must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — delete docs/Fixed_Summary.md (a required tracker doc missing)"
rm -f "$TMP/docs/Fixed_Summary.md"
expect 1 "Mut1 mutated, gate must FAIL"
seed_doc docs/Fixed_Summary.md "Fixed Summary"
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — strip the '**Revision:**' header line from docs/Issues.md"
printf '# Issues\n\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n' > "$TMP/docs/Issues.md"
expect 1 "Mut2 mutated (no Revision), gate must FAIL"
seed_doc docs/Issues.md "Issues"
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Mut3 — strip the '**Last modified:**' header line from docs/Fixed.md"
printf '# Fixed\n\n**Revision:** 1\n\n---\n' > "$TMP/docs/Fixed.md"
expect 1 "Mut3 mutated (no Last modified), gate must FAIL"
seed_doc docs/Fixed.md "Fixed"
expect 0 "Mut3 restored, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-TRACKER-DOCS-PRESENT catches a missing doc AND a missing revision/last-modified header without false-FAILing the well-formed constellation"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
