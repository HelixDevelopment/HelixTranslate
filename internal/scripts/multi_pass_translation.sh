#!/bin/bash

# Multi-Pass Translation Pipeline
# This script performs a comprehensive 3-pass translation process:
# 1. Initial translation with multiple LLM providers
# 2. Verification and quality checking
# 3. Polishing and refinement

set -e  # Exit on error

# Configuration - API keys MUST be set via environment variables
# Example: export DEEPSEEK_API_KEY="your-key-here"
#          export ZHIPU_API_KEY="your-key-here"
if [ -z "$DEEPSEEK_API_KEY" ]; then
    echo "ERROR: DEEPSEEK_API_KEY environment variable is not set"
    echo "Please set it with: export DEEPSEEK_API_KEY=\"your-api-key\""
    exit 1
fi

if [ -z "$ZHIPU_API_KEY" ]; then
    echo "ERROR: ZHIPU_API_KEY environment variable is not set"
    echo "Please set it with: export ZHIPU_API_KEY=\"your-api-key\""
    exit 1
fi

BOOKS_DIR="Books"
OUTPUT_DIR="Books/Translated"
LOGS_DIR="/tmp/translation_logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Create directories
mkdir -p "$OUTPUT_DIR"
mkdir -p "$LOGS_DIR"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verify book with Go script
verify_book() {
    local book_path="$1"
    local book_name=$(basename "$book_path" .epub)

    log_info "Verifying: $book_name"

    go run /tmp/verify_translation.go "$book_path" 2>&1 | tee "$LOGS_DIR/${book_name}_verification.log"

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "Verification passed: $book_name"
        return 0
    else
        log_error "Verification failed: $book_name"
        return 1
    fi
}

# Pass 1: Initial Translation with Multi-LLM
pass1_initial_translation() {
    local input_file="$1"
    local basename=$(basename "$input_file" .epub)
    # Use shorter filename to avoid length issues
    local short_name="${basename:0:30}"
    local output_file="$OUTPUT_DIR/${short_name}_p1.epub"
    local log_file="$LOGS_DIR/${basename}_pass1.log"

    log_info "===== PASS 1: Initial Translation ====="
    log_info "Input: $input_file"
    log_info "Output: $output_file"
    log_info "Provider: multi-llm (DeepSeek + Zhipu AI)"

    ./build/translator \
        -input "$input_file" \
        -output "$output_file" \
        -locale sr \
        -provider multi-llm \
        -format epub \
        -script cyrillic \
        2>&1 | tee "$log_file"

    if [ ${PIPESTATUS[0]} -eq 0 ] && [ -f "$output_file" ]; then
        log_success "Pass 1 completed successfully"
        verify_book "$output_file"
        echo "$output_file"
        return 0
    else
        log_error "Pass 1 failed"
        return 1
    fi
}

# Pass 2: Verification and Quality Check
pass2_verification() {
    local input_file="$1"
    local basename=$(basename "$input_file" _p1.epub)
    # Use shorter filename to avoid length issues
    local short_name="${basename:0:30}"
    local output_file="$OUTPUT_DIR/${short_name}_p2.epub"
    local log_file="$LOGS_DIR/${basename}_pass2.log"

    log_info "===== PASS 2: Verification & Quality Check ====="
    log_info "Input: $input_file"
    log_info "Output: $output_file"
    log_info "Provider: deepseek (quality verification)"

    # Use DeepSeek for verification pass - it's good at catching errors
    ./build/translator \
        -input "$input_file" \
        -output "$output_file" \
        -locale sr \
        -provider deepseek \
        -format epub \
        -script cyrillic \
        2>&1 | tee "$log_file"

    if [ ${PIPESTATUS[0]} -eq 0 ] && [ -f "$output_file" ]; then
        log_success "Pass 2 completed successfully"
        verify_book "$output_file"
        echo "$output_file"
        return 0
    else
        log_error "Pass 2 failed"
        return 1
    fi
}

# Pass 3: Polishing and Refinement
pass3_polishing() {
    local input_file="$1"
    local basename=$(basename "$input_file" _p2.epub)
    # Use shorter filename to avoid length issues
    local short_name="${basename:0:30}"
    local output_file="$OUTPUT_DIR/${short_name}_p3_final.epub"
    local log_file="$LOGS_DIR/${basename}_pass3.log"

    log_info "===== PASS 3: Polishing & Refinement ====="
    log_info "Input: $input_file"
    log_info "Output: $output_file"
    log_info "Provider: zhipu (literary polishing)"

    # Use Zhipu for final polishing - it excels at literary quality
    ./build/translator \
        -input "$input_file" \
        -output "$output_file" \
        -locale sr \
        -provider zhipu \
        -format epub \
        -script cyrillic \
        2>&1 | tee "$log_file"

    if [ ${PIPESTATUS[0]} -eq 0 ] && [ -f "$output_file" ]; then
        log_success "Pass 3 completed successfully"
        verify_book "$output_file"
        echo "$output_file"
        return 0
    else
        log_error "Pass 3 failed"
        return 1
    fi
}

# Main translation pipeline
translate_book() {
    local input_file="$1"
    local book_name=$(basename "$input_file")

    log_info "========================================"
    log_info "Starting multi-pass translation pipeline"
    log_info "Book: $book_name"
    log_info "Timestamp: $TIMESTAMP"
    log_info "========================================"

    # Pass 1: Initial Translation
    pass1_output=$(pass1_initial_translation "$input_file")
    if [ $? -ne 0 ]; then
        log_error "Pipeline aborted: Pass 1 failed"
        return 1
    fi

    # Pass 2: Verification
    pass2_output=$(pass2_verification "$pass1_output")
    if [ $? -ne 0 ]; then
        log_warning "Pass 2 failed, using Pass 1 output"
        pass2_output="$pass1_output"
    fi

    # Pass 3: Polishing
    pass3_output=$(pass3_polishing "$pass2_output")
    if [ $? -ne 0 ]; then
        log_warning "Pass 3 failed, using Pass 2 output"
        pass3_output="$pass2_output"
    fi

    log_success "========================================"
    log_success "Translation pipeline completed!"
    log_success "Final output: $pass3_output"
    log_success "All logs saved to: $LOGS_DIR"
    log_success "========================================"

    return 0
}

# Process all books
process_all_books() {
    log_info "Searching for books in: $BOOKS_DIR"

    # Find all EPUB files that aren't already translated
    local books=$(find "$BOOKS_DIR" -maxdepth 1 -type f -name "*.epub" ! -name "*_sr_*" ! -name "*ranslat*")

    if [ -z "$books" ]; then
        log_warning "No books found to translate"
        return 0
    fi

    local count=0
    while IFS= read -r book; do
        count=$((count + 1))
        log_info "Processing book $count: $book"
        translate_book "$book"
    done <<< "$books"

    log_success "Processed $count book(s)"
}

# Main execution
if [ $# -eq 0 ]; then
    # No arguments - process all books
    process_all_books
elif [ -f "$1" ]; then
    # Single file provided
    translate_book "$1"
else
    log_error "File not found: $1"
    echo "Usage: $0 [book.epub]"
    echo "  No arguments: Process all books in $BOOKS_DIR"
    echo "  With argument: Process single book"
    exit 1
fi
