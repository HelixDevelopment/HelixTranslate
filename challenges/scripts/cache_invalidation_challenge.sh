#!/bin/bash
set -euo pipefail

# Challenge: Cache Invalidation
# Verifies that cache refresh triggers re-fetch from LLMsVerifier.
# Anti-bluff: Tests the InvalidateCache method exists and functions.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Cache Invalidation Challenge ==="

CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"
if [ ! -f "${CLIENT_GO}" ]; then
    echo "FAIL: verifier client not found"
    exit 1
fi

# Verify InvalidateCache method exists
if ! grep -q "func (c \*Client) InvalidateCache()" "${CLIENT_GO}"; then
    echo "FAIL: Client.InvalidateCache() method not found"
    exit 1
fi

# Verify cache TTL is configurable
if ! grep -q "cacheTTL" "${CLIENT_GO}"; then
    echo "FAIL: cacheTTL field not found in client"
    exit 1
fi

# Verify lastFetch tracking
if ! grep -q "lastFetch" "${CLIENT_GO}"; then
    echo "FAIL: lastFetch field not found in client"
    exit 1
fi

# Verify GetVerifiedModels uses cache
if ! grep -q "time.Since(c.lastFetch) < c.cacheTTL" "${CLIENT_GO}"; then
    echo "FAIL: GetVerifiedModels does not implement cache TTL check"
    exit 1
fi

echo "PASS: Cache invalidation is properly implemented"
exit 0
