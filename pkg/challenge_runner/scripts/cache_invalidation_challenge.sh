#!/bin/bash
set -euo pipefail

# Challenge: Cache Invalidation
# Verifies that cache refresh triggers re-fetch from LLMsVerifier.
# Anti-bluff: Runs the actual unit tests AND mutation-tests the cache logic.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Cache Invalidation Challenge ==="

cd "${PROJECT_ROOT}"

# Step 1: Run the real cache tests
if ! go test -v -run "TestClientInvalidateCache|TestClientGetVerifiedModelsCache" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Cache invalidation unit tests failed"
    exit 1
fi

# Step 2: Mutation test — break InvalidateCache and confirm tests fail
echo ">>> Mutation test: breaking InvalidateCache..."
CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"
ORIGINAL=$(grep -n "c.lastFetch = time.Time{}" "${CLIENT_GO}")
if [ -z "${ORIGINAL}" ]; then
    echo "FAIL: Could not locate InvalidateCache mutation target"
    exit 1
fi

# Backup
sed -i.bak 's/c.lastFetch = time.Time{}/c.lastFetch = time.Now()/' "${CLIENT_GO}"
MUTATION_FAILED=0
if go test -run "TestClientInvalidateCache" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — tests are bluffing"
    MUTATION_FAILED=1
fi
# Restore
mv "${CLIENT_GO}.bak" "${CLIENT_GO}"

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Cache invalidation works and is protected by mutation tests"
exit 0
