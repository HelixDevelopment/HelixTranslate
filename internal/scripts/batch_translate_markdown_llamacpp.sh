#!/bin/bash

################################################################################
# Markdown-Based Batch Translation Script with llamacpp
#
# This script implements the complete multi-stage workflow:
# 1. EPUB → Markdown
# 2. Preparation Phase (Multi-LLM Analysis)
# 3. Translation (with Preparation guidance)
# 4. Markdown → EPUB
#
# CRITICAL: Only 1 LLM instance runs at a time to prevent system freeze!
#
# Usage:
#   ./batch_translate_markdown_llamacpp.sh <input_dir> <output_dir> [--prepare] [--prep-passes N]
#
# Examples:
#   # Basic translation (no preparation)
#   ./batch_translate_markdown_llamacpp.sh Books/Source/ Books/Translated/
#
#   # With 2-pass preparation
#   ./batch_translate_markdown_llamacpp.sh Books/Source/ Books/Translated/ --prepare --prep-passes 2
#
#   # With 3-pass preparation
#   ./batch_translate_markdown_llamacpp.sh Books/Source/ Books/Translated/ --prepare --prep-passes 3
################################################################################

set -e  # Exit on error

# Default configuration
INPUT_DIR="${1:-Books/Source}"
OUTPUT_DIR="${2:-Books/Translated}"
ENABLE_PREP=false
PREP_PASSES=2
TARGET_LANG="sr"
SCRIPT_TYPE="cyrillic"
PROVIDER="llamacpp"
LOG_DIR="logs/markdown_workflow"

# Parse optional arguments
shift 2 2>/dev/null || true
while [[ $# -gt 0 ]]; do
    case $1 in
        --prepare)
            ENABLE_PREP=true
            shift
            ;;
        --prep-passes)
            PREP_PASSES="$2"
            shift 2
            ;;
        --provider)
            PROVIDER="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Create directories
mkdir -p "$OUTPUT_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "Books/Images"

# Function to log with timestamp
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check system resources
check_resources() {
    log "${BLUE}Checking system resources...${NC}"

    # Check if llama-cli is running (for llamacpp provider)
    if [ "$PROVIDER" = "llamacpp" ]; then
        if pgrep -f "llama-cli" > /dev/null; then
            log "${RED}ERROR: llama-cli is already running!${NC}"
            log "${RED}Only 1 instance is safe on this system.${NC}"
            log "${RED}Please wait for the current process to finish.${NC}"
            exit 1
        fi
    fi

    # Check available RAM
    AVAILABLE_RAM=$(vm_stat | awk '/Pages free/ {free=$3} /Pages inactive/ {inactive=$3} END {print (free+inactive)*4096/1024/1024/1024}')
    log "${GREEN}Available RAM: ${AVAILABLE_RAM} GB${NC}"

    if (( $(echo "$AVAILABLE_RAM < 8" | bc -l) )); then
        log "${YELLOW}WARNING: Low RAM (< 8GB available)${NC}"
        if [ "$PROVIDER" = "llamacpp" ]; then
            log "${YELLOW}Translation may be slow or may fail${NC}"
        fi
    fi
}

# Function to translate a single book
translate_book() {
    local input_file="$1"
    local filename=$(basename "$input_file")
    local basename="${filename%.*}"
    local output_epub="${OUTPUT_DIR}/${basename}_SR_Cyrillic.epub"
    local log_file="${LOG_DIR}/${basename}_workflow.log"

    log "${BLUE}========================================${NC}"
    log "${BLUE}Processing: ${filename}${NC}"
    log "${BLUE}========================================${NC}"

    # Check if already translated
    if [ -f "$output_epub" ]; then
        log "${YELLOW}SKIP: Already translated - ${output_epub}${NC}"
        return 0
    fi

    # Build command
    CMD="./markdown-translator -input \"$input_file\" -output \"$output_epub\" -lang \"$TARGET_LANG\" -provider \"$PROVIDER\" -keep-md=true"

    # Add preparation flags if enabled
    if [ "$ENABLE_PREP" = true ]; then
        CMD="$CMD -prepare -prep-passes $PREP_PASSES"
        log "${CYAN}🔍 Preparation: ENABLED ($PREP_PASSES passes)${NC}"
    else
        log "${CYAN}🔍 Preparation: DISABLED${NC}"
    fi

    # Log the command
    echo "Command: $CMD" > "$log_file"
    echo "Started: $(date)" >> "$log_file"
    log "Provider: $PROVIDER"
    log "Target: $TARGET_LANG ($SCRIPT_TYPE)"
    log "Log file: $log_file"

    # Start translation with timestamp
    START_TIME=$(date +%s)
    log "${GREEN}Started at: $(date '+%Y-%m-%d %H:%M:%S')${NC}"

    # Show workflow stages
    log ""
    log "${CYAN}Workflow Stages:${NC}"
    log "  1️⃣  EPUB → Markdown"
    if [ "$ENABLE_PREP" = true ]; then
        log "  2️⃣  Preparation Phase ($PREP_PASSES passes)"
        log "  3️⃣  Markdown Translation"
        log "  4️⃣  Markdown → EPUB"
    else
        log "  2️⃣  Markdown Translation"
        log "  3️⃣  Markdown → EPUB"
    fi
    log ""

    # Run translation and capture output
    if eval "$CMD 2>&1 | tee -a \"$log_file\""; then
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        HOURS=$((DURATION / 3600))
        MINUTES=$(((DURATION % 3600) / 60))
        SECONDS=$((DURATION % 60))

        log ""
        log "${GREEN}✅ SUCCESS: Translation completed${NC}"
        log "${GREEN}Duration: ${HOURS}h ${MINUTES}m ${SECONDS}s${NC}"
        log "${GREEN}Output: ${output_epub}${NC}"

        # Show generated files
        log ""
        log "${CYAN}Generated files:${NC}"
        local md_source="Books/${basename}_source.md"
        local md_translated="Books/${basename}_translated.md"
        local prep_json="Books/${basename}_preparation.json"

        if [ -f "$md_source" ]; then
            local size=$(du -h "$md_source" | cut -f1)
            log "  📄 Source MD:      $md_source ($size)"
        fi

        if [ "$ENABLE_PREP" = true ] && [ -f "$prep_json" ]; then
            local size=$(du -h "$prep_json" | cut -f1)
            log "  🔍 Preparation:    $prep_json ($size)"
        fi

        if [ -f "$md_translated" ]; then
            local size=$(du -h "$md_translated" | cut -f1)
            log "  📄 Translated MD:  $md_translated ($size)"
        fi

        if [ -f "$output_epub" ]; then
            local size=$(du -h "$output_epub" | cut -f1)
            log "  📚 Final EPUB:     $output_epub ($size)"
        fi

        # Add to completed list
        echo "$filename,$output_epub,$DURATION" >> "${LOG_DIR}/completed_translations.csv"

        return 0
    else
        log "${RED}❌ ERROR: Translation failed${NC}"
        log "${RED}Check log file: ${log_file}${NC}"

        # Add to failed list
        echo "$filename,$input_file,failed" >> "${LOG_DIR}/failed_translations.csv"

        return 1
    fi
}

