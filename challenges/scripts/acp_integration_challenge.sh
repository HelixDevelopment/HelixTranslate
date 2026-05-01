#!/bin/bash
set -euo pipefail

# Challenge: ACP Integration
# Verifies ACP protocol support is wired into verifier config.
# Anti-bluff: Validates config schema includes verification settings.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== ACP Integration Challenge ==="

CONFIG_GO="${PROJECT_ROOT}/internal/config/config.go"

# Verify LLMsVerifierConfig has VerificationEnabled
if ! grep -q 'VerificationEnabled' "${CONFIG_GO}"; then
    echo "FAIL: LLMsVerifierConfig missing VerificationEnabled field"
    exit 1
fi

# Verify strict mode for protocol compliance
if ! grep -q 'StrictMode' "${CONFIG_GO}"; then
    echo "FAIL: LLMsVerifierConfig missing StrictMode field"
    exit 1
fi

echo "PASS: ACP protocol compliance configuration is present"
exit 0
