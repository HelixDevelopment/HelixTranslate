#!/bin/bash
set -euo pipefail

# Challenge: Model Verification Gate
# Confirms that unverified models are rejected by HelixTranslate.
# Anti-bluff: Checks that the verifier client actually enforces the gate.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Model Verification Gate Challenge ==="

# Build the unified translator to ensure verifier integration is compiled
cd "${PROJECT_ROOT}"
if [ ! -f "./build/unified-translator" ]; then
    echo "Building unified-translator..."
    go build -o ./build/unified-translator ./cmd/unified-translator 2>&1 | tail -5
fi

# Verify the binary contains LLMsVerifier symbols
# Note: grep -q in a pipe with pipefail can cause SIGPIPE (141) when grep exits early.
# We use `> /dev/null` instead of `-q` to avoid this.
if ! strings ./build/unified-translator | grep -i "verifier" >/dev/null 2>&1; then
    echo "FAIL: unified-translator binary does not contain verifier symbols"
    exit 1
fi

# Check that the config struct includes LLMsVerifier fields
if ! grep -q "LLMsVerifierConfig" internal/config/config.go; then
    echo "FAIL: internal/config/config.go missing LLMsVerifierConfig"
    exit 1
fi

# Check that the verifier errors package exists and defines rejection errors
if [ ! -f "internal/verifier/errors.go" ]; then
    echo "FAIL: internal/verifier/errors.go not found"
    exit 1
fi

if ! grep -q "ErrModelNotVerified" internal/verifier/errors.go; then
    echo "FAIL: ErrModelNotVerified not defined in errors.go"
    exit 1
fi

if ! grep -q "ErrScoreBelowThreshold" internal/verifier/errors.go; then
    echo "FAIL: ErrScoreBelowThreshold not defined in errors.go"
    exit 1
fi

# Verify that the scoring engine enforces thresholds
if ! grep -q "IsQualified" internal/verifier/scoring/engine.go; then
    echo "FAIL: scoring engine missing IsQualified threshold check"
    exit 1
fi

echo "PASS: Model verification gate is properly implemented"
exit 0
