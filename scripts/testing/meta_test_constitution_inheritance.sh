#!/usr/bin/env bash
# scripts/testing/meta_test_constitution_inheritance.sh
#
# PAIRED MUTATION PROOF (Helix Constitution §1.1 — false-positive
# immunity) for tests/constitution_inheritance_gate.sh.
#
# A gate without a paired mutation is itself a Constitution violation
# (§35.7). This harness PROVES the inheritance gate is not a bluff
# gate: for EVERY invariant, breaking it MUST flip the gate to FAIL,
# and a clean tree MUST PASS.
#
# Mutations are surgical (delete one heading/reference line), applied
# to a backup-protected copy, and ALWAYS restored via an EXIT trap —
# even on Ctrl-C — so the working tree is never left mutated.
#
# Exit codes: 0 = PASS (gate proven sound), 1 = FAIL (gate is a bluff
# on >=1 invariant), 2 = harness error.

set -uo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"
cd "$ROOT" || { echo "harness error: cannot cd to repo root"; exit 2; }

GATE='bash tests/constitution_inheritance_gate.sh'
RC=0

# Files we may transiently mutate — register a global restore trap so
# nothing is ever left broken.
declare -a _BAKS=()
restore_all() {
  local pair f b
  for pair in "${_BAKS[@]:-}"; do
    [ -z "$pair" ] && continue
    f="${pair%%::*}"; b="${pair##*::}"
    [ -f "$b" ] && cp "$b" "$f" && rm -f "$b"
  done
}
trap restore_all EXIT INT TERM

run_gate_rc() { eval "$GATE" >/dev/null 2>&1; echo $?; }

expect_rc() { # <want> <label>
  local want="$1" label="$2" got
  got="$(run_gate_rc)"
  if [ "$got" = "$want" ]; then
    echo "  OK   $label (gate rc=$got)"
  else
    echo "  XX   $label — wanted gate rc=$want, got rc=$got"
    RC=1
  fi
}

# mutate <file> <fixed-heading-substring> <label>
#   removes every line containing the (heading) substring, asserts the
#   gate FAILs, then restores and asserts the gate PASSes again.
mutate() {
  local file="$1" sentinel="$2" label="$3" bak
  if [ ! -f "$file" ]; then echo "  XX   $label — $file missing"; RC=1; return; fi
  if ! grep -qF "$sentinel" "$file"; then
    echo "  XX   $label — sentinel absent pre-mutation (gate/anchor drift)"; RC=1; return
  fi
  bak="$(mktemp)"; cp "$file" "$bak"; _BAKS+=("$file::$bak")
  grep -vF "$sentinel" "$bak" > "$file"
  expect_rc 1 "$label — mutated, gate must FAIL"
  cp "$bak" "$file"; rm -f "$bak"
  # drop this entry from the restore list (already restored)
  local i; for i in "${!_BAKS[@]}"; do [ "${_BAKS[$i]}" = "$file::$bak" ] && unset '_BAKS[$i]'; done
  expect_rc 0 "$label — restored, gate must PASS"
}

echo "=== META-TEST: constitution inheritance gate (CM-CONSTITUTION-INHERITANCE) ==="
echo "baseline — clean tree must PASS"
expect_rc 0 "baseline PASS"
echo

echo "Inv2 — strip §11.4 forensic anchor heading from Constitution.md"
mutate "constitution/Constitution.md" '### §11.4 End-user quality guarantee — forensic anchor' "Inv2 anchor"
echo
echo "Inv3 — strip ANTI-BLUFF COVENANT heading from constitution/CLAUDE.md"
mutate "constitution/CLAUDE.md" '## MANDATORY ANTI-BLUFF COVENANT' "Inv3 covenant"
echo
echo "Inv4 — strip Anti-bluff covenant heading from constitution/AGENTS.md"
mutate "constitution/AGENTS.md" '### Anti-bluff covenant — END-USER QUALITY GUARANTEE' "Inv4 covenant"
echo
echo "Inv5a — strip parent CLAUDE.md submodule reference"
mutate "CLAUDE.md" 'constitution/CLAUDE.md' "Inv5a parent ref"
echo

echo "Canonical cross-check — constitution/meta_test_inheritance.sh with our gate"
if bash constitution/meta_test_inheritance.sh "$GATE" >/tmp/const_meta_xcheck.out 2>&1; then
  echo "  OK   shipped meta_test_inheritance.sh agrees (gate caught the §11.4 mutation)"
  sed 's/^/       /' /tmp/const_meta_xcheck.out | tail -2
else
  echo "  XX   shipped meta_test_inheritance.sh disagrees — gate did NOT catch the mutation"
  sed 's/^/       /' /tmp/const_meta_xcheck.out | tail -8
  RC=1
fi
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — gate proven to catch every inheritance regression"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF on at least one invariant"
exit 1
