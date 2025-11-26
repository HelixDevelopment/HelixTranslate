#!/bin/bash

# Production-ready Llama.cpp Multi-LLM Translation System
# Usage: translate_markdown_llamacpp.sh <input_file> <output_file> <workflow_config> <llama_config>

set -e

INPUT_FILE="$1"
OUTPUT_FILE="$2"
WORKFLOW_CONFIG="$3"
LLAMA_CONFIG="$4"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Check dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    command -v python3 >/dev/null 2>&1 || { error "Python3 is required but not installed."; exit 1; }
    command -v jq >/dev/null 2>&1 || { error "jq is required but not installed."; exit 1; }
    
    # Check if llama.cpp is available
    if ! python3 -c "import llama_cpp" 2>/dev/null; then
        warn "llama-cpp-python not found, attempting to install..."
        pip3 install llama-cpp-python --extra-index-url https://abetlen.github.io/llama-cpp-python/whl/cu121 || \
        pip3 install llama-cpp-python --extra-index-url https://abetlen.github.io/llama-cpp-python/whl/cpu || \
        { error "Failed to install llama-cpp-python"; exit 1; }
    fi
    
    log "Dependencies check completed"
}

# Validate input
validate_input() {
    if [[ ! -f "$INPUT_FILE" ]]; then
        error "Input file not found: $INPUT_FILE"
        exit 1
    fi
    
    if [[ ! -f "$WORKFLOW_CONFIG" ]]; then
        error "Workflow config not found: $WORKFLOW_CONFIG"
        exit 1
    fi
    
    if [[ ! -f "$LLAMA_CONFIG" ]]; then
        error "Llama config not found: $LLAMA_CONFIG"
        exit 1
    fi
    
    log "Input validation completed"
    log "Input file: $INPUT_FILE"
    log "Output file: $OUTPUT_FILE"
}

# Load configurations
load_configs() {
    log "Loading configuration files..."
    
    # Load workflow config
    WORKFLOW_JSON=$(cat "$WORKFLOW_CONFIG")
    
    # Load llama config
    LLAMA_JSON=$(cat "$LLAMA_CONFIG")
    
    # Extract settings using jq
    MODEL_PATH=$(echo "$LLAMA_JSON" | jq -r '.model_path // "/models/llama-3-8b-instruct.gguf"')
    N_CTX=$(echo "$LLAMA_JSON" | jq -r '.n_ctx // 4096')
    N_GPU_LAYERS=$(echo "$LLAMA_JSON" | jq -r '.n_gpu_layers // -1')
    TEMPERATURE=$(echo "$LLAMA_JSON" | jq -r '.temperature // 0.7')
    TOP_P=$(echo "$LLAMA_JSON" | jq -r '.top_p // 0.95')
    
    # Extract workflow settings
    CHUNK_SIZE=$(echo "$WORKFLOW_JSON" | jq -r '.chunk_size // 2000')
    OVERLAP=$(echo "$WORKFLOW_JSON" | jq -r '.overlap // 200')
    MODEL_TYPE=$(echo "$WORKFLOW_JSON" | jq -r '.model_type // "llama3"')
    
    log "Configuration loaded successfully"
    info "Model: $MODEL_PATH"
    info "Context size: $N_CTX"
    info "GPU layers: $N_GPU_LAYERS"
    info "Chunk size: $CHUNK_SIZE"
}

