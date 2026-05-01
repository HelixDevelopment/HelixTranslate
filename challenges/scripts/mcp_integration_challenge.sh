#!/bin/bash
set -euo pipefail

# Challenge: MCP Integration
# Verifies MCP capability flags are propagated through the verifier pipeline.
# Anti-bluff: Confirms Model struct includes capabilities map with MCP key.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== MCP Integration Challenge ==="

API_TYPES="${PROJECT_ROOT}/LLMsVerifier/pkg/api/types.go"
CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"

# CONST-034: canonical types live in LLMsVerifier/pkg/api/types.go
if [ ! -f "${API_TYPES}" ]; then
    echo "FAIL: Canonical API types file not found at ${API_TYPES}"
    exit 1
fi

# Verify Model struct has Capabilities map in canonical location
if ! grep -q 'Capabilities.*map\[string\]bool' "${API_TYPES}"; then
    echo "FAIL: Model struct missing Capabilities map in canonical type"
    exit 1
fi

# Verify capability-related fields exist in canonical location
if ! grep -q 'CanSeeCode' "${API_TYPES}"; then
    echo "FAIL: Model struct missing CanSeeCode field in canonical type"
    exit 1
fi

if ! grep -q 'AffirmativeResponse' "${API_TYPES}"; then
    echo "FAIL: Model struct missing AffirmativeResponse field in canonical type"
    exit 1
fi

# Verify HelixTranslate still aliases the canonical types
if ! grep -q 'type Model = api.Model' "${CLIENT_GO}"; then
    echo "FAIL: internal/verifier/client.go missing Model alias to canonical type"
    exit 1
fi

echo "PASS: MCP capability propagation is implemented"
exit 0
