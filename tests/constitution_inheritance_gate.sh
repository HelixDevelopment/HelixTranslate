#!/usr/bin/env bash
# tests/constitution_inheritance_gate.sh
#
# Constitution inheritance GATE (Helix Constitution §1.1 / §11.4).
#
# Verifies that the Helix Constitution submodule is present AND that
# this project actually INHERITS from it. Exits non-zero (FAIL) on any
# violated invariant, so it can guard the pre-build / pre-commit /
# pre-merge pipeline.
#
# Anchors are matched as FIXED STRINGS that include the markdown
# heading prefix ("### " / "## ") so a Table-of-Contents mention does
# NOT satisfy the gate — only the real section heading does. This is
# also what makes the paired mutation deterministic: deleting the
# heading line flips the gate to FAIL.
#
# Paired mutation proof (proves this gate is not a bluff gate):
#   scripts/testing/meta_test_constitution_inheritance.sh
#
# Exit codes: 0 = PASS (inheritance real), 1 = FAIL, 2 = harness error.

set -uo pipefail

# --- resolve repo root (callable from anywhere) -----------------------
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/.." && pwd)"
cd "$ROOT" || { echo "FAIL: cannot cd to repo root"; exit 2; }

FAILS=0
pass() { echo "  PASS  $1"; }
fail() { echo "  FAIL  $1"; FAILS=$((FAILS + 1)); }

CONST_DIR="constitution"
C_CONST="$CONST_DIR/Constitution.md"
C_CLAUDE="$CONST_DIR/CLAUDE.md"
C_AGENTS="$CONST_DIR/AGENTS.md"

# Real section-heading anchors (verified against HelixConstitution main).
ANCHOR_CONST='### §11.4 End-user quality guarantee — forensic anchor'
ANCHOR_CLAUDE='## MANDATORY ANTI-BLUFF COVENANT'
ANCHOR_AGENTS='### Anti-bluff covenant — END-USER QUALITY GUARANTEE'

echo "Constitution inheritance gate @ $ROOT"

# Invariant 1: submodule directory present.
if [ -d "$CONST_DIR" ]; then
  pass "Inv1: constitution/ exists"
else
  fail "Inv1: constitution/ missing"
fi

# Invariant 2: Constitution.md ships the §11.4 forensic anchor heading.
if [ -f "$C_CONST" ] && grep -qF "$ANCHOR_CONST" "$C_CONST"; then
  pass "Inv2: Constitution.md has '§11.4 … forensic anchor' heading"
else
  fail "Inv2: Constitution.md missing or lacks the §11.4 forensic anchor heading"
fi

# Invariant 3: constitution/CLAUDE.md ships the anti-bluff covenant heading.
if [ -f "$C_CLAUDE" ] && grep -qF "$ANCHOR_CLAUDE" "$C_CLAUDE"; then
  pass "Inv3: constitution/CLAUDE.md has the ANTI-BLUFF COVENANT heading"
else
  fail "Inv3: constitution/CLAUDE.md missing or lacks the ANTI-BLUFF COVENANT heading"
fi

# Invariant 4: constitution/AGENTS.md ships the anti-bluff covenant heading.
if [ -f "$C_AGENTS" ] && grep -qF "$ANCHOR_AGENTS" "$C_AGENTS"; then
  pass "Inv4: constitution/AGENTS.md has the Anti-bluff covenant heading"
else
  fail "Inv4: constitution/AGENTS.md missing or lacks the Anti-bluff covenant heading"
fi

# Invariant 5: parent governance files reference the submodule.
if [ -f CLAUDE.md ] && grep -qF 'constitution/CLAUDE.md' CLAUDE.md; then
  pass "Inv5a: CLAUDE.md references constitution/CLAUDE.md"
else
  fail "Inv5a: CLAUDE.md does not reference constitution/CLAUDE.md"
fi
if [ -f AGENTS.md ] && grep -qF 'constitution/AGENTS.md' AGENTS.md; then
  pass "Inv5b: AGENTS.md references constitution/AGENTS.md"
else
  fail "Inv5b: AGENTS.md does not reference constitution/AGENTS.md"
fi
if [ -f CONSTITUTION.md ] && grep -qF 'constitution/Constitution.md' CONSTITUTION.md; then
  pass "Inv5c: CONSTITUTION.md references constitution/Constitution.md"
else
  fail "Inv5c: CONSTITUTION.md does not reference constitution/Constitution.md"
fi

echo
if [ "$FAILS" -eq 0 ]; then
  echo "RESULT: PASS — constitution inheritance is real ($ROOT)"
  exit 0
fi
echo "RESULT: FAIL — $FAILS invariant(s) violated"
exit 1
