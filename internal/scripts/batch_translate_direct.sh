#!/bin/bash
# batch_translate_direct.sh - Direct batch translation using worker API

set -e

# Configuration
WORKER_HOST="${WORKER_HOST:-thinker.local}"
WORKER_PORT="${WORKER_PORT:-8443}"
BOOKS_DIR="${BOOKS_DIR:-./Books}"
OUTPUT_DIR="${OUTPUT_DIR:-./Books/translated}"
LOG_FILE="${LOG_FILE:-batch_direct_translation.log}"
API_LOG="${API_LOG:-workers_api_communication.log}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1${NC}" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARN: $1${NC}" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}" | tee -a "$LOG_FILE"
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."

    # Check if books directory exists
    if [[ ! -d "$BOOKS_DIR" ]]; then
        log_error "Books directory not found: $BOOKS_DIR"
        exit 1
    fi

    # Check if there are books to translate
    local book_count
    book_count=$(find "$BOOKS_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.txt" \) | wc -l)
    if [[ $book_count -eq 0 ]]; then
        log_error "No books found in $BOOKS_DIR"
        exit 1
    fi

    log_info "Found $book_count books to translate"

    # Check worker connectivity
    if ! curl -k -f "https://$WORKER_HOST:$WORKER_PORT/health" >/dev/null 2>&1; then
        log_error "Worker at $WORKER_HOST:$WORKER_PORT is not responding"
        exit 1
    fi

    log_success "Prerequisites validated"
}

# Get list of books to translate
get_books_list() {
    # Find all supported book formats, excluding already translated files
    find "$BOOKS_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.txt" \) ! -name "*_sr*" -print0
}

# Translate a single book
translate_book() {
    local book_path="$1"
    local book_name
    book_name=$(basename "$book_path")
    local book_basename="${book_name%.*}"

    log_info "Translating book: $book_name"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Prepare output path
    local output_path="$OUTPUT_DIR/${book_basename}_sr.fb2"

    # Log API request
    echo "{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"source_host\":\"localhost\",\"source_port\":0,\"target_host\":\"$WORKER_HOST\",\"target_port\":$WORKER_PORT,\"method\":\"POST\",\"url\":\"https://$WORKER_HOST:$WORKER_PORT/api/v1/translate/fb2\",\"request_size\":$(stat -f%z "$book_path" 2>/dev/null || stat -c%s "$book_path" 2>/dev/null || echo 0)}" >> "$API_LOG"

    # Make API call to translate
    local start_time
    start_time=$(date +%s)

    local response
    response=$(curl -k -s -w "%{http_code}" -X POST "https://$WORKER_HOST:$WORKER_PORT/api/v1/translate/fb2" \
        -F "file=@$book_path" \
        -F "target_language=sr" \
        -F "provider=ollama" \
        -o "$output_path")

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Log API response
    local response_size=0
    if [[ -f "$output_path" ]]; then
        response_size=$(stat -f%z "$output_path" 2>/dev/null || stat -c%s "$output_path" 2>/dev/null || echo 0)
    fi

    if [[ "${response: -3}" == "200" ]]; then
        echo "{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"status_code\":200,\"response_size\":$response_size,\"duration\":\"${duration}s\"}" >> "$API_LOG"
    else
        echo "{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"status_code\":${response: -3},\"error\":\"Translation failed\"}" >> "$API_LOG"
    fi

    if [[ -f "$output_path" ]] && [[ -s "$output_path" ]]; then
        log_success "Translation completed: $book_name -> ${book_basename}_sr.fb2 (${response_size} bytes, ${duration}s)"
    else
        log_error "Translation failed: $book_name (HTTP ${response: -3})"
        return 1
    fi
}

# Generate translation report
generate_report() {
    local total_books="$1"
    local successful_translations="$2"
    local failed_translations="$3"
    local total_time="$4"

    log_info "Generating translation report..."

    cat > "$OUTPUT_DIR/translation_report.txt" << EOF
Direct Worker Batch Translation Report
=====================================

Generated: $(date)
Worker: $WORKER_HOST:$WORKER_PORT
Total books processed: $total_books
Successful translations: $successful_translations
Failed translations: $failed_translations
Total time: ${total_time}s
Average time per book: $((total_time / total_books))s

API Communications Summary:
$(if [[ -f "$API_LOG" ]]; then
    echo "Total API calls: $(grep -c "timestamp" "$API_LOG")"
    echo "Status code distribution:"
    grep -o '"status_code":[0-9]*' "$API_LOG" | cut -d: -f2 | sort | uniq -c | sort -nr || echo "Unable to parse API log"
else
    echo "API log not available"
fi)

Output directory: $OUTPUT_DIR
API log: $API_LOG
Batch log: $LOG_FILE
EOF

    log_success "Report generated: $OUTPUT_DIR/translation_report.txt"
}

# Main execution
main() {
    log_info "=== Direct Worker Batch Translation Started ==="
    log_info "Worker: $WORKER_HOST:$WORKER_PORT"
    log_info "Books directory: $BOOKS_DIR"
    log_info "Output directory: $OUTPUT_DIR"

    # Validate prerequisites
    validate_prerequisites

    # Get list of books
    local books=()
    while IFS= read -r -d '' file; do
        books+=("$file")
    done < <(get_books_list)
    local total_books=${#books[@]}

    log_info "Starting translation of $total_books books..."

    # Translate all books
    local successful=0
    local failed=0
    local start_time
    start_time=$(date +%s)

    for book in "${books[@]}"; do
        if translate_book "$book"; then
            ((successful++))
        else
            ((failed++))
        fi

        # Small delay between translations to avoid overwhelming the worker
        sleep 2
    done

    local end_time
    end_time=$(date +%s)
    local total_time=$((end_time - start_time))

    # Generate report
    generate_report "$total_books" "$successful" "$failed" "$total_time"

    log_info "=== Batch Translation Completed ==="
    log_info "Total books: $total_books"
    log_info "Successful: $successful"
    log_info "Failed: $failed"
    log_info "Total time: ${total_time}s"

    if [[ $failed -eq 0 ]]; then
        log_success "All translations completed successfully! 🎉"
        exit 0
    else
        log_error "Some translations failed. Check logs for details."
        exit 1
    fi
}

# Run main function
main "$@"