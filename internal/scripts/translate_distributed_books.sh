#!/bin/bash

# Distributed Translation Script for Russian-Serbian FB2 Translation
# This script translates all books from materials/books directory to Serbian Cyrillic EPUB

set -e

# Configuration
CONFIG_FILE="config.distributed.thinker.json"
INPUT_DIR="materials/books"
OUTPUT_DIR="materials/books_translated"
LOG_FILE="translation_$(date +%Y%m%d_%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Check if required files exist
check_prerequisites() {
    log "Checking prerequisites..."
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi
    
    if [ ! -d "$INPUT_DIR" ]; then
        log_error "Input directory not found: $INPUT_DIR"
        exit 1
    fi
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Check if translator binary exists
    if [ ! -f "./translator" ]; then
        log_warning "Translator binary not found, building..."
        make build || {
            log_error "Failed to build translator"
            exit 1
        }
    fi
    
    log_success "Prerequisites check completed"
}

# Test SSH connection to worker
test_ssh_connection() {
    log "Testing SSH connection to thinker.local..."
    
    if ssh -o ConnectTimeout=10 -o BatchMode=yes milosvasic@thinker.local "echo 'SSH connection successful'" 2>/dev/null; then
        log_success "SSH connection to thinker.local successful"
        return 0
    else
        log_error "SSH connection to thinker.local failed"
        log_error "Please ensure:"
        log_error "1. thinker.local is reachable"
        log_error "2. SSH key authentication is set up"
        log_error "3. User 'milosvasic' has access"
        return 1
    fi
}

# Check if llama.cpp is running on worker
check_worker_status() {
    log "Checking llama.cpp status on thinker.local..."
    
    # Check if llama.cpp server is running
    if ssh milosvasic@thinker.local "pgrep -f 'llama.cpp' > /dev/null 2>&1"; then
        log_success "llama.cpp is running on thinker.local"
    else
        log_warning "llama.cpp not detected on thinker.local"
        log_warning "Attempting to start llama.cpp..."
        
        # Try to start llama.cpp (adjust paths as needed)
        ssh milosvasic@thinker.local "nohup /path/to/llama.cpp/server -m /path/to/model.gguf --host 0.0.0.0 --port 8080 > /dev/null 2>&1 &" || {
            log_warning "Could not start llama.cpp automatically"
            log_warning "Please ensure llama.cpp is running on thinker.local"
        }
    fi
}

# Translate a single file
translate_file() {
    local input_file="$1"
    local filename=$(basename "$input_file")
    local name="${filename%.*}"
    local output_file="$OUTPUT_DIR/${name}_sr.epub"
    
    log "Translating: $filename"
    
    # Run translation with distributed configuration
    if ./translator \
        -input "$input_file" \
        -output "$output_file" \
        -config "$CONFIG_FILE" \
        -provider distributed \
        -language Serbian \
        -script cyrillic \
        -format epub 2>&1 | tee -a "$LOG_FILE"; then
        
        log_success "Translation completed: $filename -> ${name}_sr.epub"
        
        # Verify output file was created and has content
        if [ -f "$output_file" ] && [ -s "$output_file" ]; then
            local file_size=$(stat -f%z "$output_file" 2>/dev/null || stat -c%s "$output_file" 2>/dev/null)
            log "Output file size: $file_size bytes"
            
            # Check if EPUB is valid (basic check)
            if unzip -t "$output_file" >/dev/null 2>&1; then
                log_success "EPUB file validation passed for $filename"
            else
                log_warning "EPUB validation failed for $filename"
            fi
        else
            log_error "Output file is empty or missing: $output_file"
            return 1
        fi
    else
        log_error "Translation failed for: $filename"
        return 1
    fi
    
    return 0
}

# Main translation function
translate_all_books() {
    log "Starting translation of all books in $INPUT_DIR"
    
    local total_files=0
    local successful_files=0
    local failed_files=0
    
    # Find all supported files
    while IFS= read -r -d '' file; do
        ((total_files++))
    done < <(find "$INPUT_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.txt" -o -name "*.html" -o -name "*.pdf" -o -name "*.mobi" -o -name "*.azw" -o -name "*.azw3" \) -print0)
    
    if [ $total_files -eq 0 ]; then
        log_error "No supported files found in $INPUT_DIR"
        exit 1
    fi
    
    log "Found $total_files files to translate"
    
    # Translate each file
    while IFS= read -r -d '' file; do
        if translate_file "$file"; then
            ((successful_files++))
        else
            ((failed_files++))
        fi
    done < <(find "$INPUT_DIR" -type f \( -name "*.fb2" -o -name "*.epub" -o -name "*.txt" -o -name "*.html" -o -name "*.pdf" -o -name "*.mobi" -o -name "*.azw" -o -name "*.azw3" \) -print0)
    
    # Summary
    log "Translation Summary:"
    log "  Total files: $total_files"
    log "  Successful: $successful_files"
    log "  Failed: $failed_files"
    
    if [ $failed_files -eq 0 ]; then
        log_success "All files translated successfully!"
    else
        log_warning "$failed_files files failed to translate"
    fi
}

# Verify final results
verify_results() {
    log "Verifying translation results..."
    
    local output_files=0
    local valid_epubs=0
    
    for file in "$OUTPUT_DIR"/*_sr.epub; do
        if [ -f "$file" ]; then
            ((output_files++))
            
            # Check if file is a valid EPUB
            if unzip -t "$file" >/dev/null 2>&1; then
                ((valid_epubs++))
                
                # Extract and check content
                local temp_dir=$(mktemp -d)
                if unzip -q "$file" -d "$temp_dir"; then
                    # Check for mandatory files
                    if [ -f "$temp_dir/mimetype" ] && [ -f "$temp_dir/META-INF/container.xml" ]; then
                        log "✓ Valid EPUB structure: $(basename "$file")"
                    else
                        log_warning "Invalid EPUB structure: $(basename "$file")"
                    fi
                    
                    # Check for content files
                    if find "$temp_dir" -name "*.xhtml" -o -name "*.html" | grep -q .; then
                        local content_size=$(find "$temp_dir" -name "*.xhtml" -o -name "*.html" -exec cat {} \; | wc -c)
                        log "  Content size: $content_size characters"
                        
                        if [ $content_size -gt 100 ]; then
                            log_success "✓ Content verified: $(basename "$file")"
                        else
                            log_warning "Very little content: $(basename "$file")"
                        fi
                    else
                        log_warning "No content files found: $(basename "$file")"
                    fi
                fi
                rm -rf "$temp_dir"
            else
                log_error "Invalid EPUB file: $(basename "$file")"
            fi
        fi
    done
    
    log "Verification Summary:"
    log "  Output files: $output_files"
    log "  Valid EPUBs: $valid_epubs"
    
    if [ $valid_epubs -eq $output_files ] && [ $output_files -gt 0 ]; then
        log_success "All output files are valid EPUBs with content!"
    else
        log_warning "Some output files may have issues"
    fi
}

# Main execution
main() {
    log "Starting distributed translation process"
    log "Log file: $LOG_FILE"
    
    check_prerequisites
    
    if ! test_ssh_connection; then
        log_error "Cannot proceed without SSH connection"
        exit 1
    fi
    
    check_worker_status
    
    translate_all_books
    
    verify_results
    
    log "Translation process completed!"
    log "Check the log file for details: $LOG_FILE"
}

# Handle script interruption
trap 'log_warning "Script interrupted"; exit 1' INT TERM

# Run main function
main "$@"