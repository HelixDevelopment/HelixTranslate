#!/bin/bash
set -euo pipefail

# Challenge: Anti-Bluff Execution Verification (CONST-035 enforcement)
# This is a META-challenge: it verifies that the codebase actually BUILDS
# and that the TESTS actually EXECUTE against real functionality.
#
# Anti-bluff mandate: A test suite that passes while features are broken
# is worse than no tests at all. This challenge enforces that green = usable.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "${SCRIPT_DIR}" rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../.." && pwd))"

echo "=== Anti-Bluff Execution Challenge ==="
echo "Verifying: build success + test execution + mutation failure"
cd "${PROJECT_ROOT}"

# -----------------------------------------------------------------------------
# 1. BUILD VERIFICATION — Code must compile before it can be usable
# -----------------------------------------------------------------------------
echo ">>> Step 1: Building all binaries..."
if ! make build >/dev/null 2>&1; then
    echo "FAIL: make build failed — code does not compile"
    exit 1
fi

for BIN in grpc-server api-server unified-translator; do
    if [ ! -x "./build/${BIN}" ]; then
        echo "FAIL: build/${BIN} is missing or not executable"
        exit 1
    fi
done
echo "PASS: All binaries built successfully"

# -----------------------------------------------------------------------------
# 2. UNIT TEST EXECUTION — Tests must run and pass against real code
# -----------------------------------------------------------------------------
echo ">>> Step 2: Running unit tests..."
if ! go test ./test/unit/... -count=1 -timeout=60s >/dev/null 2>&1; then
    echo "FAIL: Unit tests failed"
    exit 1
fi
echo "PASS: Unit tests execute and pass"

# -----------------------------------------------------------------------------
# 3. INTEGRATION TEST EXECUTION — Cross-package tests must run
# -----------------------------------------------------------------------------
echo ">>> Step 3: Running integration tests..."
if ! go test ./test/integration/... -tags=integration -count=1 -timeout=120s >/dev/null 2>&1; then
    echo "FAIL: Integration tests failed"
    exit 1
fi
echo "PASS: Integration tests execute and pass"

# -----------------------------------------------------------------------------
# 4. MUTATION TEST — Verify tests FAIL when the feature is broken
# This is the core anti-bluff check: green tests must mean working code.
# -----------------------------------------------------------------------------
echo ">>> Step 4: Mutation test — breaking code must fail tests..."

MUTATION_TARGET="${PROJECT_ROOT}/internal/verifier/client.go"
MUTATION_BACKUP="${MUTATION_TARGET}.backup"
REGISTRY_TARGET="${PROJECT_ROOT}/pkg/models/verifier_registry.go"
REGISTRY_BACKUP="${REGISTRY_TARGET}.backup"

# Restore BOTH mutation targets even if interrupted (timeout/SIGINT/SIGTERM)
# during a long `go test` step — otherwise mutated source + .backup leak into
# the working tree (§11.4.84 mutation residue). Trap armed before any mutation.
restore_mutation_targets() {
    if [ -f "${MUTATION_BACKUP}" ]; then
        cp "${MUTATION_BACKUP}" "${MUTATION_TARGET}" && rm -f "${MUTATION_BACKUP}"
    fi
    if [ -f "${REGISTRY_BACKUP}" ]; then
        cp "${REGISTRY_BACKUP}" "${REGISTRY_TARGET}" && rm -f "${REGISTRY_BACKUP}"
    fi
}
trap restore_mutation_targets EXIT INT TERM

# Backup original
cp "${MUTATION_TARGET}" "${MUTATION_BACKUP}"

# Apply mutation: break the Ping method so it always returns an error.
# Portable across BSD (macOS) and GNU sed: BSD `sed -i <script>` consumes the
# script as the backup-extension arg and treats the file path as the script
# ("invalid command code" on the first path char) — §11.4.67. The script already
# keeps its own ${MUTATION_BACKUP} + restores it, so use the tmpfile pattern.
sed 's/func (c \*Client) Ping(ctx context.Context) error {/func (c *Client) Ping(ctx context.Context) error { return fmt.Errorf("MUTATION: deliberately broken")/' "${MUTATION_TARGET}" > "${MUTATION_TARGET}.tmp" && mv "${MUTATION_TARGET}.tmp" "${MUTATION_TARGET}"

