#!/bin/bash
# batch_translate_distributed.sh - Batch translation using distributed workers with llama.cpp

set -e

# Configuration
MAIN_CONFIG="${MAIN_CONFIG:-config.distributed.json}"
WORKER_CONFIG="${WORKER_CONFIG:-config.worker.llamacpp.json}"
BOOKS_DIR="${BOOKS_DIR:-materials/books}"
OUTPUT_DIR="${OUTPUT_DIR:-materials/books}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-epub}"
LOG_FILE="${LOG_FILE:-batch_translation.log}"
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

# Retry a command with exponential backoff
retry_command() {
    local cmd="$1"
    local max_attempts="${2:-3}"
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        log_info "Executing (attempt $attempt/$max_attempts): $cmd"
        if eval "$cmd"; then
            return 0
        else
            log_warn "Command failed (attempt $attempt/$max_attempts)"
            if [[ $attempt -lt $max_attempts ]]; then
                local sleep_time=$((attempt * 2))
                log_info "Retrying in ${sleep_time}s..."
                sleep $sleep_time
            fi
            ((attempt++))
        fi
    done

    log_error "Command failed after $max_attempts attempts: $cmd"
    return 1
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."

    # Check if main binary exists
    if [[ ! -f "./translator-server" ]]; then
        log_error "Main server binary not found. Run 'make build' first."
        exit 1
    fi

    # Check if worker binary exists
    if [[ ! -f "./translator-server" ]]; then
        log_error "Worker binary not found. Run 'make build' first."
        exit 1
    fi

    # Check if books directory exists
    if [[ ! -d "$BOOKS_DIR" ]]; then
        log_error "Books directory not found: $BOOKS_DIR"
        exit 1
    fi

    # Check if there are books to translate
    local book_count
    book_count=$(find "$BOOKS_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.mobi" -o -name "*.pdf" -o -name "*.azw3" -o -name "*.txt" \) | wc -l)
    if [[ $book_count -eq 0 ]]; then
        log_error "No books found in $BOOKS_DIR"
        exit 1
    fi

    log_info "Found $book_count books to translate"

    # Check configurations
    if [[ ! -f "$MAIN_CONFIG" ]]; then
        log_error "Main config not found: $MAIN_CONFIG"
        exit 1
    fi

    if [[ ! -f "$WORKER_CONFIG" ]]; then
        log_error "Worker config not found: $WORKER_CONFIG"
        exit 1
    fi

    # Check certificates
    if [[ ! -f "certs/server.crt" ]] || [[ ! -f "certs/server.key" ]]; then
        log_warn "TLS certificates not found, generating..."
        make generate-certs
    fi

    log_success "Prerequisites validated"
}

# Start main server
start_main_server() {
    log_info "Starting main translation server..."

    # Kill any existing server
    pkill -f "translator-server.*$MAIN_CONFIG" || true
    sleep 2

    # Start server in background
    ./translator-server --config "$MAIN_CONFIG" > server.log 2>&1 &
    local server_pid=$!

    echo $server_pid > server.pid
    log_info "Main server started with PID: $server_pid"

    # Wait for server to be ready
    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -k "https://localhost:8443/health" >/dev/null 2>&1; then
            log_success "Main server is healthy"
            return 0
        fi

        log_info "Waiting for main server to be ready (attempt $attempt/$max_attempts)..."
        sleep 2
        ((attempt++))
    done

    log_error "Main server failed to start within timeout"
    kill $server_pid 2>/dev/null || true
    exit 1
}

