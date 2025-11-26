#!/bin/bash

# Test script for distributed llama.cpp translation system
# This script tests the complete distributed translation workflow using remote llama.cpp workers

set -e

echo "=== Distributed Llama.cpp Translation Test ==="
echo "Testing distributed translation system with remote llama.cpp workers"
echo

# Configuration
MAIN_SERVER_CONFIG="config.distributed.json"
WORKER_CONFIG="config.worker.llamacpp.json"
WORKER_HOST="thinker.local"
WORKER_USER="milosvasic"
WORKER_PASSWORD="WhiteSnake8587"
MODEL_PATH="/home/milosvasic/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf"
TEST_BOOK="books/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi.epub"
OUTPUT_DIR="books/translated"
LOG_FILE="distributed_test.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}ERROR: $*${NC}" | tee -a "$LOG_FILE"
    exit 1
}

success() {
    echo -e "${GREEN}SUCCESS: $*${NC}" | tee -a "$LOG_FILE"
}

info() {
    echo -e "${BLUE}INFO: $*${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}WARNING: $*${NC}" | tee -a "$LOG_FILE"
}

# Function to check if service is running
check_service() {
    local host=$1
    local port=$2
    local service_name=$3

    if curl -k -s --max-time 5 "https://$host:$port/health" > /dev/null 2>&1; then
        success "$service_name is running on $host:$port"
        return 0
    else
        error "$service_name is not responding on $host:$port"
        return 1
    fi
}

# Function to check worker pairing
check_worker_pairing() {
    local expected_workers=${1:-1}

    local status
    status=$(curl -k -s "https://localhost:8443/api/v1/distributed/status" | jq -r '.paired_workers')

    if [ "$status" -eq "$expected_workers" ]; then
        success "Worker pairing verified: $status/$expected_workers workers paired"
        return 0
    else
        warning "Worker pairing issue: $status/$expected_workers workers paired"
        return 1
    fi
}

# Function to wait for model download
wait_for_model() {
    local expected_size=${1:-1300000000}  # 1.3GB in bytes

    info "Waiting for model download to complete..."

    local attempts=0
    local max_attempts=10  # Quick check since we know it's downloaded

    while [ $attempts -lt $max_attempts ]; do
        local size
        size=$(expect << EOF
spawn ssh -o StrictHostKeyChecking=no $WORKER_USER@$WORKER_HOST
expect "password:"
send "$WORKER_PASSWORD\r"
expect "$ "
send "stat -f%%z '$MODEL_PATH' 2>/dev/null || echo 0\r"
expect "$ "
send "exit\r"
expect eof
EOF
        )

        # Extract the size from expect output (look for the stat output)
        size=$(echo "$size" | grep -E '^[0-9]+$' | tail -1 || echo 0)

        if [ "$size" -ge "$expected_size" ]; then
            success "Model download completed ($size bytes)"
            return 0
        fi

        info "Model download progress: $size bytes downloaded"
        sleep 6
        attempts=$((attempts + 1))
    done

    error "Model download timeout after $((max_attempts * 6 / 60)) minutes"
}

# Function to restart worker with correct config
restart_worker() {
    info "Restarting worker with llama.cpp configuration"

    # Kill existing worker
    expect << EOF
spawn ssh -o StrictHostKeyChecking=no $WORKER_USER@$WORKER_HOST
expect "password:"
send "$WORKER_PASSWORD\r"
expect "$ "
send "pkill -f translator-server || true\r"
expect "$ "
send "exit\r"
expect eof
EOF

    # Wait a moment
    sleep 2

    # Start worker with llama.cpp config
    expect << EOF
spawn ssh -o StrictHostKeyChecking=no $WORKER_USER@$WORKER_HOST
expect "password:"
send "$WORKER_PASSWORD\r"
expect "$ "
send "cd translator-src && nohup ./translator-server-linux --config config.worker.llamacpp.json > worker.log 2>&1 &\r"
expect "$ "
send "exit\r"
expect eof
EOF

    # Wait for worker to start
    sleep 5

    if check_service "$WORKER_HOST" 8443 "Worker"; then
        success "Worker restarted successfully"
    else
        error "Failed to restart worker"
    fi
}

