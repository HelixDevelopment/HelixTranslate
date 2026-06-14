#!/usr/bin/env bash
#
# ============================================================================
# meta_test_no_fakes_beyond_unit.sh — PAIRED §1.1 MUTATION PROOF
# ============================================================================
#
# Purpose:
#   Prove CM-NO-FAKES-BEYOND-UNIT (in scripts/pre_build_verification.sh) is
#   NOT a bluff gate (§1.1 / §11.4.27). In a throwaway git repo:
#     - a UNIT test importing a mock path stays PASS (mocks allowed in units),
#     - a NON-UNIT (build-tagged integration/e2e) test importing a mock path
#       flips the gate to FAIL,
#     - removing the fake import returns the gate to PASS.
#
#   The real project working tree is NEVER mutated: every mutation happens in
#   a disposable git repo under a mktemp dir, gate pointed via PBV_REPO_ROOT.
#
# Usage:   bash scripts/testing/meta_test_no_fakes_beyond_unit.sh
# Exit:    0 = gate proven sound; 1 = gate is a bluff; 2 = harness error.
#
# Dependencies: bash, git, grep, head. Cross-ref: scripts/pre_build_verification.sh
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

git -C "$TMP" init -q
git -C "$TMP" config user.email t@t.t
git -C "$TMP" config user.name t
mkdir -p "$TMP/pkg/svc"

# A clean UNIT test (no build tag) that DOES import a mock — this is ALLOWED
# and must NOT trip the gate (proves no false positive on units).
cat > "$TMP/pkg/svc/svc_unit_test.go" <<'EOF'
package svc

import (
	"testing"

	"example.com/proj/pkg/svc/mocks"
)

func TestUnit(t *testing.T) { _ = mocks.New() }
EOF

# A clean NON-UNIT integration test that does NOT import a fake.
cat > "$TMP/pkg/svc/svc_integration_test.go" <<'EOF'
//go:build integration

package svc

import "testing"

func TestIntegration(t *testing.T) {}
EOF

git -C "$TMP" add -A >/dev/null 2>&1
git -C "$TMP" commit -qm seed >/dev/null 2>&1

run_gate() { PBV_REPO_ROOT="$TMP" sh "$GATE" --gate CM-NO-FAKES-BEYOND-UNIT >/dev/null 2>&1; echo $?; }
expect() { got=$(run_gate); if [ "$got" = "$1" ]; then echo "  OK   $2 (gate rc=$got)"; else echo "  XX   $2 — wanted rc=$1 got rc=$got"; RC=1; fi; }

echo "=== META-TEST: CM-NO-FAKES-BEYOND-UNIT (paired §1.1 mutation) ==="
echo "baseline — unit-test-with-mock + clean integration test must PASS"
expect 0 "baseline PASS (mock in UNIT test is allowed)"
echo

echo "Mut1 — make the integration (non-unit) test import a mock path"
cat > "$TMP/pkg/svc/svc_integration_test.go" <<'EOF'
//go:build integration

package svc

import (
	"testing"

	"example.com/proj/pkg/svc/mocks"
)

func TestIntegration(t *testing.T) { _ = mocks.New() }
EOF
git -C "$TMP" add -A >/dev/null 2>&1
expect 1 "Mut1 mutated, gate must FAIL"

echo "Mut1 restore — drop the fake import from the integration test"
cat > "$TMP/pkg/svc/svc_integration_test.go" <<'EOF'
//go:build integration

package svc

import "testing"

func TestIntegration(t *testing.T) {}
EOF
git -C "$TMP" add -A >/dev/null 2>&1
expect 0 "Mut1 restored, gate must PASS"
echo

echo "Mut2 — e2e-tagged test importing a stub path must FAIL"
cat > "$TMP/pkg/svc/svc_e2e_test.go" <<'EOF'
//go:build e2e

package svc

import (
	"testing"

	"example.com/proj/internal/stubs"
)

func TestE2E(t *testing.T) { _ = stubs.New() }
EOF
git -C "$TMP" add -A >/dev/null 2>&1
expect 1 "Mut2 mutated, gate must FAIL"
git -C "$TMP" rm -q --cached pkg/svc/svc_e2e_test.go >/dev/null 2>&1
rm -f "$TMP/pkg/svc/svc_e2e_test.go"
expect 0 "Mut2 removed, gate must PASS"
echo

if [ "$RC" -eq 0 ]; then
  echo "META-TEST RESULT: PASS — CM-NO-FAKES-BEYOND-UNIT catches non-unit fakes, allows unit mocks"
  exit 0
fi
echo "META-TEST RESULT: FAIL — gate is a BLUFF"
exit 1