# Deploy and configure remote worker
deploy_remote_worker() {
    log_info "Deploying remote worker to thinker.local..."

    # Use the deployment system to deploy the worker
    if [[ -f "./build/deployment-cli" ]]; then
        log_info "Using automated deployment system..."

        # Generate deployment plan for remote worker
        ./build/deployment-cli -action generate-plan -config "$MAIN_CONFIG" -verbose

        # Deploy the worker
        ./build/deployment-cli -action deploy -plan deployment-plan.json -verbose

        # Wait for deployment to complete
        sleep 10

        # Check deployment status
        ./build/deployment-cli -action status

    else
        log_warn "Automated deployment system not available, using manual deployment..."

        # Manual deployment using existing scripts
        ./deploy_worker.sh

        # Wait for worker to be ready
        sleep 5
    fi

    # Verify worker is accessible
    if curl -k -f "https://thinker.local:8443/health" >/dev/null 2>&1; then
        log_success "Remote worker is healthy"
    else
        log_error "Remote worker is not responding"
        return 1
    fi
}

# Discover and pair workers
discover_workers() {
    log_info "Discovering and pairing workers..."

    # Make API call to discover workers
    local response
    response=$(curl -s -k -X POST "https://localhost:8443/api/v1/distributed/workers/discover" \
        -H "Content-Type: application/json")

    if [[ $? -eq 0 ]]; then
        log_success "Worker discovery initiated"
        log_info "Response: $response"
    else
        log_error "Worker discovery failed"
        return 1
    fi

    # Wait for pairing to complete
    sleep 5

    # Check distributed status
    local status
    status=$(curl -s -k "https://localhost:8443/api/v1/distributed/status")

    if echo "$status" | grep -q "paired_workers.*[1-9]"; then
        log_success "Workers successfully paired"
    else
        log_error "Worker pairing failed"
        log_info "Status: $status"
        return 1
    fi
}

# Get list of books to translate
get_books_list() {
    # Find all supported book formats, excluding already translated files
    local books=()
    while IFS= read -r -d '' file; do
        books+=("$file")
    done < <(find "$BOOKS_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.mobi" -o -name "*.pdf" -o -name "*.azw3" \) ! -name "*_sr*" ! -name "translation_report*" ! -name "*.md" ! -name "*.jpg" -print0)

    printf '%s\n' "${books[@]}"
}

# Translate a single book with retry logic
translate_book() {
    local book_path="$1"
    local book_name
    book_name=$(basename "$book_path")
    local book_basename="${book_name%.*}"

    log_info "Translating book: $book_name"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Step 1: Convert source to markdown with retry
    local md_source="$OUTPUT_DIR/${book_basename}.md"
    if ! retry_command "pandoc -f \"${book_path##*.}\" -t markdown \"$book_path\" -o \"$md_source\" --extract-media=\"$OUTPUT_DIR\"" 3; then
        log_error "Failed to convert $book_name to markdown after retries"
        return 1
    fi
    log_info "Converted source to markdown: $md_source"

    # Step 2: Preparation phase (analyze content with LLM)
    log_info "Preparation phase: Analyzing content with multi-LLM analysis..."

    # Step 3: Translate the ebook with retry
    local temp_output="$OUTPUT_DIR/${book_basename}_temp.fb2"
    local api_endpoint="fb2"
    local start_time
    start_time=$(date +%s)

    local max_retries=3
    local retry_count=0
    local success=false

    while [[ $retry_count -lt $max_retries ]]; do
        log_info "Translation attempt $((retry_count + 1))/$max_retries for $book_name"

    local response
    response=$(curl -s -k -X POST "https://localhost:8443/api/v1/translate/$api_endpoint" \
        -F "file=@$book_path" \
        -F "provider=dictionary" \
        -o "$temp_output" 2>&1)

        if [[ -f "$temp_output" ]] && [[ -s "$temp_output" ]]; then
            success=true
            break
        else
            log_warn "Translation attempt $((retry_count + 1)) failed: $response"
            ((retry_count++))
            if [[ $retry_count -lt $max_retries ]]; then
                sleep $((retry_count * 2))  # Exponential backoff
            fi
        fi
    done

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [[ "$success" != "true" ]]; then
        log_error "Translation failed after $max_retries attempts: $book_name"
        return 1
    fi

    # Step 4: Generate final output format
    local output_file="$OUTPUT_DIR/${book_basename}_sr.${OUTPUT_FORMAT}"

    # Copy the translated file directly (API returns EPUB)
    if cp "$temp_output" "$output_file"; then
        log_info "Generated $OUTPUT_FORMAT: $output_file"
        log_success "Translation completed: $book_name -> $output_file (${duration}s)"
    else
        log_error "Failed to copy translated file for $book_name"
        return 1
    fi

    # Copy preparation analysis if it exists
    prep_analysis="${temp_output%.*}_preparation.json"
    if [[ -f "$prep_analysis" ]]; then
        cp "$prep_analysis" "$OUTPUT_DIR/"
        log_info "Preparation analysis copied: $(basename "$prep_analysis")"
    fi

    # Clean up temp file
    rm -f "$temp_output"

    local size
    size=$(stat -f%z "$output_epub" 2>/dev/null || stat -c%s "$output_epub" 2>/dev/null || echo "unknown")
    log_success "Translation completed: $book_name -> ${book_basename}_sr.epub (${size} bytes, ${duration}s)"
    log_info "Generated formats: EPUB, FB2, MOBI, PDF, AZW3"
}

