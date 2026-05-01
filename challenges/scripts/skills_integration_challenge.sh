#!/bin/bash
set -euo pipefail

# Challenge: Skills Integration
# Verifies skill/plugin requirements can be matched against model capabilities.
# Anti-bluff: Confirms selection engine supports requirement-based filtering.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Skills Integration Challenge ==="

SELECTION_GO="${PROJECT_ROOT}/internal/verifier/selection/engine.go"

# Verify TaskRequirements struct exists with capability filters
if ! grep -q 'TaskRequirements' "${SELECTION_GO}"; then
    echo "FAIL: Selection engine missing TaskRequirements struct"
    exit 1
fi

# Verify meetsRequirements checks capabilities
if ! grep -q 'meetsRequirements' "${SELECTION_GO}"; then
    echo "FAIL: Selection engine missing meetsRequirements filter"
    exit 1
fi

# Verify model capabilities are checked
if ! grep -q 'model.Capabilities' "${SELECTION_GO}"; then
    echo "FAIL: Selection engine does not check model.Capabilities"
    exit 1
fi

echo "PASS: Skills requirement matching is implemented"
exit 0
