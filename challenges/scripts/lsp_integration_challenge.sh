#!/bin/bash
set -euo pipefail

# Challenge: LSP Integration
# Verifies LSP support is declared in capability detection.
# Anti-bluff: Checks selection engine for code-aware requirements.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== LSP Integration Challenge ==="

SELECTION_GO="${PROJECT_ROOT}/internal/verifier/selection/engine.go"

# Verify TaskRequirements includes code requirement
if ! grep -q 'RequireCode' "${SELECTION_GO}"; then
    echo "FAIL: TaskRequirements missing RequireCode field"
    exit 1
fi

# Verify selection engine checks CanSeeCode
if ! grep -q 'model.CanSeeCode' "${SELECTION_GO}"; then
    echo "FAIL: Selection engine does not check CanSeeCode for code tasks"
    exit 1
fi

echo "PASS: LSP integration requirements are enforced"
exit 0