# Run the test that should now fail
MUTATION_FAILED=false
if go test ./test/unit/... -run TestClient_Ping -count=1 -timeout=30s >/dev/null 2>&1; then
    echo "FAIL: Mutation test did NOT fail — tests are bluff (pass when code is broken)"
    MUTATION_FAILED=true
else
    echo "PASS: Mutation test correctly FAILED after breaking Ping"
fi

# Restore original
cp "${MUTATION_BACKUP}" "${MUTATION_TARGET}"
rm -f "${MUTATION_BACKUP}"

if [ "${MUTATION_FAILED}" = "true" ]; then
    exit 1
fi

# -----------------------------------------------------------------------------
# 5. VERIFIER REGISTRY MUTATION — Break registry, verify test fails
# -----------------------------------------------------------------------------
echo ">>> Step 5: Registry mutation test..."
REGISTRY_TARGET="${PROJECT_ROOT}/pkg/models/verifier_registry.go"
REGISTRY_BACKUP="${REGISTRY_TARGET}.backup"
cp "${REGISTRY_TARGET}" "${REGISTRY_BACKUP}"

# Break IsModelVerified to always return false (portable sed — see Step 4 note, §11.4.67)
sed 's/return m.VerificationStatus == "verified" && m.CanSeeCode && m.AffirmativeResponse/return false \/\/ MUTATION: always false/' "${REGISTRY_TARGET}" > "${REGISTRY_TARGET}.tmp" && mv "${REGISTRY_TARGET}.tmp" "${REGISTRY_TARGET}"

REGISTRY_MUTATION_FAILED=false
if go test ./pkg/models/... -run TestVerifierRegistry -count=1 -timeout=30s >/dev/null 2>&1; then
    echo "FAIL: Registry mutation did NOT fail — tests are bluff"
    REGISTRY_MUTATION_FAILED=true
else
    echo "PASS: Registry mutation correctly FAILED after breaking IsModelVerified"
fi

cp "${REGISTRY_BACKUP}" "${REGISTRY_TARGET}"
rm -f "${REGISTRY_BACKUP}"

if [ "${REGISTRY_MUTATION_FAILED}" = "true" ]; then
    exit 1
fi

# -----------------------------------------------------------------------------
# 6. API HANDLER EXECUTION — REST endpoints must serve real data
# -----------------------------------------------------------------------------
echo ">>> Step 6: API handler execution test..."
if ! go test ./pkg/api/... -run TestVerifiedModels -count=1 -timeout=30s >/dev/null 2>&1; then
    echo "FAIL: API handler tests failed"
    exit 1
fi
echo "PASS: API handler tests execute and pass"

# -----------------------------------------------------------------------------
# 7. CHALLENGE INTEGRITY — All 14 challenges must be executable
# -----------------------------------------------------------------------------
echo ">>> Step 7: Challenge script integrity..."
CHALLENGE_COUNT=0
# §11.4.1 fix: this project's executable challenge scripts live in this very
# directory (pkg/challenge_runner/scripts/, == SCRIPT_DIR), not in a
# challenges/scripts/ subdir (the old path matched 0 here and FAIL-bluffed even
# though 11+ real challenges exist). Count the project's actual challenge home.
for script in "${SCRIPT_DIR}"/*_challenge.sh; do
    if [ -x "$script" ]; then
        CHALLENGE_COUNT=$((CHALLENGE_COUNT + 1))
    fi
done

if [ "${CHALLENGE_COUNT}" -lt 10 ]; then
    echo "FAIL: Only ${CHALLENGE_COUNT} executable challenges found, expected 10+"
    exit 1
fi
echo "PASS: ${CHALLENGE_COUNT} challenges are present and executable"

# -----------------------------------------------------------------------------
echo ""
echo "========================================"
echo "Anti-Bluff Execution Challenge: PASS"
echo "========================================"
echo "Guarantees established:"
echo "  - Code compiles (build verification)"
echo "  - Unit tests execute and pass"
echo "  - Integration tests execute and pass"
echo "  - Mutation test 1: breaking Ping → test FAILS"
echo "  - Mutation test 2: breaking registry → test FAILS"
echo "  - API handlers serve real data"
echo "  - All challenges are executable"
echo ""
echo "CONST-035 enforced: green tests = usable features"
exit 0
