#!/bin/bash
set -euo pipefail

echo "=== API Smoke Test Challenge ==="

API_URL="${API_URL:-http://localhost:8080}"
TIMEOUT=5

echo "Testing API at $API_URL"

# Test health endpoint
echo "Step 1: Health endpoint"
curl -sf --max-time $TIMEOUT "$API_URL/health" > /dev/null
echo "PASS: Health endpoint"

# Test languages endpoint
echo "Step 2: Languages endpoint"
curl -sf --max-time $TIMEOUT "$API_URL/api/languages" > /dev/null
echo "PASS: Languages endpoint"

# Test stats endpoint
echo "Step 3: Stats endpoint"
curl -sf --max-time $TIMEOUT "$API_URL/api/stats" > /dev/null
echo "PASS: Stats endpoint"

echo "=== API Smoke Test Complete ==="
