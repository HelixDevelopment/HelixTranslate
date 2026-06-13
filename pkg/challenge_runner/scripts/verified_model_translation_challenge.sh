#!/bin/bash
set -euo pipefail

# Challenge: Verified Model Translation
# Confirms that translation workflows can use verified models.
# Anti-bluff: Runs real verified factory tests AND mutation-tests factory logic.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

echo "=== Verified Model Translation Challenge ==="

cd "${PROJECT_ROOT}"

# Step 1: Build all main binaries
make build >/dev/null 2>&1 || go build -o ./build/unified-translator ./cmd/unified-translator

for BIN in unified-translator api-server grpc-server; do
    if [ ! -f "./build/${BIN}" ]; then
        echo "FAIL: ${BIN} binary not built"
        exit 1
    fi
done

# Step 2: Run real verified factory tests
if ! go test -v -run "TestVerifiedFactory" ./pkg/translator/llm/... >/dev/null 2>&1; then
    echo "FAIL: Verified factory tests failed"
    exit 1
fi

# Step 3: Run selection engine tests
if ! go test -v ./internal/verifier/selection/... >/dev/null 2>&1; then
    echo "FAIL: Selection engine tests failed"
    exit 1
fi

# Step 4: Mutation test — break CreateTranslator and confirm tests fail
echo ">>> Mutation test: breaking VerifiedFactory.CreateTranslator..."
FACTORY_GO="${PROJECT_ROOT}/pkg/translator/llm/verified_factory.go"
sed -i.bak 's/return NewLLMTranslatorWithConfig(transConfig)/return nil, fmt.Errorf("broken factory")/' "${FACTORY_GO}"
MUTATION_FAILED=0
if go test -run "TestVerifiedFactoryKeyResolver" ./pkg/translator/llm/... >/dev/null 2>&1; then
    echo "FAIL: Mutation test did not fail — verified factory test is bluffing"
    MUTATION_FAILED=1
fi
mv "${FACTORY_GO}.bak" "${FACTORY_GO}"

if [ "${MUTATION_FAILED}" -eq 1 ]; then
    exit 1
fi

echo "PASS: Verified model translation works and is protected by mutation tests"
exit 0
