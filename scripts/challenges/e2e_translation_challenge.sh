#!/bin/bash
set -euo pipefail

echo "=== E2E Translation Challenge ==="

# Check for test fixtures
if [ ! -d "test/fixtures" ]; then
    echo "FAIL: test/fixtures directory not found"
    exit 1
fi

echo "Step 1: Running unit tests"
go test -short ./pkg/translator/... > /dev/null 2>&1
echo "PASS: Translator unit tests"

echo "Step 2: Running format detection tests"
go test -short ./pkg/format/... > /dev/null 2>&1
echo "PASS: Format detection tests"

echo "Step 3: Running ebook parsing tests"
go test -short ./pkg/ebook/... > /dev/null 2>&1
echo "PASS: Ebook parsing tests"

echo "Step 4: Running markdown conversion tests"
go test -short ./pkg/markdown/... > /dev/null 2>&1
echo "PASS: Markdown conversion tests"

echo "Step 5: Running verification tests"
go test -short ./pkg/verification/... > /dev/null 2>&1
echo "PASS: Verification tests"

echo "=== E2E Translation Challenge Complete ==="
