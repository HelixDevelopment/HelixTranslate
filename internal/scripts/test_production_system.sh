#!/bin/bash

# Production Test Suite for SSH Translation System
# Tests hash verification, multi-LLM translation, and complete workflow

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="/tmp/translate_test_$(date +%s)"
REMOTE_TEST_DIR="/tmp/translate_ssh_test"
RESULTS_FILE="$TEST_DIR/test_results.json"
LOG_FILE="$TEST_DIR/test.log"

# SSH configuration (from environment or defaults)
SSH_HOST="${SSH_HOST:-thinker.local}"
SSH_USER="${SSH_USER:-milosvasic}"
SSH_PASSWORD="${SSH_PASSWORD:-WhiteSnake8587}"

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $*${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR] $*${NC}" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS] $*${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING] $*${NC}" | tee -a "$LOG_FILE"
}

# Test result functions
record_test() {
    local test_name="$1"
    local status="$2"
    local message="${3:-}"
    local details="${4:-}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    if [[ "$status" == "PASS" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        success "TEST PASSED: $test_name - $message"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        error "TEST FAILED: $test_name - $message"
    fi
    
    # Record to JSON
    local entry="{\"name\":\"$test_name\",\"status\":\"$status\",\"message\":\"$message\",\"details\":\"$details\",\"timestamp\":\"$(date -Iseconds)\"}"
    
    if [[ ! -f "$RESULTS_FILE" ]]; then
        echo "{\"tests\":[$entry]}" > "$RESULTS_FILE"
    else
        # Simple JSON array append (not robust but works for testing)
        sed -i.bak "s/]/,$entry]/" "$RESULTS_FILE" && rm "$RESULTS_FILE.bak"
    fi
}

# Test 1: SSH Connection Test
test_ssh_connection() {
    log "Testing SSH connection to $SSH_HOST..."
    
    if sshpass -p "$SSH_PASSWORD" ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no "$SSH_USER@$SSH_HOST" "echo 'SSH connection successful'" > /dev/null 2>&1; then
        record_test "SSH Connection" "PASS" "Successfully connected to $SSH_HOST"
        return 0
    else
        record_test "SSH Connection" "FAIL" "Failed to connect to $SSH_HOST"
        return 1
    fi
}

# Test 2: Codebase Hash Generation
test_codebase_hash() {
    log "Testing codebase hash generation..."
    
    local original_dir="$PWD"
    
    if cd "$PROJECT_ROOT" 2>/dev/null || cd .. 2>/dev/null; then
        if python3 scripts/codebase_hasher.py calculate > "$TEST_DIR/hash1.txt" 2>/dev/null; then
            local hash1=$(cat "$TEST_DIR/hash1.txt")
            
            # Generate second hash to verify consistency
            if python3 scripts/codebase_hasher.py calculate > "$TEST_DIR/hash2.txt" 2>/dev/null; then
                local hash2=$(cat "$TEST_DIR/hash2.txt")
                
                if [[ "$hash1" == "$hash2" ]]; then
                    record_test "Codebase Hash Generation" "PASS" "Hash generation consistent: ${hash1:0:16}..." "Hash: $hash1"
                else
                    record_test "Codebase Hash Generation" "FAIL" "Hashes not consistent: ${hash1:0:16}... vs ${hash2:0:16}..."
                fi
            else
                record_test "Codebase Hash Generation" "FAIL" "Failed to generate second hash"
            fi
        else
            record_test "Codebase Hash Generation" "FAIL" "Failed to generate first hash"
        fi
        
        cd "$original_dir"
    else
        record_test "Codebase Hash Generation" "FAIL" "Could not navigate to project root"
    fi
}

# Test 3: Remote Hash Comparison
test_remote_hash() {
    log "Testing remote hash comparison..."
    
    # Generate local hash
    cd "$PROJECT_ROOT" 2>/dev/null || cd ..
    local local_hash=$(python3 scripts/codebase_hasher.py calculate 2>/dev/null || echo "")
    
    if [[ -n "$local_hash" ]]; then
        # Get remote hash (simplified test)
        local remote_hash=$(sshpass -p "$SSH_PASSWORD" ssh "$SSH_USER@$SSH_HOST" \
            "cd $REMOTE_TEST_DIR 2>/dev/null && python3 scripts/codebase_hasher.py calculate 2>/dev/null" || echo "")
        
        if [[ -n "$remote_hash" ]]; then
            if [[ "$local_hash" == "$remote_hash" ]]; then
                record_test "Remote Hash Comparison" "PASS" "Local and remote hashes match" "Hash: ${local_hash:0:16}..."
            else
                record_test "Remote Hash Comparison" "FAIL" "Hashes differ - synchronization needed" "Local: ${local_hash:0:16}..., Remote: ${remote_hash:0:16}..."
            fi
        else
            record_test "Remote Hash Comparison" "FAIL" "Could not get remote hash"
        fi
    else
        record_test "Remote Hash Comparison" "FAIL" "Could not generate local hash"
    fi
    
    cd - > /dev/null
}

