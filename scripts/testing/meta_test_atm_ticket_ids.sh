#!/usr/bin/env bash
#
# ============================================================================
# meta_test_atm_ticket_ids.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-ATM-TICKET-IDS (in scripts/pre_build_verification.sh) is NOT a
#   bluff gate (§1.1 / §11.4.54). In a throwaway git repo seeded with
#   docs/Issues.md + docs/Fixed.md whose workable-item headings carry a
#   contiguous, unique [ATM-001..ATM-NNN] sequence, assert the gate PASSes,
#   then introduce real §11.4.54 violations one at a time and assert the gate
#   FLIPS to FAIL, restoring to PASS after each:
#     Mut1 — drop an [ATM-NNN] token from a workable-item heading
#     Mut2 — duplicate an id across the two files
#     Mut3 — gap the sequence (skip a number)
#   Plus a negative case: a non-item heading (the H1) without an id must NOT
#   false-FAIL (§11.4.1).
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, pointed at via PBV_REPO_ROOT. The
#   tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_atm_ticket_ids.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, git, grep, sort, uniq. Cross-ref: scripts/pre_build_verification.sh
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

# Seed: Fixed holds ATM-001..003, Issues holds ATM-004..005 (contiguous 1..5).
seed_fixed() {
  printf '# Fixed\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n\n### §1. [ATM-001] first closed item\n### §2. [ATM-002] second closed item\n### §3. [ATM-003] third closed item\n' \
    > "$TMP/docs/Fixed.md"
}
seed_issues() {
  printf '# Issues\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n\n### §1. [ATM-004] first open item\n### §2. [ATM-005] second open item\n' \
    > "$TMP/docs/Issues.md"
}
seed_all() { seed_fixed; seed_issues; }
seed_all
git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" bash "$GATE" --gate CM-ATM-TICKET-IDS >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

echo "=== META-TEST: CM-ATM-TICKET-IDS (paired §1.1 mutation) ==="
echo "baseline — contiguous unique ATM-001..005 across both files, must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — strip the [ATM-005] token from an Issues workable-item heading"
printf '# Issues\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n\n### §1. [ATM-004] first open item\n### §2. second open item (NO TOKEN)\n' \
  > "$TMP/docs/Issues.md"
expect 1 "Mut1 mutated (heading without [ATM-NNN]), gate must FAIL"
seed_issues
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — duplicate id: Issues §2 re-uses [ATM-004]"
printf '# Issues\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n\n### §1. [ATM-004] first open item\n### §2. [ATM-004] duplicate id item\n' \
  > "$TMP/docs/Issues.md"
expect 1 "Mut2 mutated (duplicate ATM-004), gate must FAIL"
seed_issues
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Mut3 — gap the sequence: Issues uses [ATM-006] [ATM-007] (skips 004/005)"
printf '# Issues\n\n**Revision:** 1\n**Last modified:** 2026-06-14T00:00:00Z\n\n---\n\n### §1. [ATM-006] gapped item a\n### §2. [ATM-007] gapped item b\n' \
  > "$TMP/docs/Issues.md"
expect 1 "Mut3 mutated (gap at 004/005), gate must FAIL"
seed_issues
expect 0 "Mut3 restored, gate must PASS"
echo

echo "Neg — the doc H1 ('# Issues') carries no id but is NOT a §X. item; must NOT false-FAIL (§11.4.1)"
# Baseline already exercises this (H1 present, no id) — assert PASS holds.
expect 0 "Neg H1-without-id, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-ATM-TICKET-IDS catches missing tokens, duplicate ids, and sequence gaps without false-FAILing a non-item heading"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
