#!/bin/bash

################################################################################
# Safe Batch Translation Script for llama.cpp
#
# This script translates multiple books sequentially (NOT in parallel) to
# prevent system freeze due to excessive RAM usage.
#
# CRITICAL: Only 1 LLM instance runs at a time!
#
# Usage:
#   ./batch_translate_llamacpp.sh <input_dir> <output_dir> [model]
#
# Example:
#   ./batch_translate_llamacpp.sh Books/ Books/Translated_Llamacpp/
#   ./batch_translate_llamacpp.sh Books/ Books/Translated_Llamacpp/ hunyuan-mt-7b-q8
################################################################################

set -e  # Exit on error

# Configuration
INPUT_DIR="${1:-Books}"
OUTPUT_DIR="${2:-Books/Translated_Llamacpp}"
MODEL="${3:-}"  # Optional: specify model, otherwise auto-select
TARGET_LOCALE="sr"
TARGET_SCRIPT="cyrillic"
LOG_DIR="logs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create directories
mkdir -p "$OUTPUT_DIR"
mkdir -p "$LOG_DIR"

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check system resources
check_resources() {
    log "${BLUE}Checking system resources...${NC}"

    # Check if llama-cli is running
    if pgrep -f "llama-cli" > /dev/null; then
        log "${RED}ERROR: llama-cli is already running!${NC}"
        log "${RED}Only 1 instance is safe on this system. Please wait for it to finish.${NC}"
        exit 1
    fi

    # Check available RAM
    AVAILABLE_RAM=$(vm_stat | awk '/Pages free/ {free=$3} /Pages inactive/ {inactive=$3} END {print (free+inactive)*4096/1024/1024/1024}')
    log "${GREEN}Available RAM: ${AVAILABLE_RAM} GB${NC}"

    if (( $(echo "$AVAILABLE_RAM < 8" | bc -l) )); then
        log "${YELLOW}WARNING: Low RAM (< 8GB available). Translation may be slow.${NC}"
    fi
}

# Function to translate a single book
translate_book() {
    local input_file="$1"
    local filename=$(basename "$input_file")
    local basename="${filename%.*}"
    local output_file="${OUTPUT_DIR}/${basename}_SR_Cyrillic.epub"
    local log_file="${LOG_DIR}/${basename}_translation.log"

    log "${BLUE}========================================${NC}"
    log "${BLUE}Translating: ${filename}${NC}"
    log "${BLUE}========================================${NC}"

    # Check if already translated
    if [ -f "$output_file" ]; then
        log "${YELLOW}SKIP: Already translated - ${output_file}${NC}"
        return 0
    fi

    # Build command
    CMD="./cli -input \"$input_file\" -output \"$output_file\" -locale \"$TARGET_LOCALE\" -script \"$TARGET_SCRIPT\" -provider llamacpp -format epub"

    # Add model if specified
    if [ -n "$MODEL" ]; then
        CMD="$CMD -model \"$MODEL\""
    fi

    # Log the command
    echo "$CMD" > "$log_file"
    log "Command: $CMD"
    log "Log file: $log_file"

    # Start translation with timestamp
    START_TIME=$(date +%s)
    log "${GREEN}Started at: $(date '+%Y-%m-%d %H:%M:%S')${NC}"

    # Run translation and capture output
    if eval "$CMD 2>&1 | tee -a \"$log_file\""; then
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        HOURS=$((DURATION / 3600))
        MINUTES=$(((DURATION % 3600) / 60))
        SECONDS=$((DURATION % 60))

        log "${GREEN}SUCCESS: Translation completed${NC}"
        log "${GREEN}Duration: ${HOURS}h ${MINUTES}m ${SECONDS}s${NC}"
        log "${GREEN}Output: ${output_file}${NC}"

        # Add to completed list
        echo "$filename,$output_file,$DURATION" >> "${LOG_DIR}/completed_translations.csv"

        return 0
    else
        log "${RED}ERROR: Translation failed${NC}"
        log "${RED}Check log file: ${log_file}${NC}"

        # Add to failed list
        echo "$filename,$input_file,failed" >> "${LOG_DIR}/failed_translations.csv"

        return 1
    fi
}

