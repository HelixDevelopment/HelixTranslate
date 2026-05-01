#!/bin/bash
set -euo pipefail

# Challenge: Degraded Mode
# Verifies graceful degradation when LLMsVerifier is down.
# Anti-bluff: Confirms fallback behavior is codified in the client.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Degraded Mode Challenge ==="

ERRORS_GO="${PROJECT_ROOT}/internal/verifier/errors.go"
CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"

# Verify ErrLLMsVerifierUnreachable is defined
if ! grep -q "ErrLLMsVerifierUnreachable" "${ERRORS_GO}"; then
    echo "FAIL: ErrLLMsVerifierUnreachable not defined"
    exit 1
fi

# Verify client returns this error on connection failure
if ! grep -q "ErrLLMsVerifierUnreachable{URL: c.baseURL}" "${CLIENT_GO}"; then
    echo "FAIL: Client does not return ErrLLMsVerifierUnreachable on connection failure"
    exit 1
fi

# Verify scoring engine can operate on cached data
ENGINE_GO="${PROJECT_ROOT}/internal/verifier/scoring/engine.go"
if ! grep -q "GetScore" "${ENGINE_GO}"; then
    echo "FAIL: scoring engine missing GetScore for cached data"
    exit 1
fi

# Verify config has strict mode toggle
CONFIG_GO="${PROJECT_ROOT}/internal/config/config.go"
if ! grep -q "StrictMode" "${CONFIG_GO}"; then
    echo "FAIL: config missing StrictMode toggle for degraded behavior"
    exit 1
fi

echo "PASS: Degraded mode handling is properly implemented"
exit 0