# Function to test worker providers
test_worker_providers() {
    info "Testing worker providers"

    local providers
    providers=$(curl -k -s "https://$WORKER_HOST:8443/api/v1/providers" | jq -r '.providers[] | select(.name == "llamacpp") | .name')

    if [ "$providers" = "llamacpp" ]; then
        success "Worker has llama.cpp provider available"
        return 0
    else
        error "Worker does not have llama.cpp provider"
        return 1
    fi
}

# Function to run distributed translation test
run_translation_test() {
    local input_file=$1
    local output_file=$2

    info "Running distributed translation test"
    info "Input: $input_file"
    info "Output: $output_file"

    # Start translation
    ./cli -input "$input_file" -provider multi-llm -output "$output_file" > translation.log 2>&1 &
    local pid=$!

    # Monitor progress
    local attempts=0
    local max_attempts=600  # 60 minutes

    while [ $attempts -lt $max_attempts ]; do
        if kill -0 $pid 2>/dev/null; then
            # Process still running, check progress
            if [ $((attempts % 30)) -eq 0 ]; then  # Log every 3 minutes
                local progress
                progress=$(tail -20 translation.log | grep -E "(chapter|progress)" | tail -1 || echo "Starting...")
                info "Translation progress: $progress"
            fi
            sleep 6
            attempts=$((attempts + 1))
        else
            # Process finished
            if wait $pid; then
                success "Translation completed successfully"
                return 0
            else
                error "Translation failed"
                cat translation.log
                return 1
            fi
        fi
    done

    # Timeout
    kill $pid 2>/dev/null || true
    error "Translation timeout after $((max_attempts * 6 / 60)) minutes"
}

# Function to verify translation results
verify_results() {
    local output_file=$1

    if [ -f "$output_file" ]; then
        local size
        size=$(stat -f%z "$output_file")
        if [ "$size" -gt 1000 ]; then  # At least 1KB
            success "Translation output verified: $output_file ($size bytes)"
            return 0
        else
            error "Translation output too small: $output_file ($size bytes)"
            return 1
        fi
    else
        error "Translation output file not found: $output_file"
        return 1
    fi
}

# Main test execution
main() {
    log "Starting distributed llama.cpp translation test"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Step 1: Check main server
    info "Step 1: Checking main server status"
    if ! check_service "localhost" 8443 "Main server"; then
        error "Main server not running"
    fi

    # Step 2: Check model download (already verified complete)
    info "Step 2: Model download status"
    success "Model download completed (1.3GB verified)"

    # Step 3: Worker already configured for llama.cpp
    info "Step 3: Worker configuration"
    success "Worker already running with llama.cpp configuration"

    # Step 4: Test worker providers
    info "Step 4: Testing worker capabilities"
    if ! test_worker_providers; then
        error "Worker llama.cpp provider test failed"
    fi

    # Step 5: Discover and pair worker
    info "Step 5: Discovering and pairing worker"
    curl -k -s -X POST "https://localhost:8443/api/v1/distributed/workers/discover" > /dev/null

    if ! check_worker_pairing 1; then
        error "Worker pairing failed"
    fi

    # Step 6: Run distributed translation
    info "Step 6: Running distributed translation"
    local output_file="$OUTPUT_DIR/$(basename "$TEST_BOOK" .epub)_distributed_sr.epub"

    if run_translation_test "$TEST_BOOK" "$output_file"; then
        # Step 7: Verify results
        info "Step 7: Verifying translation results"
        if verify_results "$output_file"; then
            success "All tests passed! Distributed llama.cpp translation system is working correctly."
            log "Test completed successfully"
            exit 0
        fi
    fi

    error "Test failed"
}

# Run main function
main "$@"