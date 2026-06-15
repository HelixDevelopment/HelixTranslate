#!/bin/bash
set -euo pipefail

# Challenge: Cache Invalidation
# Verifies that cache refresh triggers re-fetch from LLMsVerifier.
# Anti-bluff: Runs the actual unit tests AND mutation-tests the cache logic.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

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

# Restore the source even if interrupted (timeout/SIGINT/SIGTERM) during the
# long `go test` step — otherwise mutated source + .bak leak into the working
# tree (§11.4.84 mutation residue). Trap MUST be armed before the mutating sed.
restore_client_go() {
    if [ -f "${CLIENT_GO}.bak" ]; then
        mv "${CLIENT_GO}.bak" "${CLIENT_GO}"
    fi
}
trap restore_client_go EXIT INT TERM

# Backup
sed -i.bak 's/c.lastFetch = time.Time{}/c.lastFetch = time.Now()/' "${CLIENT_GO}"
MUTATION_FAILED=0
if go test -run "TestClientInvalidateCache" ./internal/verifier/ >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — tests are bluffing"
    MUTATION_FAILED=1
fi
# Restore (trap also restores on interrupt)
restore_client_go
trap - EXIT INT TERM

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Cache invalidation works and is protected by mutation tests"
exit 0
