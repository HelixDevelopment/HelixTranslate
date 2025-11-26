#!/bin/bash

# Final working translation test
INPUT_FB2="internal/materials/books/book1.fb2"
OUTPUT_EPUB="internal/working/book1_sr.epub"

echo "=== FINAL TRANSLATION TEST ==="
echo "Input: $INPUT_FB2"
echo "Output: $OUTPUT_EPUB"
echo "Starting at: $(date)"
echo

# Build the binary
echo "Building translation binary..."
go build -o internal/scripts/translator-final ./cmd/translate-ssh

if [ $? -ne 0 ]; then
    echo "✗ Build failed"
    exit 1
fi

echo "✓ Build successful"
echo

# Run translation
echo "Starting translation..."
timeout 600 ./internal/scripts/translator-final \
  -input "$INPUT_FB2" \
  -output "$OUTPUT_EPUB" \
  -host thinker.local \
  -user milosvasic \
  -password WhiteSnake8587

EXIT_CODE=$?
echo
echo "Translation completed with exit code: $EXIT_CODE"

# Check results
if [ -f "$OUTPUT_EPUB" ]; then
    echo "✓ EPUB file created: $OUTPUT_EPUB"
    echo "File size: $(stat -f%z "$OUTPUT_EPUB" 2>/dev/null || stat -c%s "$OUTPUT_EPUB" 2>/dev/null || echo "unknown") bytes"
    
    # Test EPUB validity
    echo "Testing EPUB validity..."
    if command -v unzip >/dev/null 2>&1; then
        if unzip -t "$OUTPUT_EPUB" >/dev/null 2>&1; then
            echo "✓ EPUB file is valid"
            
            # Check content
            echo "Checking translation quality..."
            if unzip -p "$OUTPUT_EPUB" "OEBPS/*.xhtml" 2>/dev/null | head -20 | grep -q "sr\|ср\|ћ\|љ\|њ"; then
                echo "✓ Serbian Cyrillic characters detected"
            else
                echo "? Serbian Cyrillic characters not clearly detected"
            fi
        else
            echo "✗ EPUB file is invalid"
        fi
    fi
    
    echo "✓ Translation test completed successfully"
else
    echo "✗ EPUB file was not created"
fi

echo "Completed at: $(date)"
echo "=== TEST COMPLETE ==="