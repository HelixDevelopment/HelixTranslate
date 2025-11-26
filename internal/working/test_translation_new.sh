#!/bin/bash

# Simple test of translation using SSH worker directly
INPUT_FB2="materials/books/book1.fb2"
OUTPUT_EPUB="test_book_sr.epub"

echo "Starting translation test..."
echo "Input: $INPUT_FB2"
echo "Output: $OUTPUT_EPUB"

# Run with timeout to prevent hanging
timeout 300 ./translator-ssh-test \
  -input "$INPUT_FB2" \
  -output "$OUTPUT_EPUB" \
  -host thinker.local \
  -user milosvasic \
  -password WhiteSnake8587

echo "Translation test completed with exit code: $?"

# Check if output was created
if [ -f "$OUTPUT_EPUB" ]; then
    echo "✓ EPUB file created: $OUTPUT_EPUB"
    echo "File size: $(stat -f%z "$OUTPUT_EPUB" 2>/dev/null || stat -c%s "$OUTPUT_EPUB" 2>/dev/null || echo "unknown") bytes"
    
    # Test EPUB validity
    echo "Testing EPUB validity..."
    if command -v unzip >/dev/null 2>&1; then
        if unzip -t "$OUTPUT_EPUB" >/dev/null 2>&1; then
            echo "✓ EPUB file is valid"
        else
            echo "✗ EPUB file is invalid"
        fi
    fi
else
    echo "✗ EPUB file was not created"
fi