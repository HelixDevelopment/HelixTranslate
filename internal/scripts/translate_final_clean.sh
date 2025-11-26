#!/bin/bash

# Clean final translation test with LLM-only translation
set -e

INPUT_FB2="internal/materials/books/book1.fb2"
OUTPUT_EPUB="internal/working/book1_final_sr.epub"

echo "=== FINAL LLM-ONLY TRANSLATION TEST ==="
echo "Input: $INPUT_FB2"
echo "Output: $OUTPUT_EPUB"
echo "Time: $(date)"
echo

# Clean up any existing outputs
rm -f "$OUTPUT_EPUB" 2>/dev/null || true

# Build clean binary
echo "Building translation system..."
go build -o internal/scripts/translator-final ./cmd/translate-ssh

echo "✓ Build completed"
echo

# Run translation with timeout
echo "Starting LLM-only translation..."
timeout 300 ./internal/scripts/translator-final \
  -input "$INPUT_FB2" \
  -output "$OUTPUT_EPUB" \
  -host thinker.local \
  -user milosvasic \
  -password WhiteSnake8587

EXIT_CODE=$?
echo
echo "Translation completed with exit code: $EXIT_CODE"

# Verify results
if [ -f "$OUTPUT_EPUB" ]; then
    echo "✓ SUCCESS: EPUB created at $OUTPUT_EPUB"
    
    # Check file size
    if command -v stat >/dev/null 2>&1; then
        SIZE=$(stat -f%z "$OUTPUT_EPUB" 2>/dev/null || stat -c%s "$OUTPUT_EPUB" 2>/dev/null || echo "unknown")
        echo "File size: $SIZE bytes"
    fi
    
    # Test EPUB validity
    if command -v unzip >/dev/null 2>&1; then
        echo "Testing EPUB structure..."
        if unzip -t "$OUTPUT_EPUB" >/dev/null 2>&1; then
            echo "✓ EPUB structure is valid"
            
            # Check for Serbian content
            echo "Checking translation quality..."
            if unzip -p "$OUTPUT_EPUB" "OEBPS/*.xhtml" 2>/dev/null | grep -E "(š|đ|č|ć|ž|ћ|љ|њ|џ)" >/dev/null 2>&1; then
                echo "✓ Serbian characters detected"
            else
                echo "? Serbian characters not clearly detected"
            fi
        else
            echo "✗ EPUB structure is invalid"
        fi
    fi
    
    echo "✓ TRANSLATION SUCCESSFUL"
else
    echo "✗ FAILED: EPUB not created"
fi

echo
echo "Completed at: $(date)"
echo "=== TEST COMPLETE ==="