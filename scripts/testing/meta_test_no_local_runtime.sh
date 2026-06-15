#!/usr/bin/env bash
#
# ============================================================================
# meta_test_no_local_runtime.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-NO-LOCAL-RUNTIME (in scripts/pre_build_verification.sh) is NOT a
#   bluff gate (§1.1 / bridge phase-2 R-5 / §11.4.69). The gate asserts the
#   default translator provisioning path sources ONLY the LLMsVerifier bridge —
#   no local-runtime (llama.cpp / Ollama) client constructed on it.
#
#   Mutations (each in a disposable git repo under mktemp, pointed at via
#   PBV_REPO_ROOT; the real project tree is NEVER mutated):
#     baseline — a bridge-only default path + the bridge prohibition literal → PASS
#     Mut1     — re-add a ProviderOllama construction on a default-path file → FAIL (Arm 1)
#     Mut2     — delete the bridge prohibition string from bridge.go        → FAIL (Arm 3)
#     Neg      — an EXPLICIT `case "llamacpp"` error arm + a comment naming
#                ollama + a worker config carrying default_provider:ollama
#                must NOT trip the gate (documented exceptions / §11.4.1).
#
# Usage:   bash scripts/testing/meta_test_no_local_runtime.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff / false-FAILs; 2 = harness error.
#
# Dependencies: bash, sh, git, grep. Cross-ref: scripts/pre_build_verification.sh
#   (gate_no_local_runtime), docs/scripts/pre_build_verification.sh.md (§11.4.18),
#   docs/design/LOCAL_RUNTIME_REMOVAL.md §6 (gate plan).
# Parseability (§11.4.67): honest bash shebang; passes bash -n AND sh -n.
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

# Build a minimal, COMPLIANT throwaway repo mirroring the default-path layout:
#   cmd/unified-translator/main.go — default arm routes through bridgeTranslator(
#   pkg/api/handler.go             — default arm routes through bridge.
#   pkg/bridge/bridge.go           — carries the no-fail-open prohibition literal
git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/cmd/unified-translator" "$TMP/pkg/api" "$TMP/pkg/bridge"

cat > "$TMP/cmd/unified-translator/main.go" <<'GO'
package main

import "x/pkg/bridge"

// default arm: source the strongest verified model via the bridge.
func bridgeTranslator(ctx int) { _ = bridge.Open }

func main() { bridgeTranslator(0) }
GO

cat > "$TMP/pkg/api/handler.go" <<'GO'
package api

import "x/pkg/bridge"

func (h *Handler) bridgeFor() { _ = bridge.Open }
GO

cat > "$TMP/pkg/bridge/bridge.go" <<'GO'
package bridge

var Open = func() {}

// On no API keys the bridge hard-errors honestly
// (local llama.cpp fallback is not permitted).
GO

git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" sh "$GATE" --gate CM-NO-LOCAL-RUNTIME >/dev/null 2>&1; echo $?; }

expect() { # <want-rc> <label>
  got=$(run_gate)
  if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi
}

write() { printf '%s' "$2" > "$TMP/$1"; git -C "$TMP" add -f "$1" >/dev/null 2>&1; }

echo "=== META-TEST: CM-NO-LOCAL-RUNTIME (paired §1.1 mutation) ==="
echo "baseline — bridge-only default path + prohibition literal, must PASS"
expect 0 "baseline PASS"
echo

echo "Mut1 — cmd/unified-translator re-adds a ProviderOllama construction on the default arm (Arm 1)"
write "cmd/unified-translator/main.go" 'package main

import "x/pkg/bridge"

func bridgeTranslator(ctx int) {
	_ = bridge.Open
	_ = ProviderOllama("ollama") // re-introduced local-runtime construction
}

func main() { bridgeTranslator(0) }
'
expect 1 "Mut1 mutated, gate must FAIL (Arm 1)"
write "cmd/unified-translator/main.go" 'package main

import "x/pkg/bridge"

func bridgeTranslator(ctx int) { _ = bridge.Open }

func main() { bridgeTranslator(0) }
'
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — pkg/bridge/bridge.go deletes the no-fail-open prohibition literal (Arm 3)"
write "pkg/bridge/bridge.go" 'package bridge

var Open = func() {}

// On no API keys the bridge hard-errors honestly.
'
expect 1 "Mut2 mutated, gate must FAIL (Arm 3)"
write "pkg/bridge/bridge.go" 'package bridge

var Open = func() {}

// On no API keys the bridge hard-errors honestly
// (local llama.cpp fallback is not permitted).
'
expect 0 "Mut2 restored, gate must PASS"
echo

echo "Neg — explicit case \"llamacpp\" error arm + ollama comment + worker config must NOT false-FAIL (§11.4.1)"
write "cmd/unified-translator/main.go" 'package main

import (
	"fmt"

	"x/pkg/bridge"
)

func bridgeTranslator(ctx int) { _ = bridge.Open }

// executeProviderTranslation routes explicit local providers to an honest error.
func executeProviderTranslation(provider string) error {
	switch provider {
	case "llamacpp":
		// llamacpp / ollama are no longer supported; honest hard error.
		return fmt.Errorf("provider=llamacpp is no longer supported (use an API provider)")
	default:
		bridgeTranslator(ctx)
		return nil
	}
}

func main() { _ = executeProviderTranslation("openai") }
'
mkdir -p "$TMP/internal/working"
write "internal/working/config.worker.json" '{
  "default_provider": "ollama"
}
'
expect 0 "Neg explicit-arm + comment + worker config present, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-NO-LOCAL-RUNTIME catches local-runtime re-introduction + prohibition deletion without false-FAILing explicit error arms / comments / worker configs"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF or false-FAILs on >=1 case"
exit 1
