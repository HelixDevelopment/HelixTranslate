#!/usr/bin/env bash
#
# ============================================================================
# meta_test_script_target_shell_parseable.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-SCRIPT-TARGET-SHELL-PARSEABLE (in scripts/pre_build_verification.sh)
#   is NOT a bluff gate (§1.1 / §11.4.67). Introduce real shell-parse violations
#   in a throwaway git repo and assert the gate FLIPS to FAIL, then remove them
#   and assert it returns to PASS. Also assert the gate does NOT false-FAIL on a
#   legitimate honest-bash script that uses bash-only constructs (§11.4.1 — a
#   FAIL-bluff on a valid script is just as forbidden as a PASS-bluff).
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, and the gate is pointed at it via
#   PBV_REPO_ROOT. The tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_script_target_shell_parseable.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, sh, git, grep. Cross-ref: scripts/pre_build_verification.sh
#   docs/scripts/pre_build_verification.sh.md (companion guide, §11.4.18)
# Parseability (§11.4.67): honest bash shebang; passes bash -n.
# ============================================================================

set -u

SELF_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)
[ -z "$ROOT" ] && ROOT=$(CDPATH= cd -- "$SELF_DIR/../.." && pwd)
GATE="$ROOT/scripts/pre_build_verification.sh"
[ -f "$GATE" ] || { echo "harness error: gate runner not found at $GATE" >&2; exit 2; }
command -v sh   >/dev/null 2>&1 || { echo "harness error: sh not on PATH"   >&2; exit 2; }
command -v bash >/dev/null 2>&1 || { echo "harness error: bash not on PATH" >&2; exit 2; }

TMP=$(mktemp -d 2>/dev/null) || { echo "harness error: mktemp failed" >&2; exit 2; }
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT INT TERM

RC=0

# Build a minimal, COMPLIANT throwaway repo with a tracked, valid scripts/ dir.
git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/scripts/testing"
printf '#!/usr/bin/env bash\nset -u\necho hi\n' > "$TMP/scripts/good_bash.sh"
printf '#!/bin/sh\nset -u\necho hi\n'           > "$TMP/scripts/good_sh.sh"
git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" sh "$GATE" --gate CM-SCRIPT-TARGET-SHELL-PARSEABLE >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

track()   { printf '%s' "$2" > "$TMP/$1"; git -C "$TMP" add -f "$1" >/dev/null 2>&1; }
untrack() { git -C "$TMP" rm -q --cached "$1" >/dev/null 2>&1; rm -f "$TMP/$1"; }

echo "=== META-TEST: CM-SCRIPT-TARGET-SHELL-PARSEABLE (paired §1.1 mutation) ==="
echo "baseline — compliant repo (valid bash + valid sh scripts) must PASS"
expect 0 "baseline PASS"
echo

# Mut1: an sh-shebang script that uses a bash-only construct → must FAIL sh -n.
echo "Mut1 — #!/bin/sh script with bash-only process substitution (mapfile < <())"
track "scripts/bad_sh.sh" '#!/bin/sh
set -u
mapfile -t X < <(echo a)
'
expect 1 "Mut1 mutated, gate must FAIL (sh script not sh-parseable)"
untrack "scripts/bad_sh.sh"
expect 0 "Mut1 restored, gate must PASS"
echo

# Mut2: a script that is not even valid bash → must FAIL bash -n.
echo "Mut2 — script with an outright bash syntax error (unbalanced 'if')"
track "scripts/testing/broken.sh" '#!/usr/bin/env bash
if [ 1 -eq 1 ]; then
echo never-closed
'
expect 1 "Mut2 mutated, gate must FAIL (not valid bash)"
untrack "scripts/testing/broken.sh"
expect 0 "Mut2 restored, gate must PASS"
echo

# Mut3: no-shebang script with a bash-only construct → conservative sh -n → FAIL.
# Process substitution `< <(...)` is rejected by POSIX sh on every platform
# (unlike [[ ]], which some sh implementations accept), so it is a reliable
# cross-platform sh-incompatibility trigger.
echo "Mut3 — no-shebang script with bash-only process substitution (conservatively requires sh -n)"
track "scripts/noshebang.sh" 'X=a
cat < <(echo "$X")
'
expect 1 "Mut3 mutated, gate must FAIL (no shebang + sh -n fails)"
untrack "scripts/noshebang.sh"
expect 0 "Mut3 restored, gate must PASS"
echo

# Neg: an HONEST bash script using bash-only constructs MUST NOT false-FAIL
# (§11.4.1 — a FAIL-bluff on a valid script is forbidden). It is sh-INVALID by
# design but bash-VALID and bash-declared, so the gate must keep PASSing.
echo "Neg — honest #!/usr/bin/env bash script with mapfile/<() must stay PASS (no false-FAIL)"
track "scripts/honest_bash.sh" '#!/usr/bin/env bash
set -uo pipefail
mapfile -t MODULES < <(printf "a\nb\n")
echo "${MODULES[@]}"
'
expect 0 "Neg honest-bash script tracked, gate must PASS"
untrack "scripts/honest_bash.sh"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-SCRIPT-TARGET-SHELL-PARSEABLE catches real parse defects without false-FAILing honest bash"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
