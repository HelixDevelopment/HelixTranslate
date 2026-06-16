#!/bin/bash

# EBook Translation Workflow Script
# This script translates an ebook from FB2 to Serbian Cyrillic EPUB using remote SSH worker with llama.cpp

set -e  # Exit on any error

# Default values
SOURCE_FILE=""
TARGET_LANGUAGE="sr-cyrl"
REMOTE_HOST="thinker.local"
REMOTE_USER="milosvasic"
REMOTE_PASS=""
VERBOSE=false
TEST_MODE=false

# Parse command line arguments
function parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --source|-s)
                SOURCE_FILE="$2"
                shift 2
                ;;
            --target|-t)
                TARGET_LANGUAGE="$2"
                shift 2
                ;;
            --host|-h)
                REMOTE_HOST="$2"
                shift 2
                ;;
            --user|-u)
                REMOTE_USER="$2"
                shift 2
                ;;
            --password|-p)
                REMOTE_PASS="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --test)
                TEST_MODE=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Show help message
function show_help() {
    cat << EOF
EBook Translation Workflow Script

Usage: $0 --source <fb2_file> [OPTIONS]

Required arguments:
    --source, -s <file>        Path to the source FB2 file

Optional arguments:
    --target, -t <lang>         Target language (default: sr-cyrl)
    --host, -h <host>          Remote SSH host (default: thinker.local)
    --user, -u <user>          SSH username (default: milosvasic)
    --password, -p <pass>      SSH password (default: <redacted-per-11.4.10>)
    --verbose, -v               Enable verbose logging
    --test                      Run in test mode (verify components only)
    --help                      Show this help message

Examples:
    $0 --source materials/books/book1.fb2
    $0 --source materials/books/book1.fb2 --target sr-cyrl --host thinker.local --user milosvasic --password <redacted-per-11.4.10> --verbose
    $0 --source materials/books/book1.fb2 --test

Supported target languages:
    sr-cyrl    - Serbian Cyrillic (default)
    sr          - Serbian (Latin)
    en          - English
    es          - Spanish
    fr          - French
    de          - German
    it          - Italian
    ru          - Russian

EOF
}

# Validate inputs
function validate_inputs() {
    if [[ -z "$SOURCE_FILE" ]]; then
        echo "Error: Source file is required"
        show_help
        exit 1
    fi

    if [[ ! -f "$SOURCE_FILE" ]]; then
        echo "Error: Source file not found: $SOURCE_FILE"
        exit 1
    fi

    if [[ "${SOURCE_FILE##*.}" != "fb2" ]]; then
        echo "Error: Source file must be an FB2 file (.fb2 extension)"
        exit 1
    fi

    # Check if the source file has readable content
    if [[ ! -r "$SOURCE_FILE" ]]; then
        echo "Error: Source file is not readable: $SOURCE_FILE"
        exit 1
    fi

    # Check file size (should be reasonable for an ebook)
    FILE_SIZE=$(stat -f%z "$SOURCE_FILE" 2>/dev/null || stat -c%s "$SOURCE_FILE" 2>/dev/null || echo "0")
    if [[ $FILE_SIZE -lt 100 ]]; then
        echo "Error: Source file appears to be too small ($FILE_SIZE bytes)"
        exit 1
    fi
}

# Check prerequisites
function check_prerequisites() {
    echo "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed or not in PATH"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Go version: $GO_VERSION"

    # Check if we can build the ebook-translator
    if ! go build -o /tmp/ebook-translator ./cmd/ebook-translator/; then
        echo "Error: Failed to build ebook-translator"
        exit 1
    fi

    # Check if SSH client is available
    if ! command -v ssh &> /dev/null; then
        echo "Error: SSH client is not installed or not in PATH"
        exit 1
    fi

    # Test SSH connection (non-verbose)
    echo "Testing SSH connection to $REMOTE_USER@$REMOTE_HOST..."
    if ! ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=no $REMOTE_USER@$REMOTE_HOST "echo 'SSH connection test successful'" 2>/dev/null; then
        echo "Warning: SSH connection test failed, but continuing anyway..."
        echo "         The workflow will attempt to establish connection during execution"
    fi

    echo "Prerequisites check completed"
}

# Build the ebook-translator binary
function build_translator() {
    echo "Building ebook-translator..."

    # Create output directory
    mkdir -p build

    # Build the binary
    if ! go build -o build/ebook-translator ./cmd/ebook-translator/; then
        echo "Error: Failed to build ebook-translator"
        exit 1
    fi

    echo "Ebook-translator built successfully: build/ebook-translator"
}