# Translate single chunk using llama.cpp
translate_chunk() {
    local chunk_text="$1"
    local system_prompt="$2"
    
    python3 -c "
import sys
import json
import os
from llama_cpp import Llama

# Model configuration
MODEL_PATH = os.getenv('MODEL_PATH', '$MODEL_PATH')
N_CTX = int(os.getenv('N_CTX', '$N_CTX'))
N_GPU_LAYERS = int(os.getenv('N_GPU_LAYERS', '$N_GPU_LAYERS'))
TEMPERATURE = float(os.getenv('TEMPERATURE', '$TEMPERATURE'))
TOP_P = float(os.getenv('TOP_P', '$TOP_P'))

# System prompt for Russian to Serbian translation
SYSTEM_PROMPT = '''You are a professional translator specializing in Russian to Serbian translation. 
Translate the given text accurately while maintaining:
1. Original meaning and context
2. Cultural nuances and idioms
3. Professional tone and style
4. Markdown formatting
5. Character voice and narrative flow

Output only the translated text without explanations or notes.'''

# Initialize Llama model
try:
    llm = Llama(
        model_path=MODEL_PATH,
        n_ctx=N_CTX,
        n_gpu_layers=N_GPU_LAYERS,
        verbose=False
    )
except Exception as e:
    print(f'ERROR: Failed to load model: {e}', file=sys.stderr)
    sys.exit(1)

# Translation prompt
prompt = f'''<|begin_of_text|><|start_header_id|>system<|end_header_id|>
{SYSTEM_PROMPT}<|eot_id|><|start_header_id|>user<|end_header_id|>

Translate the following Russian text to Serbian, preserving markdown formatting:

{sys.argv[1]}<|eot_id|><|start_header_id|>assistant<|end_header_id|>'''

# Generate translation
try:
    output = llm(
        prompt,
        max_tokens=4000,
        temperature=TEMPERATURE,
        top_p=TOP_P,
        stop=['<|eot_id|>'],
        echo=False
    )
    
    # Extract only the assistant's response
    translated_text = output['choices'][0]['text'].strip()
    print(translated_text)
    
except Exception as e:
    print(f'ERROR: Translation failed: {e}', file=sys.stderr)
    sys.exit(1)
" "$chunk_text"
}

# Main translation function
perform_translation() {
    log "Starting translation process..."
    
    # Read input file
    local input_text
    input_text=$(cat "$INPUT_FILE")
    
    info "Input file size: $(echo "$input_text" | wc -c) characters"
    
    # Split text into chunks
    log "Splitting text into chunks..."
    local chunks
    chunks=$(split_text "$input_text" "$CHUNK_SIZE" "$OVERLAP")
    
    # Count chunks
    local chunk_count
    chunk_count=$(echo "$chunks" | grep -c '---CHUNK_[0-9]*---' || true)
    
    if [[ $chunk_count -eq 0 ]]; then
        warn "No chunks to translate"
        echo "$input_text" > "$OUTPUT_FILE"
        return 0
    fi
    
    info "Text split into $chunk_count chunks"
    
    # Translate each chunk
    local translated_chunks=""
    local current_chunk=0
    
    while IFS= read -r line; do
        if [[ $line =~ ^---CHUNK_([0-9]+)---$ ]]; then
            local chunk_num="${BASH_REMATCH[1]}"
            log "Translating chunk $((chunk_num + 1))/$chunk_count..."
            
            # Read chunk content until CHUNK_END
            local chunk_text=""
            while IFS= read -r chunk_line; do
                if [[ $chunk_line == "---CHUNK_END---" ]]; then
                    break
                fi
                chunk_text+="$chunk_line"$'\n'
            done
            
            # Translate the chunk
            local translated_chunk
            translated_chunk=$(translate_chunk "$chunk_text")
            
            if [[ $? -ne 0 ]]; then
                error "Failed to translate chunk $chunk_num"
                exit 1
            fi
            
            translated_chunks+="$translated_chunk"$'\n\n'
            
            # Progress update
            local progress=$(( (current_chunk + 1) * 100 / chunk_count ))
            info "Progress: $progress% ($((current_chunk + 1))/$chunk_count chunks)"
            ((current_chunk++))
            
        elif [[ $line != "---CHUNK_END---" ]]; then
            # This shouldn't happen in normal flow
            translated_chunks+="$line"$'\n'
        fi
    done <<< "$chunks"
    
    # Write translated content to output file
    log "Writing translated content to $OUTPUT_FILE..."
    echo -n "$translated_chunks" > "$OUTPUT_FILE"
    
    local output_size
    output_size=$(wc -c < "$OUTPUT_FILE")
    info "Translation completed"
    info "Output file size: $output_size characters"
    
    log "Translation process completed successfully"
}

# Main execution
main() {
    log "Starting Llama.cpp Translation System"
    log "====================================="
    
    check_dependencies
    validate_input
    load_configs
    perform_translation
    
    log "====================================="
    log "Translation system completed successfully"
}

# Execute main function
main "$@"