# Main script
main() {
    log "${BLUE}========================================${NC}"
    log "${BLUE}llama.cpp Batch Translation Script${NC}"
    log "${BLUE}========================================${NC}"
    log "Input directory: $INPUT_DIR"
    log "Output directory: $OUTPUT_DIR"
    log "Target locale: $TARGET_LOCALE"
    log "Target script: $TARGET_SCRIPT"
    if [ -n "$MODEL" ]; then
        log "Model: $MODEL"
    else
        log "Model: Auto-select"
    fi
    log "${BLUE}========================================${NC}"
    echo ""

    # Check resources
    check_resources
    echo ""

    # Find all books
    log "${BLUE}Searching for books...${NC}"

    # Find EPUB and FB2 files
    BOOKS=()
    while IFS= read -r -d '' file; do
        BOOKS+=("$file")
    done < <(find "$INPUT_DIR" -type f \( -name "*.epub" -o -name "*.fb2" -o -name "*.b2" \) -print0)

    TOTAL_BOOKS=${#BOOKS[@]}

    if [ "$TOTAL_BOOKS" -eq 0 ]; then
        log "${RED}No books found in $INPUT_DIR${NC}"
        exit 1
    fi

    log "${GREEN}Found ${TOTAL_BOOKS} book(s) to translate${NC}"
    echo ""

    # Initialize CSV files
    echo "filename,output_file,duration_seconds" > "${LOG_DIR}/completed_translations.csv"
    echo "filename,input_file,status" > "${LOG_DIR}/failed_translations.csv"

    # Translate each book sequentially
    SUCCESS_COUNT=0
    FAIL_COUNT=0
    SKIP_COUNT=0

    for i in "${!BOOKS[@]}"; do
        BOOK_NUM=$((i + 1))
        BOOK_FILE="${BOOKS[$i]}"

        log "${BLUE}========================================${NC}"
        log "${BLUE}Book ${BOOK_NUM}/${TOTAL_BOOKS}${NC}"
        log "${BLUE}========================================${NC}"

        # Check if already translated
        FILENAME=$(basename "$BOOK_FILE")
        BASENAME="${FILENAME%.*}"
        OUTPUT_FILE="${OUTPUT_DIR}/${BASENAME}_SR_Cyrillic.epub"

        if [ -f "$OUTPUT_FILE" ]; then
            log "${YELLOW}SKIP: Already translated${NC}"
            SKIP_COUNT=$((SKIP_COUNT + 1))
            continue
        fi

        # Translate
        if translate_book "$BOOK_FILE"; then
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            FAIL_COUNT=$((FAIL_COUNT + 1))
            log "${RED}ERROR: Failed to translate ${FILENAME}${NC}"
            log "${YELLOW}Continuing with next book...${NC}"
        fi

        echo ""

        # Show progress
        COMPLETED=$((SUCCESS_COUNT + FAIL_COUNT + SKIP_COUNT))
        PROGRESS=$((COMPLETED * 100 / TOTAL_BOOKS))
        log "${GREEN}Progress: ${COMPLETED}/${TOTAL_BOOKS} (${PROGRESS}%)${NC}"
        echo ""
    done

    # Final summary
    log "${BLUE}========================================${NC}"
    log "${BLUE}Batch Translation Complete${NC}"
    log "${BLUE}========================================${NC}"
    log "${GREEN}Successful: ${SUCCESS_COUNT}${NC}"
    log "${YELLOW}Skipped: ${SKIP_COUNT}${NC}"
    log "${RED}Failed: ${FAIL_COUNT}${NC}"
    log "Total books: ${TOTAL_BOOKS}"
    log "${BLUE}========================================${NC}"

    # Show completed translations
    if [ "$SUCCESS_COUNT" -gt 0 ]; then
        log ""
        log "${GREEN}Completed translations:${NC}"
        tail -n +2 "${LOG_DIR}/completed_translations.csv" | while IFS=, read -r filename output duration; do
            HOURS=$((duration / 3600))
            MINUTES=$(((duration % 3600) / 60))
            log "  - $filename -> $output (${HOURS}h ${MINUTES}m)"
        done
    fi

    # Show failed translations
    if [ "$FAIL_COUNT" -gt 0 ]; then
        log ""
        log "${RED}Failed translations:${NC}"
        tail -n +2 "${LOG_DIR}/failed_translations.csv" | while IFS=, read -r filename input status; do
            log "  - $filename"
        done
    fi

    log ""
    log "${BLUE}Logs saved to: ${LOG_DIR}/${NC}"
}

# Run main function
main
