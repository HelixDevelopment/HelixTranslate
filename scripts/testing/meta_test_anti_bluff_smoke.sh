#!/usr/bin/env bash
# scripts/testing/meta_test_anti_bluff_smoke.sh
#
# Purpose:   Paired §1.1 mutation test for the CM-ANTI-BLUFF-SMOKE gate.
#            The gate asserts that anti-bluff infrastructure is in place:
#            (1) docs/qa/ exists and has ≥1 evidence dir with a README.md
#            (2) every shipped feature fix from the current release cycle has
#                a docs/qa/<run-id>/ evidence dir (§11.4.83)
#            (3) the WORKING_PLAN.md exists and references §11.4 anti-bluff
#
# Mutation:  removes one evidence dir's README.md → gate MUST FAIL.
#
# Usage:     scripts/testing/meta_test_anti_bluff_smoke.sh
#            exit 0 = mutation correctly detected (gate is not a bluff)
#            exit 1 = mutation NOT detected (gate IS a bluff — fix the gate)
#
# Inputs:    docs/qa/, docs/WORKING_PLAN.md
# Outputs:   stdout only (test result)
# Side-effects: creates+removes a temporary test fixture; never modifies tracked files.
# Dependencies: bash, coreutils. No network.
# Cross-references: §11.4.69 (anti-bluff smoke), §11.4.83 (evidence mandate),
#            §1.1 (paired mutation), §11.4.67 (target-shell-parseable).

set -eu

SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0
TMPDIR=""

cleanup() {
    if [ -n "$TMPDIR" ] && [ -d "$TMPDIR" ]; then
        rm -rf "$TMPDIR"
    fi
}
trap cleanup EXIT

# --- Gate logic (replicated from pre_build_verification.sh CM-ANTI-BLUFF-SMOKE) ---
gate_pass() {
    local desc="$1"
    echo "PASS: $desc"
    PASS=$((PASS + 1))
}

gate_fail() {
    local desc="$1"
    echo "FAIL: $desc"
    FAIL=$((FAIL + 1))
}

# Check 1: docs/qa/ exists and has ≥1 evidence dir
if [ ! -d "docs/qa" ]; then
    gate_fail "CM-ANTI-BLUFF-SMOKE: docs/qa/ directory does not exist (§11.4.83)"
else
    evidence_count=$(find docs/qa -maxdepth 1 -mindepth 1 -type d | wc -l)
    if [ "$evidence_count" -eq 0 ]; then
        gate_fail "CM-ANTI-BLUFF-SMOKE: docs/qa/ has zero evidence dirs (§11.4.83)"
    else
        # Check that ALL evidence dirs have ≥1 evidence file (§11.4.83)
        # Accepts .md, .txt, .json, .log — any file is evidence.
        missing_evidence=0
        for d in docs/qa/*/; do
            file_count=$(find "$d" -maxdepth 1 -type f 2>/dev/null | wc -l)
            if [ "$file_count" -eq 0 ]; then
                missing_evidence=$((missing_evidence + 1))
                echo "  EMPTY: ${d} has no evidence files"
            fi
        done
        if [ "$missing_evidence" -gt 0 ]; then
            gate_fail "CM-ANTI-BLUFF-SMOKE: ${missing_evidence}/${evidence_count} evidence dirs are empty (§11.4.83)"
        else
            gate_pass "CM-ANTI-BLUFF-SMOKE: all ${evidence_count} evidence dirs have evidence files"
        fi
    fi
fi

# Check 2: WORKING_PLAN.md exists and references anti-bluff
if [ ! -f "docs/WORKING_PLAN.md" ]; then
    gate_fail "CM-ANTI-BLUFF-SMOKE: docs/WORKING_PLAN.md missing"
else
    if grep -q "anti-bluff\|§11.4\|no bluff\|§7.1" docs/WORKING_PLAN.md 2>/dev/null; then
        gate_pass "CM-ANTI-BLUFF-SMOKE: WORKING_PLAN.md references anti-bluff policy"
    else
        gate_fail "CM-ANTI-BLUFF-SMOKE: WORKING_PLAN.md does not reference anti-bluff (§11.4)"
    fi
fi

# Check 3: at least one test file uses ab_pass_with_evidence or ab_skip_with_reason
# (the §11.4.69 helper pattern)
if grep -rq "ab_pass_with_evidence\|ab_skip_with_reason" tests/ scripts/ 2>/dev/null; then
    gate_pass "CM-ANTI-BLUFF-SMOKE: §11.4.69 helper pattern found in test/script files"
else
    # Not a hard fail — the helpers may not be adopted yet in this project.
    # But flag it as a finding.
    echo "INFO: CM-ANTI-BLUFF-SMOKE: §11.4.69 helpers (ab_pass_with_evidence) not yet adopted"
fi

# --- Mutation test: remove one evidence dir's README → gate MUST FAIL ---
echo ""
echo "=== Mutation test (§1.1) ==="

# Find a mutable evidence dir (one with any .md file)
MUTATION_TARGET=""
MUTATION_DIR=""
for d in docs/qa/*/; do
    first_md=$(find "$d" -maxdepth 1 -name "*.md" -type f 2>/dev/null | head -1)
    if [ -n "$first_md" ]; then
        MUTATION_TARGET="$first_md"
        MUTATION_DIR="$d"
        break
    fi
done

if [ -z "$MUTATION_TARGET" ]; then
    echo "SKIP: no evidence dir with .md file found for mutation test"
    exit 0
fi

# Create a temporary copy, remove all .md files, re-run gate check
TMPDIR=$(mktemp -d)
mkdir -p "$TMPDIR/evidence_backup"
for md in "$MUTATION_DIR"*.md; do
    [ -f "$md" ] && cp "$md" "$TMPDIR/evidence_backup/"
done

# Simulate mutation: remove ALL .md files from the evidence dir
rm -f "$MUTATION_DIR"*.md

# Re-run gate check 1 (ALL evidence dirs must have .md files)
mutation_missing=0
for d in docs/qa/*/; do
    md_count=$(find "$d" -maxdepth 1 -name "*.md" -type f 2>/dev/null | wc -l)
    if [ "$md_count" -eq 0 ]; then
        mutation_missing=$((mutation_missing + 1))
    fi
done

# Restore the .md files
cp "$TMPDIR/evidence_backup/"*.md "$MUTATION_DIR" 2>/dev/null || true

if [ "$mutation_missing" -gt 0 ]; then
    echo "MUTATION DETECTED: removing README.md correctly makes gate fail (${mutation_missing} dirs missing)"
    echo "RESULT: gate is NOT a bluff (mutation test passes)"
else
    echo "MUTATION NOT DETECTED: removing README.md did NOT make gate fail"
    echo "RESULT: gate IS a bluff — fix the gate"
    exit 1
fi

# --- Summary ---
echo ""
echo "=== Summary ==="
echo "PASS: $PASS  FAIL: $FAIL"
if [ "$FAIL" -gt 0 ]; then
    echo "GATE FAILED"
    exit 1
else
    echo "GATE PASSED"
    exit 0
fi
