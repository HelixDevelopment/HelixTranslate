#!/bin/bash
set -euo pipefail

# Run all HelixTranslate challenges with anti-bluff validation.
# Every challenge MUST pass for the system to be considered valid.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CHALLENGES=(
    "host_no_auto_suspend_challenge.sh"
    "no_suspend_calls_challenge.sh"
    "llmsverifier_connectivity_challenge.sh"
    "model_verification_gate_challenge.sh"
    "api_key_provisioning_challenge.sh"
    "cache_invalidation_challenge.sh"
    "degraded_mode_challenge.sh"
    "provider_sync_challenge.sh"
    "verified_model_translation_challenge.sh"
    "helixqa_wiring_challenge.sh"
    "anti_bluff_execution_challenge.sh"
)

PASSED=0
FAILED=0

for CHALLENGE in "${CHALLENGES[@]}"; do
    PATH_TO_CHALLENGE="${SCRIPT_DIR}/${CHALLENGE}"
    if [ ! -f "${PATH_TO_CHALLENGE}" ]; then
        echo "SKIP: ${CHALLENGE} not found"
        continue
    fi
    echo ""
    echo ">>> Running ${CHALLENGE}..."
    if bash "${PATH_TO_CHALLENGE}"; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
        echo "!!! ${CHALLENGE} FAILED"
    fi
done

echo ""
echo "========================================"
echo "Challenge Results: ${PASSED} passed, ${FAILED} failed"
echo "========================================"

if [ "${FAILED}" -gt 0 ]; then
    exit 1
fi
exit 0
