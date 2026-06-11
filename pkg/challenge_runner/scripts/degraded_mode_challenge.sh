#!/bin/bash
set -euo pipefail

# Challenge: Degraded Mode
# Verifies graceful degradation when LLMsVerifier is down.
# Anti-bluff: Runs real fallback manager tests AND mutation-tests degraded mode.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

echo "=== Degraded Mode Challenge ==="

cd "${PROJECT_ROOT}"

# Step 1: Run real degraded mode / fallback tests
if ! go test -v ./pkg/distributed/... >/dev/null 2>&1; then
    echo "FAIL: Distributed/fallback tests failed"
    exit 1
fi

# Step 2: Verify ErrLLMsVerifierUnreachable is used in real code paths
if ! go test -run "TestClientPingUnreachable" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Unreachable error test failed"
    exit 1
fi

# Step 3: Mutation test — break the unreachable error and confirm test fails
echo ">>> Mutation test: breaking unreachable error..."
CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"
sed -i.bak 's/ErrLLMsVerifierUnreachable{URL: c.baseURL}/fmt.Errorf("some other error")/' "${CLIENT_GO}"
MUTATION_FAILED=0
if go test -run "TestClientPingUnreachable" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — unreachable error test is bluffing"
    MUTATION_FAILED=1
fi
mv "${CLIENT_GO}.bak" "${CLIENT_GO}"

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Degraded mode handling works and is protected by mutation tests"
exit 0
