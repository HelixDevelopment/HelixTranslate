#!/bin/bash
set -euo pipefail

# Challenge: API Key Provisioning
# Verifies that API keys are loaded from .env file and NOT git-tracked.
# Anti-bluff: Checks file permissions, git ignore status, env loading logic,
# AND mutation-tests the env-loading code.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

echo "=== API Key Provisioning Challenge ==="

# 1. .env file must exist with restricted permissions
ENV_FILE="${PROJECT_ROOT}/.env"
if [ ! -f "${ENV_FILE}" ]; then
    echo "FAIL: .env file not found at project root"
    exit 1
fi

PERMS=$(stat -c "%a" "${ENV_FILE}" 2>/dev/null || stat -f "%Lp" "${ENV_FILE}" 2>/dev/null)
if [ "${PERMS}" != "600" ]; then
    echo "FAIL: .env file permissions are ${PERMS}, expected 600"
    exit 1
fi

# 2. .env must be gitignored
if ! git -C "${PROJECT_ROOT}" check-ignore -q "${ENV_FILE}" 2>/dev/null; then
    echo "FAIL: .env file is tracked by git (must be gitignored)"
    exit 1
fi

# 3. Verify config loader reads API keys from environment
CONFIG_GO="${PROJECT_ROOT}/internal/config/config.go"
if ! grep -q "loadAPIKeysFromEnv" "${CONFIG_GO}"; then
    echo "FAIL: config.go missing loadAPIKeysFromEnv function"
    exit 1
fi

# 4. Verify multiple provider keys are mapped
EXPECTED_PROVIDERS=("OPENAI_API_KEY" "ANTHROPIC_API_KEY" "DEEPSEEK_API_KEY" "ZHIPU_API_KEY" "GROQ_API_KEY" "COHERE_API_KEY" "MISTRAL_API_KEY")
for KEY in "${EXPECTED_PROVIDERS[@]}"; do
    if ! grep -q "${KEY}" "${CONFIG_GO}"; then
        echo "FAIL: config.go does not map ${KEY}"
        exit 1
    fi
done

# 5. Verify no hardcoded keys in source
HARDCOUNT=$(grep -rE "(sk-|pk-|api_key|apikey|api-key)\s*[:=]\s*[\"'][^\"']{20,}" --include="*.go" "${PROJECT_ROOT}/pkg" "${PROJECT_ROOT}/internal" "${PROJECT_ROOT}/cmd" 2>/dev/null | grep -v "_test.go" | grep -v "os.Getenv" | wc -l || true)
if [ "${HARDCOUNT}" -gt 0 ]; then
    echo "FAIL: Found ${HARDCOUNT} potential hardcoded API keys in source code"
    exit 1
fi

# 6. Mutation test — break env loading and confirm tests fail
echo ">>> Mutation test: breaking env-based key loading..."
UNIFIED_GO="${PROJECT_ROOT}/cmd/unified-translator/main.go"
# Mutate resolveProviderAPIKey to always return empty string
sed -i.bak '/^func resolveProviderAPIKey/,/^}/ { s/return "flag-key"/return ""/; s/return os.Getenv(envVar)/return ""/; s/return config.APIKey/return ""/; }' "${UNIFIED_GO}"
MUTATION_FAILED=0
if go test -run "TestResolveProviderAPIKey" ./cmd/unified-translator/... >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — API key resolver test is bluffing"
    MUTATION_FAILED=1
fi
mv "${UNIFIED_GO}.bak" "${UNIFIED_GO}"

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: API keys are properly provisioned and protected by mutation tests"
exit 0
