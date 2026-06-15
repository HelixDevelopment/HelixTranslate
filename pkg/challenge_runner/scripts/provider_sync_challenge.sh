#!/bin/bash
set -euo pipefail

# Challenge: Provider Synchronization
# Verifies that the three-tier discovery pipeline works.
# Anti-bluff: Runs real discovery tests AND mutation-tests discovery logic.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

echo "=== Provider Sync Challenge ==="

cd "${PROJECT_ROOT}"

# Step 1: Run real discovery tests
if ! go test -v ./internal/verifier/discovery/... >/dev/null 2>&1; then
    echo "FAIL: Discovery tests failed"
    exit 1
fi

# Step 2: Verify at least 25 providers are registered in the code
CONFIG_GO="${PROJECT_ROOT}/internal/config/config.go"
PROVIDER_COUNT=$(grep -A 30 'envMappings := map\[string\]string{' "${CONFIG_GO}" | grep -c '".*":' || true)
if [ "${PROVIDER_COUNT}" -lt 25 ]; then
    echo "FAIL: Only ${PROVIDER_COUNT} providers mapped in config, expected at least 25"
    exit 1
fi

# Step 3: Mutation test — break Discover and confirm tests fail
echo ">>> Mutation test: breaking Discover..."
SERVICE_GO="${PROJECT_ROOT}/internal/verifier/discovery/service.go"
# Restore the source even if interrupted during the long `go test` step,
# otherwise mutated source + .bak leak into the working tree (§11.4.84).
restore_service_go() {
    if [ -f "${SERVICE_GO}.bak" ]; then
        mv "${SERVICE_GO}.bak" "${SERVICE_GO}"
    fi
}
trap restore_service_go EXIT INT TERM

sed -i.bak 's/s.lastSync = time.Now()/\/\/ s.lastSync = time.Now()/' "${SERVICE_GO}"
MUTATION_FAILED=0
if go test -run "TestLastSync" ./internal/verifier/discovery/ >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — discovery sync test is bluffing"
    MUTATION_FAILED=1
fi
restore_service_go
trap - EXIT INT TERM

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Provider synchronization works and is protected by mutation tests"
exit 0
