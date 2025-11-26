#!/bin/bash

# Simple test script for SSH translation workflow
set -euo pipefail

INPUT_FILE="$1"
OUTPUT_FILE="$2"
CONFIG_FILE="$3"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting simple test translation"
echo "Input: $INPUT_FILE"
echo "Output: $OUTPUT_FILE"
echo "Config: $CONFIG_FILE"

# Check if input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    echo "ERROR: Input file not found: $INPUT_FILE"
    exit 1
fi

# Create a simple "translation" by replacing Russian characters with Serbian
# This is just a test to verify the SSH workflow works
sed 's/русский/srpski/g; s/Российски/Srpski/g' "$INPUT_FILE" > "$OUTPUT_FILE" || {
    echo "ERROR: Failed to process file"
    exit 1
}

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Test translation completed"
echo "Output file size: $(wc -c < "$OUTPUT_FILE") bytes"