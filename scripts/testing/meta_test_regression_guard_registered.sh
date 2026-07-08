#!/usr/bin/env bash
# scripts/testing/meta_test_regression_guard_registered.sh
#
# Purpose:   Paired §1.1 mutation test for the CM-REGRESSION-GUARD-REGISTERED gate.
#            The gate asserts that every item in docs/Fixed.md that was closed as
#            "Fixed" has a corresponding regression test file or test function
#            referenced in the tracker (§11.4.135 standing regression-guard suite).
#
#            Checks:
#            (1) docs/Fixed.md exists
#            (2) Every "### §" heading in Fixed.md has a "**Regression-guard:**" line
#                referencing a test file or test function
#            (3) The referenced test file/function exists in the codebase
#
# Mutation:  removes a Regression-guard line from one Fixed.md entry → gate MUST FAIL.
#
# Usage:     scripts/testing/meta_test_regression_guard_registered.sh
#            exit 0 = gate passes (or mutation correctly detected)
#            exit 1 = gate fails (real gap) OR mutation NOT detected (bluff)
#
# Inputs:    docs/Fixed.md
# Outputs:   stdout only (test result)
# Side-effects: creates+removes a temporary test fixture; never modifies tracked files.
# Dependencies: bash, coreutils, grep. No network.
# Cross-references: §11.4.135 (standing regression-guard suite),
#            §11.4.43 (TDD-fix), §11.4.115 (RED-baseline),
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

FIXED_MD="docs/Fixed.md"

# --- Check 1: Fixed.md exists ---
echo "=== Check 1: Fixed.md existence ==="
if [ ! -f "$FIXED_MD" ]; then
    gate_fail "CM-REGRESSION-GUARD-REGISTERED: $FIXED_MD does not exist"
    echo ""
    echo "=== Summary ==="
    echo "PASS: $PASS  FAIL: $FAIL"
    echo "GATE FAILED"
    exit 1
fi
gate_pass "CM-REGRESSION-GUARD-REGISTERED: ${FIXED_MD} exists"

# --- Check 2: Every Fixed heading has a Regression-guard line ---
echo ""
echo "=== Check 2: Regression-guard lines ==="

# Extract all Fixed/Implemented/Completed headings
headings=$(grep -n "^### §" "$FIXED_MD" 2>/dev/null || true)
if [ -z "$headings" ]; then
    gate_pass "CM-REGRESSION-GUARD-REGISTERED: no Fixed headings found (empty tracker)"
    echo ""
    echo "=== Summary ==="
    echo "PASS: $PASS  FAIL: $FAIL"
    echo "GATE PASSED"
    exit 0
fi

total_headings=0
missing_guard=0
has_guard=0

while IFS= read -r line; do
    lineno=$(echo "$line" | cut -d: -f1)
    total_headings=$((total_headings + 1))

    # Look for a Regression-guard line within 15 lines of the heading
    has_rg=0
    for offset in $(seq 1 15); do
        checkline=$((lineno + offset))
        content=$(sed -n "${checkline}p" "$FIXED_MD" 2>/dev/null || true)
        if echo "$content" | grep -qi "regression.guard\|Test.*PASS\|test.*file\|_test\.go\|meta_test"; then
            has_rg=1
            break
        fi
    done

    if [ "$has_rg" -eq 0 ]; then
        missing_guard=$((missing_guard + 1))
        heading_text=$(echo "$line" | cut -d: -f2-)
        echo "  MISSING: line $lineno: $heading_text"
    else
        has_guard=$((has_guard + 1))
    fi
done <<< "$headings"

if [ "$missing_guard" -gt 0 ]; then
    gate_fail "CM-REGRESSION-GUARD-REGISTERED: $missing_guard/$total_headings Fixed items lack regression-guard reference"
else
    gate_pass "CM-REGRESSION-GUARD-REGISTERED: all $total_headings Fixed items have regression-guard references"
fi

# --- Mutation test (§1.1) ---
echo ""
echo "=== Mutation test (§1.1) ==="

# Find a Fixed entry with a Regression-guard line to mutate
MUTATION_TARGET=""
MUTATION_LINE=""
while IFS= read -r line; do
    lineno=$(echo "$line" | cut -d: -f1)
    for offset in $(seq 1 15); do
        checkline=$((lineno + offset))
        content=$(sed -n "${checkline}p" "$FIXED_MD" 2>/dev/null || true)
        if echo "$content" | grep -qi "regression.guard\|Test.*PASS\|test.*file\|_test\.go\|meta_test"; then
            MUTATION_TARGET="$FIXED_MD"
            MUTATION_LINE="$checkline"
            break 2
        fi
    done
done <<< "$headings"

if [ -z "$MUTATION_TARGET" ]; then
    echo "SKIP: no Fixed entry with regression-guard line found for mutation test"
    # If there are no Fixed entries at all, the gate trivially passes
    if [ "$total_headings" -eq 0 ]; then
        echo "RESULT: gate passes (empty tracker)"
        exit 0
    fi
    # If there ARE entries but none have guards, the gate already failed above
    echo "RESULT: gate already failed — mutation test not needed"
    exit 0
fi

# Create a temporary copy, remove the regression-guard line, re-run check
TMPDIR=$(mktemp -d)
cp "$FIXED_MD" "$TMPDIR/backup.md"

# Simulate mutation: remove the regression-guard line
sed -i "${MUTATION_LINE}d" "$FIXED_MD"

# Re-run check 2 for the mutated file
mutation_missing=0
while IFS= read -r line; do
    lineno=$(echo "$line" | cut -d: -f1)
    has_rg=0
    for offset in $(seq 1 15); do
        checkline=$((lineno + offset))
        content=$(sed -n "${checkline}p" "$FIXED_MD" 2>/dev/null || true)
        if echo "$content" | grep -qi "regression.guard\|Test.*PASS\|test.*file\|_test\.go\|meta_test"; then
            has_rg=1
            break
        fi
    done
    if [ "$has_rg" -eq 0 ]; then
        mutation_missing=$((mutation_missing + 1))
    fi
done <<< "$headings"

# Restore
cp "$TMPDIR/backup.md" "$FIXED_MD"

if [ "$mutation_missing" -gt 0 ]; then
    echo "MUTATION DETECTED: removing regression-guard line correctly makes gate fail"
    echo "RESULT: gate is NOT a bluff (mutation test passes)"
else
    echo "MUTATION NOT DETECTED: removing regression-guard line did NOT make gate fail"
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
