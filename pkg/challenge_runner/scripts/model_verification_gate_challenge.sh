#!/bin/bash
set -euo pipefail

# Challenge: Model Verification Gate
# Confirms that unverified models are rejected by HelixTranslate.
# Anti-bluff: Runs real verifier pipeline tests AND mutation-tests the gate.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

echo "=== Model Verification Gate Challenge ==="

cd "${PROJECT_ROOT}"

# Step 1: Build the unified translator to ensure verifier integration compiles
if [ ! -f "./build/unified-translator" ]; then
    echo "Building unified-translator..."
    go build -o ./build/unified-translator ./cmd/unified-translator 2>&1 | tail -5
fi

# Step 2: Run real verifier pipeline tests
if ! go test -v ./internal/verifier/... >/dev/null 2>&1; then
    echo "FAIL: Verifier tests failed"
    exit 1
fi

# Step 3: Run registry filter tests
if ! go test -run "TestRegistryFilterVerified" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Registry filter test failed"
    exit 1
fi

# Step 4: Mutation test — break FilterVerified and confirm tests fail
echo ">>> Mutation test: breaking FilterVerified..."
REGISTRY_GO="${PROJECT_ROOT}/internal/verifier/registry.go"
# Restore the source even if interrupted during the long `go test` step,
# otherwise mutated source + .bak leak into the working tree (§11.4.84).
restore_registry_go() {
    if [ -f "${REGISTRY_GO}.bak" ]; then
        mv "${REGISTRY_GO}.bak" "${REGISTRY_GO}"
    fi
}
trap restore_registry_go EXIT INT TERM

sed -i.bak 's/m.VerificationStatus == "verified"/m.VerificationStatus == "always-pass"/' "${REGISTRY_GO}"
MUTATION_FAILED=0
if go test -run "TestRegistryFilterVerified" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — registry filter test is bluffing"
    MUTATION_FAILED=1
fi
restore_registry_go
trap - EXIT INT TERM

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Model verification gate works and is protected by mutation tests"
exit 0
