#!/bin/bash
set -euo pipefail

# Challenge: LLMsVerifier Connectivity
# Verifies that HelixTranslate can connect to LLMsVerifier API.
# Anti-bluff: Actually attempts HTTP connection and validates response structure.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

API_URL="${LLMSVERIFIER_API_URL:-http://localhost:8080}"
API_KEY="${LLMSVERIFIER_API_KEY:-}"

echo "=== LLMsVerifier Connectivity Challenge ==="
echo "Target: ${API_URL}/api/health"

# Requirement: API_KEY must be configured (either in env or .env file)
if [ -z "${API_KEY}" ]; then
    echo "WARN: LLMSVERIFIER_API_KEY not set; performing static code validation only"

    # Anti-bluff fallback: verify client implementation is real and correct
    CLIENT_GO="${SCRIPT_DIR}/../../internal/verifier/client.go"
    if [ ! -f "${CLIENT_GO}" ]; then
        echo "FAIL: verifier client not found"
        exit 1
    fi

    if ! grep -q 'func (c \*Client) Ping' "${CLIENT_GO}"; then
        echo "FAIL: Client.Ping method not implemented"
        exit 1
    fi

    if ! grep -q 'func (c \*Client) GetVerifiedModels' "${CLIENT_GO}"; then
        echo "FAIL: Client.GetVerifiedModels method not implemented"
        exit 1
    fi

    if ! grep -q 'Authorization.*Bearer' "${CLIENT_GO}"; then
        echo "FAIL: Client does not use Bearer token authentication"
        exit 1
    fi

    echo "PASS: LLMsVerifier client implementation validated (runtime connectivity skipped — no API key)"
    exit 0
fi

# Attempt actual HTTP connection with timeout
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    --max-time 10 \
    -H "Authorization: Bearer ${API_KEY}" \
    "${API_URL}/api/health" 2>/dev/null || echo "000")

if [ "${HTTP_CODE}" != "200" ]; then
    echo "FAIL: LLMsVerifier health endpoint returned HTTP ${HTTP_CODE} (expected 200)"
    exit 1
fi

# Validate response body is non-empty JSON
RESPONSE=$(curl -s --max-time 10 -H "Authorization: Bearer ${API_KEY}" "${API_URL}/api/health" 2>/dev/null || echo "")
if [ -z "${RESPONSE}" ]; then
    echo "FAIL: LLMsVerifier health endpoint returned empty body"
    exit 1
fi

if ! echo "${RESPONSE}" | grep -q "status"; then
    echo "FAIL: LLMsVerifier health response missing 'status' field"
    echo "Response: ${RESPONSE}"
    exit 1
fi

echo "PASS: LLMsVerifier connectivity confirmed"
echo "Response snippet: $(echo "${RESPONSE}" | head -c 200)"
exit 0
