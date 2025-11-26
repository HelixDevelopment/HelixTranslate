#!/bin/bash

# Simple Direct Translation Script using DeepSeek
# High-quality translation in a single pass

set -e

# Configuration - API key MUST be set via environment variable
# Example: export DEEPSEEK_API_KEY="your-key-here"
if [ -z "$DEEPSEEK_API_KEY" ]; then
    echo "ERROR: DEEPSEEK_API_KEY environment variable is not set"
    echo "Please set it with: export DEEPSEEK_API_KEY=\"your-api-key\""
    exit 1
fi

OUTPUT_DIR="Books/Translated"
LOG_DIR="/tmp/translation_logs"

mkdir -p "$OUTPUT_DIR"
mkdir -p "$LOG_DIR"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

translate_book() {
    local input="$1"
    local basename=$(basename "$input" .epub)
    local output="$OUTPUT_DIR/${basename}_sr_cyrillic.epub"
    local log="$LOG_DIR/${basename}_deepseek.log"

    log_info "Translating: $basename"
    log_info "Output: $output"

    if ./build/translator \
        -input "$input" \
        -output "$output" \
        -locale sr \
        -provider deepseek \
        -format epub \
        -script cyrillic \
        2>&1 | tee "$log"; then

        log_success "Translation completed: $basename"

        # Verify output
        if [ -f "$output" ]; then
            local size=$(ls -lh "$output" | awk '{print $5}')
            log_success "Output file created: $size"
            return 0
        else
            log_error "Output file not found: $output"
            return 1
        fi
    else
        log_error "Translation failed: $basename"
        return 1
    fi
}

# Find all source books
books=$(find Books -maxdepth 1 -type f -name "*.epub" ! -name "*_sr_*" ! -name "*ranslat*")

if [ -z "$books" ]; then
    log_error "No books found to translate"
    exit 1
fi

count=0
success=0
failed=0

for book in $books; do
    count=$((count + 1))
    log_info "========== Book $count =========="

    if translate_book "$book"; then
        success=$((success + 1))
    else
        failed=$((failed + 1))
    fi

    log_info "================================"
done

log_info "Translation Summary:"
log_success "Successful: $success"
[ $failed -gt 0 ] && log_error "Failed: $failed" || log_success "Failed: 0"
log_info "Logs saved to: $LOG_DIR"
log_info "Translations saved to: $OUTPUT_DIR"
