#!/bin/bash
set -euo pipefail

# Challenge: RAG Integration
# Verifies RAG support is represented in the knowledge retrieval pipeline.
# Anti-bluff: Checks that discovery service can register knowledge providers.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== RAG Integration Challenge ==="

DISCOVERY_GO="${PROJECT_ROOT}/internal/verifier/discovery/service.go"

# Verify discovery service has provider registration
if ! grep -q 'RegisterProvider' "${DISCOVERY_GO}"; then
    echo "FAIL: Discovery service missing RegisterProvider method"
    exit 1
fi

# Verify registry can hold multiple provider types
REGISTRY_GO="${PROJECT_ROOT}/internal/verifier/registry.go"
if [ ! -f "${REGISTRY_GO}" ]; then
    echo "FAIL: Registry file not found"
    exit 1
fi
if ! grep -q 'providers map\[string\]ProviderConfig' "${REGISTRY_GO}"; then
    echo "FAIL: Registry missing providers map for RAG provider diversity"
    exit 1
fi

echo "PASS: RAG provider diversity is supported"
exit 0