# Monitor API communications
monitor_api_communications() {
    log_info "Monitoring API communications..."

    if [[ -f "$API_LOG" ]]; then
        local initial_lines
        initial_lines=$(wc -l < "$API_LOG")

        # Start monitoring in background
        (
            while true; do
                sleep 10
                local current_lines
                current_lines=$(wc -l < "$API_LOG")
                local new_lines=$((current_lines - initial_lines))

                if [[ $new_lines -gt 0 ]]; then
                    log_info "API communications: $new_lines new entries"
                    tail -n $new_lines "$API_LOG" | jq -r '"\(.timestamp): \(.method) \(.url) -> \(.status_code) (\(.duration))"' 2>/dev/null || true
                fi
            done
        ) &
        local monitor_pid=$!
        echo $monitor_pid > monitor.pid
    else
        log_warn "API log file not found: $API_LOG"
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
Translation Report
==================

Generated: $(date)
Total books processed: $total_books
Successful translations: $successful_translations
Failed translations: $failed_translations
Total time: ${total_time}s
Average time per book: $((total_time / total_books))s

API Communications Summary:
$(if [[ -f "$API_LOG" ]]; then
    echo "Total API calls: $(wc -l < "$API_LOG")"
    echo "Status code distribution:"
    jq -r '.status_code // "unknown"' "$API_LOG" 2>/dev/null | sort | uniq -c | sort -nr || echo "Unable to parse API log"
else
    echo "API log not available"
fi)

Output directory: $OUTPUT_DIR
API log: $API_LOG
Batch log: $LOG_FILE
EOF

    log_success "Report generated: $OUTPUT_DIR/translation_report.txt"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."

    # Kill background processes
    if [[ -f "server.pid" ]]; then
        kill "$(cat server.pid)" 2>/dev/null || true
        rm -f server.pid
    fi

    if [[ -f "monitor.pid" ]]; then
        kill "$(cat monitor.pid)" 2>/dev/null || true
        rm -f monitor.pid
    fi

    # Stop deployment if using automated system
    if [[ -f "./build/deployment-cli" ]]; then
        ./build/deployment-cli -action stop 2>/dev/null || true
    fi

    log_info "Cleanup completed"
}

# Main execution
main() {
    # Set up cleanup trap
    trap cleanup EXIT

    log_info "=== Distributed Batch Translation Started ==="
    log_info "Books directory: $BOOKS_DIR"
    log_info "Output directory: $OUTPUT_DIR"
    log_info "Main config: $MAIN_CONFIG"
    log_info "Worker config: $WORKER_CONFIG"

    # Validate prerequisites
    validate_prerequisites

    # Start main server
    start_main_server

    # Deploy remote worker
    # deploy_remote_worker  # Using local worker

    # Discover and pair workers
    # discover_workers  # Skipping discovery, using direct worker

    # Start API monitoring
    monitor_api_communications

    # Get list of books
    local books=()
    while IFS= read -r book; do
        books+=("$book")
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

        # Small delay between translations
        sleep 1
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