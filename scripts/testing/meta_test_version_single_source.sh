#!/usr/bin/env bash
#
# ============================================================================
# meta_test_version_single_source.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-VERSION-SINGLE-SOURCE (in scripts/pre_build_verification.sh) is NOT
#   a bluff gate (§1.1 / P0.1 / a36030e). Introduce a real hardcoded version
#   literal in a cmd/*/main.go inside a throwaway git repo and assert the gate
#   FLIPS to FAIL, then restore the version.AppVersion reference and assert it
#   returns to PASS. Also assert the gate does NOT false-FAIL on the COMPLIANT
#   `appVersion = version.AppVersion` form, nor on a 2-part XML/EPUB attr
#   (§11.4.1 — false-FAIL forbidden).
#
#   The real project working tree is NEVER mutated: every mutation happens in a
#   disposable git repo under a mktemp dir, pointed at via PBV_REPO_ROOT. The
#   tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_version_single_source.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, sh, git, grep. Cross-ref: scripts/pre_build_verification.sh
#   docs/scripts/pre_build_verification.sh.md (companion guide, §11.4.18),
#   pkg/version/app_test.go (TestNoBinaryDeclaresDivergentVersionLiteral).
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

# Build a minimal, COMPLIANT throwaway repo with a cmd/ tree whose binaries all
# source their version from version.AppVersion (no hardcoded literals).
git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/cmd/server" "$TMP/cmd/cli"
printf 'package main\n\nimport version "x/pkg/version"\n\nconst appVersion = version.AppVersion\n\nfunc main() { _ = appVersion }\n' \
  > "$TMP/cmd/server/main.go"
printf 'package main\n\nimport version "x/pkg/version"\n\nconst version2 = version.AppVersion\n\nfunc main() { _ = version2 }\n' \
  > "$TMP/cmd/cli/main.go"
git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" sh "$GATE" --gate CM-VERSION-SINGLE-SOURCE >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

write() { printf '%s' "$2" > "$TMP/$1"; git -C "$TMP" add -f "$1" >/dev/null 2>&1; }

echo "=== META-TEST: CM-VERSION-SINGLE-SOURCE (paired §1.1 mutation) ==="
echo "baseline — all binaries reference version.AppVersion, must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — cmd/server hardcodes appVersion = \"3.0.0\" (divergent literal)"
write "cmd/server/main.go" 'package main

const appVersion = "3.0.0"

func main() { _ = appVersion }
'
expect 1 "Mut1 mutated, gate must FAIL"
write "cmd/server/main.go" 'package main

import version "x/pkg/version"

const appVersion = version.AppVersion

func main() { _ = appVersion }
'
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — cmd/cli hardcodes const version = \"2.0.0\""
write "cmd/cli/main.go" 'package main

const version = "2.0.0"

func main() { _ = version }
'
expect 1 "Mut2 mutated, gate must FAIL"
write "cmd/cli/main.go" 'package main

import version "x/pkg/version"

const version2 = version.AppVersion

func main() { _ = version2 }
'
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Neg — a 2-part XML/EPUB-style attr version=\"1.0\" must NOT false-FAIL (§11.4.1)"
write "cmd/server/main.go" 'package main

import version "x/pkg/version"

const appVersion = version.AppVersion

// epubHeader carries a 2-part container attr, NOT a Go version literal.
const epubHeader = `<?xml version="1.0"?>`

func main() { _ = appVersion; _ = epubHeader }
'
expect 0 "Neg 2-part attr present, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-VERSION-SINGLE-SOURCE catches hardcoded version literals without false-FAILing the compliant form"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