# Test 4: Build System Test
test_build_system() {
    log "Testing build system..."
    
    if go build -o "$TEST_DIR/test_translator" ./cmd/translate-ssh 2>/dev/null; then
        if [[ -x "$TEST_DIR/test_translator" ]]; then
            record_test "Build System" "PASS" "Successfully built translation system"
        else
            record_test "Build System" "FAIL" "Built binary is not executable"
        fi
    else
        record_test "Build System" "FAIL" "Failed to build translation system"
    fi
}

# Test 5: Script Execution Test
test_script_execution() {
    log "Testing script execution..."
    
    # Test codebase hasher script
    if python3 "$PROJECT_ROOT/scripts/codebase_hasher.py" --help > /dev/null 2>&1; then
        record_test "Script Execution" "PASS" "Codebase hasher script executable"
    else
        record_test "Script Execution" "FAIL" "Codebase hasher script failed"
    fi
}

# Test 6: Configuration Validation
test_configuration() {
    log "Testing configuration validation..."
    
    local config_file="$TEST_DIR/test_config.json"
    cat > "$config_file" << EOF
{
  "models": ["/models/test.gguf"],
  "n_ctx": 4096,
  "max_tokens": 2048,
  "temperature": 0.7,
  "top_p": 0.95
}
EOF
    
    if python3 -c "import json; json.load(open('$config_file'))" 2>/dev/null; then
        record_test "Configuration Validation" "PASS" "JSON configuration valid"
    else
        record_test "Configuration Validation" "FAIL" "Invalid JSON configuration"
    fi
}

# Test 7: File Operations Test
test_file_operations() {
    log "Testing file operations..."
    
    # Create test file
    local test_file="$TEST_DIR/test_input.md"
    echo "# Test Document\\n\\nThis is a test paragraph for translation." > "$test_file"
    
    # Test script with test data
    local output_file="$TEST_DIR/test_output.md"
    
    if python3 "$PROJECT_ROOT/scripts/translate_markdown_test.sh" "$test_file" "$output_file" 2>/dev/null; then
        if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
            record_test "File Operations" "PASS" "Script processes files correctly"
        else
            record_test "File Operations" "FAIL" "Output file not created or empty"
        fi
    else
        record_test "File Operations" "FAIL" "Script execution failed"
    fi
}

# Test 8: End-to-End Workflow Test
test_e2e_workflow() {
    log "Testing end-to-end workflow..."
    
    # Create minimal test FB2 file
    local test_fb2="$TEST_DIR/test_book.fb2"
    cat > "$test_fb2" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>sf</genre>
      <book-title>Test Book</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
  <body>
    <title><p>Test Chapter</p></title>
    <section>
      <p>Это тестовый абзац для перевода.</p>
      <p>Второй абзац текста.</p>
    </section>
  </body>
</FictionBook>
EOF
    
    # Run translation workflow (dry run if possible)
    local output_epub="$TEST_DIR/test_book_sr.epub"
    
    if timeout 60 ./build/translator-ssh \
        --input "$test_fb2" \
        --output "$output_epub" \
        --host "$SSH_HOST" \
        --user "$SSH_USER" \
        --password "$SSH_PASSWORD" \
        --report-dir "$TEST_DIR/e2e_test" > "$TEST_DIR/e2e_output.log" 2>&1; then
        
        if [[ -f "$output_epub" ]]; then
            record_test "End-to-End Workflow" "PASS" "Complete workflow executed successfully"
        else
            record_test "End-to-End Workflow" "FAIL" "EPUB output file not created"
        fi
    else
        # Check if it's a timeout or other expected issue
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            record_test "End-to-End Workflow" "FAIL" "Workflow timed out (60s)"
        else
            record_test "End-to-End Workflow" "FAIL" "Workflow failed with exit code $exit_code"
        fi
        
        warning "E2E output log:"
        cat "$TEST_DIR/e2e_output.log" | tail -20
    fi
}

