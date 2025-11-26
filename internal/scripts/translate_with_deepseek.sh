#!/bin/bash

# Efficient translation script using DeepSeek only
# Avoids network overhead from Qwen/Zhipu connection issues

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <input_file> [output_file]"
    echo ""
    echo "Example:"
    echo "  $0 Books/book.epub"
    echo "  $0 Books/book.epub output_serbian.epub"
    exit 1
fi

INPUT="$1"
OUTPUT="${2:-}"

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file not found: $INPUT"
    exit 1
fi

# Check if DEEPSEEK_API_KEY is set
if [ -z "$DEEPSEEK_API_KEY" ]; then
    echo "Error: DEEPSEEK_API_KEY environment variable not set"
    echo ""
    echo "Please set it first:"
    echo "  export DEEPSEEK_API_KEY=\"your-api-key\""
    exit 1
fi

echo "=== DeepSeek Translation ==="
echo "Input: $INPUT"
echo "Provider: DeepSeek (reliable, no network issues)"
echo "Target: Serbian (Cyrillic)"
echo "Timeout: 180 seconds per request"
echo ""

# Build output filename if not provided
if [ -z "$OUTPUT" ]; then
    BASENAME=$(basename "$INPUT" .epub)
    OUTPUT="Books/${BASENAME}_sr_deepseek.epub"
    echo "Output: $OUTPUT (auto-generated)"
else
    echo "Output: $OUTPUT"
fi

echo ""
echo "Starting translation..."
echo ""

# Run translation
./build/translator \
    -input "$INPUT" \
    -output "$OUTPUT" \
    -provider deepseek \
    -locale sr \
    -format epub \
    -script cyrillic

echo ""
echo "=== Translation Complete ==="

if [ -f "$OUTPUT" ]; then
    SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
    echo "✅ Output file created: $OUTPUT ($SIZE)"
    echo ""
    echo "To verify quality, check for Russian text:"
    echo "  unzip -p \"$OUTPUT\" | grep -o '[А-Яа-я]\\{20,\\}' | head -10"
else
    echo "❌ Output file not found - translation may have failed"
    echo "Check the error messages above"
fi
