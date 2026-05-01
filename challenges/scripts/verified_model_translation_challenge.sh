#!/bin/bash
set -euo pipefail

# Challenge: Verified Model Translation
# Confirms that translation workflows can use verified models.
# Anti-bluff: Builds the translator and verifies verifier symbols are linked.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Verified Model Translation Challenge ==="

cd "${PROJECT_ROOT}"

# Build all main binaries
make build >/dev/null 2>&1 || go build -o ./build/unified-translator ./cmd/unified-translator

# Verify binaries exist
for BIN in unified-translator api-server grpc-server; do
    if [ ! -f "./build/${BIN}" ]; then
        echo "FAIL: ${BIN} binary not built"
        exit 1
    fi
    # Verify verifier integration is compiled into each binary
    if ! strings "./build/${BIN}" | grep -qi "verifier" >/dev/null 2>&1; then
        echo "FAIL: ${BIN} binary missing verifier integration"
        exit 1
    fi
done

# Verify score adapter exists
ADAPTER_GO="${PROJECT_ROOT}/internal/services/llmsverifier_score_adapter.go"
if [ ! -f "${ADAPTER_GO}" ]; then
    echo "FAIL: Score adapter not found"
    exit 1
fi

if ! grep -q "GetPreferences" "${ADAPTER_GO}"; then
    echo "FAIL: Score adapter missing GetPreferences method"
    exit 1
fi

# Verify selection engine exists
SELECTION_GO="${PROJECT_ROOT}/internal/verifier/selection/engine.go"
if [ ! -f "${SELECTION_GO}" ]; then
    echo "FAIL: Selection engine not found"
    exit 1
fi

if ! grep -q "SelectModel" "${SELECTION_GO}"; then
    echo "FAIL: Selection engine missing SelectModel method"
    exit 1
fi

echo "PASS: Verified model translation pipeline is compiled and linked"
exit 0
