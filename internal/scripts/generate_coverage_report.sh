#!/bin/bash

echo "Generating comprehensive coverage report..."

# Clean previous coverage files
rm -f coverage*.out coverage*.html coverage*.txt

echo "Running tests with coverage..."
go test ./... -coverprofile=coverage-full.out -v 2>&1 | tee coverage-test.log

echo "Generating HTML coverage report..."
go tool cover -html=coverage-full.out -o coverage-full.html

echo "Coverage by package:"
echo "========================"
go tool cover -func=coverage-full.out | grep "^pkg/" | sort -k3 -n

echo ""
echo "Low coverage packages (<80%):"
echo "==================================="
go tool cover -func=coverage-full.out | awk '$3 < "80.0%" && $1 ~ /^pkg/ {print $1, $3}'

echo ""
echo "No test coverage packages:"
echo "=========================="
find pkg/ -name "*.go" -not -name "*_test.go" | while read file; do
    pkg=$(dirname "$file")
    if ! ls "$pkg"/*_test.go >/dev/null 2>&1; then
        echo "$pkg"
    fi
done | sort -u

echo ""
echo "Overall coverage summary:"
echo "========================"
go tool cover -func=coverage-full.out | tail -1

echo ""
echo "Coverage report generated: coverage-full.html"
echo "Test log saved: coverage-test.log"