# Test 9: Error Handling Test
test_error_handling() {
    log "Testing error handling..."
    
    # Test with invalid input file
    if ! ./build/translator-ssh --input "/nonexistent.fb2" --output "test.epub" \
        --host "$SSH_HOST" --user "$SSH_USER" --password "$SSH_PASSWORD" \
        --report-dir "$TEST_DIR/error_test" > "$TEST_DIR/error_output.log" 2>&1; then
        
        record_test "Error Handling" "PASS" "System correctly handles invalid input"
    else
        record_test "Error Handling" "FAIL" "System should have failed with invalid input"
    fi
}

# Test 10: Hash-Based Synchronization Test
test_hash_sync() {
    log "Testing hash-based synchronization..."
    
    # Generate initial hash
    cd "$PROJECT_ROOT" 2>/dev/null || cd ..
    local initial_hash=$(python3 scripts/codebase_hasher.py calculate 2>/dev/null)
    
    if [[ -n "$initial_hash" ]]; then
        # Simulate remote sync process
        if ./build/translator-ssh --input test_book_small.fb2 --output test_sync.epub \
            --host "$SSH_HOST" --user "$SSH_USER" --password "$SSH_PASSWORD" \
            --report-dir "$TEST_DIR/sync_test" > "$TEST_DIR/sync_output.log" 2>&1; then
            
            # Check if sync process logged hash comparison
            if grep -q "codebase.*hash" "$TEST_DIR/sync_output.log" 2>/dev/null; then
                record_test "Hash-Based Synchronization" "PASS" "Hash verification process executed"
            else
                record_test "Hash-Based Synchronization" "FAIL" "Hash verification not found in logs"
            fi
        else
            record_test "Hash-Based Synchronization" "FAIL" "Synchronization test failed"
        fi
    else
        record_test "Hash-Based Synchronization" "FAIL" "Could not generate initial hash"
    fi
}

# Main test execution
main() {
    log "Starting Production SSH Translation System Test Suite"
    log "Test directory: $TEST_DIR"
    log "SSH target: $SSH_USER@$SSH_HOST"
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    touch "$TEST_DIR/test.log"
    
    # Find project root
    PROJECT_ROOT="$PWD"
    if [[ ! -f "$PROJECT_ROOT/scripts/codebase_hasher.py" ]]; then
        PROJECT_ROOT="$(cd .. && pwd)"
    fi
    
    # Build the system first
    log "Building translation system for tests..."
    if ! go build -o build/translator-ssh ./cmd/translate-ssh; then
        error "Failed to build translation system - aborting tests"
        exit 1
    fi
    
    # Run all tests
    test_ssh_connection
    test_codebase_hash
    test_remote_hash
    test_build_system
    test_script_execution
    test_configuration
    test_file_operations
    test_e2e_workflow
    test_error_handling
    test_hash_sync
    
    # Generate final report
    log "Test Suite Summary"
    log "=================="
    log "Total Tests: $TESTS_TOTAL"
    log "Passed: $TESTS_PASSED"
    log "Failed: $TESTS_FAILED"
    
    local success_rate=$((TESTS_PASSED * 100 / TESTS_TOTAL))
    log "Success Rate: ${success_rate}%"
    
    # Save comprehensive results
    local final_report="$TEST_DIR/final_report.json"
    cat > "$final_report" << EOF
{
  "summary": {
    "total_tests": $TESTS_TOTAL,
    "passed": $TESTS_PASSED,
    "failed": $TESTS_FAILED,
    "success_rate": $success_rate,
    "timestamp": "$(date -Iseconds)",
    "test_directory": "$TEST_DIR",
    "ssh_target": "$SSH_HOST"
  },
  "artifacts": {
    "results_file": "$RESULTS_FILE",
    "log_file": "$LOG_FILE",
    "final_report": "$final_report"
  }
}
EOF
    
    log "Detailed results saved to: $RESULTS_FILE"
    log "Final report saved to: $final_report"
    
    # Exit with appropriate code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        success "All tests passed!"
        exit 0
    else
        error "$TESTS_FAILED test(s) failed"
        exit 1
    fi
}

# Check dependencies
if ! command -v sshpass &> /dev/null; then
    error "sshpass is required but not installed"
    exit 1
fi

if ! command -v python3 &> /dev/null; then
    error "python3 is required but not installed"
    exit 1
fi

# Run main function
main "$@"