# Run tests
function run_tests() {
    echo "Running comprehensive tests..."

    # Run unit tests
    echo "Running unit tests..."
    if ! go test -v ./cmd/ebook-translator/; then
        echo "Error: Unit tests failed"
        exit 1
    fi

    # Run integration tests if SSH credentials are provided
    if [[ -n "$TEST_SSH_HOST" && -n "$TEST_SSH_USER" && -n "$TEST_SSH_PASS" ]]; then
        echo "Running integration tests..."
        export TEST_SSH_HOST="$TEST_SSH_HOST"
        export TEST_SSH_USER="$TEST_SSH_USER"
        export TEST_SSH_PASS="$TEST_SSH_PASS"
        if ! go test -v ./cmd/ebook-translator/ -run TestSSHWorkerConnection; then
            echo "Error: Integration tests failed"
            exit 1
        fi
    else
        echo "Skipping integration tests (no test credentials provided)"
    fi

    echo "All tests passed successfully"
}

# Run the translation workflow
function run_translation() {
    echo "Starting ebook translation workflow..."
    echo "Source file: $SOURCE_FILE"
    echo "Target language: $TARGET_LANGUAGE"
    echo "Remote host: $REMOTE_HOST"
    echo "Remote user: $REMOTE_USER"

    # Extract working directory from source file
    WORKING_DIR=$(dirname "$SOURCE_FILE")
    SOURCE_FILENAME=$(basename "$SOURCE_FILE")
    
    echo "Working directory: $WORKING_DIR"
    
    # Change to source file directory
    cd "$WORKING_DIR"

    # Prepare command arguments
    ARGS=(
        "--source" "$SOURCE_FILENAME"
        "--target" "$TARGET_LANGUAGE"
        "--host" "$REMOTE_HOST"
        "--user" "$REMOTE_USER"
        "--password" "$REMOTE_PASS"
    )

    if [[ "$VERBOSE" == "true" ]]; then
        ARGS+=("--verbose")
    fi

    echo "Executing: ./build/ebook-translator ${ARGS[*]}"

    # Run the translator
    if ! ./build/ebook-translator "${ARGS[@]}"; then
        echo "Error: Translation workflow failed"
        exit 1
    fi

    echo "Translation workflow completed successfully"
}

# Verify output files
function verify_output() {
    echo "Verifying output files..."

    # Get base name without extension
    BASE_NAME=$(basename "$SOURCE_FILE" .fb2)
    WORKING_DIR=$(dirname "$SOURCE_FILE")

    # Check for expected output files
    local original_md="$WORKING_DIR/${BASE_NAME}_original.md"
    local translated_md="$WORKING_DIR/${BASE_NAME}_original_translated.md"
    local final_epub="$WORKING_DIR/${BASE_NAME}_original_translated.epub"

    local missing_files=()

    [[ ! -f "$original_md" ]] && missing_files+=("Original Markdown: $original_md")
    [[ ! -f "$translated_md" ]] && missing_files+=("Translated Markdown: $translated_md")
    [[ ! -f "$final_epub" ]] && missing_files+=("Final EPUB: $final_epub")

    if [[ ${#missing_files[@]} -gt 0 ]]; then
        echo "Error: Missing output files:"
        for file in "${missing_files[@]}"; do
            echo "  - $file"
        done
        exit 1
    fi

    # Check file sizes
    echo "Output file sizes:"
    echo "  Original Markdown: $(stat -f%z "$original_md" 2>/dev/null || stat -c%s "$original_md") bytes"
    echo "  Translated Markdown: $(stat -f%z "$translated_md" 2>/dev/null || stat -c%s "$translated_md") bytes"
    echo "  Final EPUB: $(stat -f%z "$final_epub" 2>/dev/null || stat -c%s "$final_epub") bytes"

    # Basic validation of EPUB file
    if ! file "$final_epub" | grep -q "Zip archive"; then
        echo "Warning: Final EPUB file may not be valid (not a ZIP archive)"
    fi

    echo "Output verification completed"
}

# Cleanup function
function cleanup() {
    echo "Cleaning up temporary files..."
    # Remove any temporary files created during the process
    rm -f /tmp/ebook-translator
    rm -f /tmp/hasher
    echo "Cleanup completed"
}

# Main execution flow
function main() {
    echo "EBook Translation Workflow"
    echo "========================="
    echo ""

    # Parse command line arguments
    parse_args "$@"

    # Validate inputs
    validate_inputs

    # Set up cleanup on exit
    trap cleanup EXIT

    # Check prerequisites
    check_prerequisites

    # Build the translator
    build_translator

    # Run tests if requested
    if [[ "$TEST_MODE" == "true" ]]; then
        run_tests
        echo "Tests completed successfully"
        exit 0
    fi

    # Run the translation workflow
    run_translation

    # Verify output
    verify_output

    echo ""
    echo "========================="
    echo "Translation workflow completed successfully!"
    echo "Check the working directory for output files:"
    echo "  - Original Markdown"
    echo "  - Translated Markdown"
    echo "  - Final EPUB"
    echo ""
}

# Run main function with all arguments
main "$@"