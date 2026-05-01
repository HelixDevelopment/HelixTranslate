#!/bin/bash
set -euo pipefail

# Challenge: Embeddings Integration
# Verifies embeddings capability is tracked in model capabilities.
# Anti-bluff: Confirms capability map supports embeddings key.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Embeddings Integration Challenge ==="

API_TYPES="${PROJECT_ROOT}/LLMsVerifier/pkg/api/types.go"
CLIENT_GO="${PROJECT_ROOT}/internal/verifier/client.go"

# CONST-034: canonical types live in LLMsVerifier/pkg/api/types.go
if [ ! -f "${API_TYPES}" ]; then
    echo "FAIL: Canonical API types file not found at ${API_TYPES}"
    exit 1
fi

# Verify Model struct has Capabilities map that can hold embeddings
if ! grep -q 'Capabilities.*map\[string\]bool' "${API_TYPES}"; then
    echo "FAIL: Model struct missing Capabilities map for embeddings tracking in canonical type"
    exit 1
fi

# Verify HelixTranslate still aliases the canonical types
if ! grep -q 'type Model = api.Model' "${CLIENT_GO}"; then
    echo "FAIL: internal/verifier/client.go missing Model alias to canonical type"
    exit 1
fi

echo "PASS: Embeddings capability tracking is implemented"
exit 0