# Main script
main() {
    log "${BLUE}========================================${NC}"
    log "${BLUE}Markdown-Based Batch Translation${NC}"
    log "${BLUE}Multi-Stage Workflow with llamacpp${NC}"
    log "${BLUE}========================================${NC}"
    log "Input directory:  $INPUT_DIR"
    log "Output directory: $OUTPUT_DIR"
    log "Provider:         $PROVIDER"
    log "Target language:  $TARGET_LANG"
    log "Script type:      $SCRIPT_TYPE"
    log "Preparation:      $([ "$ENABLE_PREP" = true ] && echo "ENABLED ($PREP_PASSES passes)" || echo "DISABLED")"
    log "${BLUE}========================================${NC}"
    echo ""

    # Check resources
    check_resources
    echo ""

    # Find all books
    log "${BLUE}Searching for EPUB books...${NC}"

    BOOKS=()
    while IFS= read -r -d '' file; do
        BOOKS+=("$file")
    done < <(find "$INPUT_DIR" -type f -name "*.epub" -print0)

    TOTAL_BOOKS=${#BOOKS[@]}

    if [ "$TOTAL_BOOKS" -eq 0 ]; then
        log "${RED}No EPUB books found in $INPUT_DIR${NC}"
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
            echo ""
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

        # Pause between books
        if [ $BOOK_NUM -lt $TOTAL_BOOKS ]; then
            log "${CYAN}Pausing 5 seconds before next book...${NC}"
            sleep 5
        fi
    done

    # Final summary
    log "${BLUE}========================================${NC}"
    log "${BLUE}Batch Translation Complete${NC}"
    log "${BLUE}========================================${NC}"
    log "${GREEN}Successful:  ${SUCCESS_COUNT}${NC}"
    log "${YELLOW}Skipped:     ${SKIP_COUNT}${NC}"
    log "${RED}Failed:      ${FAIL_COUNT}${NC}"
    log "Total books: ${TOTAL_BOOKS}"
    log "${BLUE}========================================${NC}"

    # Show completed translations
    if [ "$SUCCESS_COUNT" -gt 0 ]; then
        log ""
        log "${GREEN}Completed translations:${NC}"
        tail -n +2 "${LOG_DIR}/completed_translations.csv" | while IFS=, read -r filename output duration; do
            HOURS=$((duration / 3600))
            MINUTES=$(((duration % 3600) / 60))
            log "  ✅ $filename"
            log "     → $output"
            log "     ⏱  ${HOURS}h ${MINUTES}m"
        done
    fi

    # Show failed translations
    if [ "$FAIL_COUNT" -gt 0 ]; then
        log ""
        log "${RED}Failed translations:${NC}"
        tail -n +2 "${LOG_DIR}/failed_translations.csv" | while IFS=, read -r filename input status; do
            log "  ❌ $filename"
        done
    fi

    log ""
    log "${BLUE}Logs saved to: ${LOG_DIR}/${NC}"
    log ""

    # Show workflow artifacts
    log "${CYAN}Workflow artifacts in Books/:${NC}"
    log "  - *_source.md - Source markdown files"
    if [ "$ENABLE_PREP" = true ]; then
        log "  - *_preparation.json - Analysis results"
    fi
    log "  - *_translated.md - Translated markdown files"
    log "  - Images/ - Extracted images and covers"
}

# Run main function
main
