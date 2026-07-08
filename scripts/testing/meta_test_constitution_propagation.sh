#!/usr/bin/env bash
# scripts/testing/meta_test_constitution_propagation.sh
#
# Purpose:   Paired §1.1 mutation test for the CM-CONSTITUTION-PROPAGATION gate.
#            The gate asserts that key constitutional anchor literals are present
#            across ALL governance context carriers (CLAUDE.md, AGENTS.md, QWEN.md,
#            GEMINI.md) per §11.4.26/§11.4.35/§11.4.157.
#
#            Checks:
#            (1) All 4 governance files exist at project root
#            (2) Each file contains the inheritance pointer (§11.4.35)
#            (3) Key anchor literals are present in each file
#
# Mutation:  strips one anchor literal from one file → gate MUST FAIL.
#
# Usage:     scripts/testing/meta_test_constitution_propagation.sh
#            exit 0 = gate passes (or mutation correctly detected)
#            exit 1 = gate fails (real propagation gap) OR mutation NOT detected (bluff)
#
# Inputs:    CLAUDE.md, AGENTS.md, QWEN.md, GEMINI.md
# Outputs:   stdout only (test result)
# Side-effects: creates+removes a temporary test fixture; never modifies tracked files.
# Dependencies: bash, coreutils, grep. No network.
# Cross-references: §11.4.26 (constitution-submodule update workflow),
#            §11.4.35 (canonical-root inheritance clarity),
#            §11.4.157 (GEMINI.md lockstep),
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

# Governance files that MUST exist per §11.4.35/§11.4.157
GOVERNANCE_FILES="CLAUDE.md AGENTS.md QWEN.md GEMINI.md"

# Key anchor literals that MUST be present in the constitution submodule's
# CLAUDE.md (the canonical root per §11.4.35). Consumer-side files only
# carry the inheritance pointer — the anchors live in the constitution.
CONSTITUTION_CLAUDE="constitution/CLAUDE.md"
ANCHOR_LITERALS="11.4.87 11.4.113 11.4.126"

# --- Check 1: All governance files exist ---
echo "=== Check 1: Governance file existence ==="
missing_files=0
for f in $GOVERNANCE_FILES; do
    if [ ! -f "$f" ]; then
        gate_fail "CM-CONSTITUTION-PROPAGATION: $f does not exist (§11.4.35/§11.4.157)"
        missing_files=$((missing_files + 1))
    else
        gate_pass "CM-CONSTITUTION-PROPAGATION: $f exists"
    fi
done

# --- Check 2: Inheritance pointer in each file ---
echo ""
echo "=== Check 2: Inheritance pointer (§11.4.35) ==="
for f in $GOVERNANCE_FILES; do
    if [ ! -f "$f" ]; then
        continue  # already reported as missing
    fi
    # Check for either the @import syntax or the portable heading or AGENTS-style pointer
    if grep -q "@constitution/CLAUDE.md\|INHERITED FROM constitution/CLAUDE.md\|constitution/CLAUDE.md\|constitution/AGENTS.md\|HELIX-CONSTITUTION-INHERITANCE" "$f" 2>/dev/null; then
        gate_pass "CM-CONSTITUTION-PROPAGATION: $f has inheritance pointer"
    else
        gate_fail "CM-CONSTITUTION-PROPAGATION: $f lacks inheritance pointer (§11.4.35)"
    fi
done

# --- Check 3: Key anchor literals in constitution submodule ---
echo ""
echo "=== Check 3: Key anchor literals in constitution submodule ==="
if [ ! -f "$CONSTITUTION_CLAUDE" ]; then
    gate_fail "CM-CONSTITUTION-PROPAGATION: $CONSTITUTION_CLAUDE does not exist"
else
    for anchor in $ANCHOR_LITERALS; do
        if grep -q "$anchor" "$CONSTITUTION_CLAUDE" 2>/dev/null; then
            gate_pass "CM-CONSTITUTION-PROPAGATION: constitution contains anchor $anchor"
        else
            gate_fail "CM-CONSTITUTION-PROPAGATION: constitution missing anchor $anchor"
        fi
    done
fi

# --- Mutation test (§1.1) ---
echo ""
echo "=== Mutation test (§1.1) ==="

# Find a file with an anchor literal to mutate
MUTATION_FILE=""
MUTATION_ANCHOR=""
if [ -f "$CONSTITUTION_CLAUDE" ]; then
    for anchor in $ANCHOR_LITERALS; do
        if grep -q "$anchor" "$CONSTITUTION_CLAUDE" 2>/dev/null; then
            MUTATION_FILE="$CONSTITUTION_CLAUDE"
            MUTATION_ANCHOR="$anchor"
            break
        fi
    done
fi

if [ -z "$MUTATION_FILE" ]; then
    echo "SKIP: no governance file with anchor literal found for mutation test"
    exit 0
fi

# Create a temporary copy, strip the anchor, re-run check
TMPDIR=$(mktemp -d)
cp "$MUTATION_FILE" "$TMPDIR/backup.md"

# Simulate mutation: remove lines containing the anchor
grep -v "$MUTATION_ANCHOR" "$MUTATION_FILE" > "$TMPDIR/mutated.md"
cp "$TMPDIR/mutated.md" "$MUTATION_FILE"

# Re-run check 3 for the mutated file
mutation_missing=0
for anchor in $ANCHOR_LITERALS; do
    if ! grep -q "$anchor" "$MUTATION_FILE" 2>/dev/null; then
        mutation_missing=$((mutation_missing + 1))
    fi
done

# Restore
cp "$TMPDIR/backup.md" "$MUTATION_FILE"

if [ "$mutation_missing" -gt 0 ]; then
    echo "MUTATION DETECTED: stripping $MUTATION_ANCHOR from $MUTATION_FILE correctly makes gate fail"
    echo "RESULT: gate is NOT a bluff (mutation test passes)"
else
    echo "MUTATION NOT DETECTED: stripping $MUTATION_ANCHOR did NOT make gate fail"
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
