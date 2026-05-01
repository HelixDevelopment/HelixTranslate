#!/bin/bash
set -euo pipefail

# Challenge: Provider Synchronization
# Verifies that all 25+ providers are configured in the system.
# Anti-bluff: Checks config.go for provider mappings and verifies count.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Provider Sync Challenge ==="

CONFIG_GO="${PROJECT_ROOT}/internal/config/config.go"

# Count provider entries in envMappings
PROVIDER_COUNT=$(grep -A 30 'envMappings := map\[string\]string{' "${CONFIG_GO}" | grep -c '".*":' || true)

if [ "${PROVIDER_COUNT}" -lt 25 ]; then
    echo "FAIL: Only ${PROVIDER_COUNT} providers mapped in config, expected at least 25"
    exit 1
fi

# Verify specific critical providers exist
CRITICAL_PROVIDERS=("openai" "anthropic" "deepseek" "zhipu" "gemini" "groq" "cohere" "mistral" "xai" "replicate" "cerebras" "cloudflare" "siliconflow" "hyperbolic" "togetherai" "sambanova" "kimi" "novita" "nlpcloud" "upstage" "sarvam" "modal" "publicai" "nia" "vulavula")
MISSING=0
for PROVIDER in "${CRITICAL_PROVIDERS[@]}"; do
    if ! grep -q "\"${PROVIDER}\":" "${CONFIG_GO}"; then
        echo "MISSING: provider '${PROVIDER}' not found in config mappings"
        MISSING=$((MISSING + 1))
    fi
done

if [ "${MISSING}" -gt 0 ]; then
    echo "FAIL: ${MISSING} critical providers are missing from configuration"
    exit 1
fi

echo "PASS: All ${PROVIDER_COUNT} providers are configured for synchronization"
exit 0
