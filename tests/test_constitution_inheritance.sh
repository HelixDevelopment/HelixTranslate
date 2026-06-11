#!/usr/bin/env bash
# tests/test_constitution_inheritance.sh
#
# Comprehensive host-side constitution-inheritance test.
#
# Asserts ALL invariants of the inheritance gate PLUS:
#   - the submodule is registered in .gitmodules (tracking main),
#   - every owned direct submodule carries the inheritance pointer in
#     BOTH its CLAUDE.md and AGENTS.md (recursive child inheritance).
#
# Pre-flight sanity check for the project's full-test orchestrator.
# Exit codes: 0 = PASS, 1 = FAIL, 2 = harness error.

set -uo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/.." && pwd)"
cd "$ROOT" || { echo "harness error: cannot cd to repo root"; exit 2; }

FAILS=0
pass() { echo "  PASS  $1"; }
fail() { echo "  FAIL  $1"; FAILS=$((FAILS + 1)); }

# Owned DIRECT submodules that must inherit (scope: parent + 8 direct).
DIRECT_SUBMODULES=(
  challenges containers doc_processor helix_qa
  llm_orchestrator llm_provider vision_engine llms_verifier
)
POINTER_MARKER='INHERITED FROM Helix Constitution'

echo "=== Constitution inheritance — comprehensive test @ $ROOT ==="

# 1) Core gate (Inv1-5).
echo "--- core inheritance gate ---"
if bash tests/constitution_inheritance_gate.sh; then
  pass "core gate (Inv1-5) PASS"
else
  fail "core gate (Inv1-5) FAIL"
fi
echo

# 2) Submodule registered in .gitmodules, tracking main.
echo "--- submodule registration ---"
if git config -f .gitmodules --get submodule.constitution.path >/dev/null 2>&1; then
  pass ".gitmodules registers submodule.constitution"
else
  fail ".gitmodules does NOT register submodule.constitution"
fi
if [ "$(git config -f .gitmodules --get submodule.constitution.branch 2>/dev/null)" = "main" ]; then
  pass "constitution submodule tracks branch=main"
else
  fail "constitution submodule does NOT track branch=main"
fi
echo

# 3) Recursive child inheritance pointers.
echo "--- child submodule inheritance pointers ---"
for sm in "${DIRECT_SUBMODULES[@]}"; do
  for f in CLAUDE.md AGENTS.md; do
    if [ -f "$sm/$f" ] && grep -qF "$POINTER_MARKER" "$sm/$f"; then
      pass "$sm/$f carries inheritance pointer"
    else
      fail "$sm/$f missing inheritance pointer"
    fi
  done
done
echo

if [ "$FAILS" -eq 0 ]; then
  echo "RESULT: PASS — parent + ${#DIRECT_SUBMODULES[@]} direct submodules inherit the constitution"
  exit 0
fi
echo "RESULT: FAIL — $FAILS check(s) violated"
exit 1
