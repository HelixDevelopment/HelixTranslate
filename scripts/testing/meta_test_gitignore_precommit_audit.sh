#!/usr/bin/env bash
#
# ============================================================================
# meta_test_gitignore_precommit_audit.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-GITIGNORE-PRECOMMIT-AUDIT (in scripts/pre_build_verification.sh)
#   is NOT a bluff gate (§1.1 / §11.4.30). For EVERY forbidden class the gate
#   guards, introduce a real violation in a throwaway git repo and assert the
#   gate FLIPS to FAIL, then remove it and assert it returns to PASS.
#
#   The real project working tree is NEVER mutated: every mutation happens in
#   a disposable git repo under a mktemp dir, and the gate is pointed at it via
#   PBV_REPO_ROOT. The tmp dir is removed on every exit path (trap).
#
# Usage:   bash scripts/testing/meta_test_gitignore_precommit_audit.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff on >=1 class; 2 = harness error.
#
# Dependencies: bash, git, grep. Cross-ref: scripts/pre_build_verification.sh
# Parseability (§11.4.67): passes bash -n AND sh -n (POSIX-portable).
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

# Build a minimal, COMPLIANT throwaway repo: a couple of legit tracked files
# plus the project's anchoring .gitignore + the allow-listed .env.example.
git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/cmd/api-server" "$TMP/build"
printf 'package main\nfunc main(){}\n' > "$TMP/cmd/api-server/main.go"
printf 'OPENAI_API_KEY=replace-me\n'    > "$TMP/.env.example"
printf '/api-server\nbuild/api-server\n.env\n.env.*\n!.env.example\n' > "$TMP/.gitignore"
git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" sh "$GATE" --gate CM-GITIGNORE-PRECOMMIT-AUDIT >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

# add_track <relpath> <content> ; remove_track <relpath>
add_track()    { mkdir -p "$TMP/$(dirname "$1")"; printf '%s' "$2" > "$TMP/$1"; git -C "$TMP" add -f "$1" >/dev/null 2>&1; }
remove_track() { git -C "$TMP" rm -q --cached "$1" >/dev/null 2>&1; rm -f "$TMP/$1"; }

echo "=== META-TEST: CM-GITIGNORE-PRECOMMIT-AUDIT (paired §1.1 mutation) ==="
echo "baseline — compliant throwaway repo must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — track build/api-server (declared build output)"
add_track "build/api-server" "ELF-binary-bytes"
expect 1 "Mut1 mutated, gate must FAIL"
remove_track "build/api-server"
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — track root prebuilt binary 'api-server' (anchoring regression)"
add_track "api-server" "ELF-binary-bytes"
expect 1 "Mut2 mutated, gate must FAIL"
remove_track "api-server"
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Mut3 — track a real .env secret file (NOT .env.example)"
add_track ".env" "OPENAI_API_KEY=sk-REALSECRET"
expect 1 "Mut3 mutated, gate must FAIL"
remove_track ".env"
expect 0 "Mut3 restored, gate must PASS"
echo

echo "Mut4 — track api_keys.json (secret-bearing config)"
add_track "api_keys.json" '{"openai":"sk-x"}'
expect 1 "Mut4 mutated, gate must FAIL"
remove_track "api_keys.json"
expect 0 "Mut4 restored, gate must PASS"
echo

echo "Neg — confirm allow-listed .env.example stays PASS (no false positive)"
expect 0 "Neg .env.example tracked, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-GITIGNORE-PRECOMMIT-AUDIT catches every forbidden class"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF on >=1 class"
exit 1
