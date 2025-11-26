#!/bin/bash

# Production llama.cpp Translation Script
# Handles Russian to Serbian translation with proper environment setup

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="$SCRIPT_DIR/llama_cpp_production.log"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Error handling
error_exit() {
    log "ERROR: $*"
    exit 1
}

# Check if we have required parameters
if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <input_markdown> <output_markdown> [config_file]"
    echo "  input_markdown: Path to input markdown file"
    echo "  output_markdown: Path to output markdown file"
    echo "  config_file: Optional JSON configuration file"
    exit 1
fi

INPUT_FILE="$1"
OUTPUT_FILE="$2"
CONFIG_FILE="${3:-}"

log "Starting production llama.cpp translation"
log "Input: $INPUT_FILE"
log "Output: $OUTPUT_FILE"

# Verify input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    error_exit "Input file not found: $INPUT_FILE"
fi

# Create virtual environment if it doesn't exist
VENV_DIR="$HOME/translate_env"
if [[ ! -d "$VENV_DIR" ]]; then
    log "Creating Python virtual environment at $VENV_DIR"
    python3 -m venv "$VENV_DIR" || error_exit "Failed to create virtual environment"
fi

# Activate virtual environment
log "Activating virtual environment"
source "$VENV_DIR/bin/activate" || error_exit "Failed to activate virtual environment"

# Upgrade pip in virtual environment
log "Upgrading pip in virtual environment"
pip install --upgrade pip setuptools wheel || error_exit "Failed to upgrade pip"

# Install required packages
log "Installing required Python packages"
pip install llama-cpp-python || error_exit "Failed to install llama-cpp-python"

# Check for jq command
if ! command -v jq &> /dev/null; then
    error_exit "jq command not found. Please install jq: apt-get install jq"
fi

# Parse configuration if provided
LLAMA_MODEL=""
MODEL_CONTEXT=""
MAX_TOKENS=""
TEMPERATURE=""

if [[ -n "$CONFIG_FILE" && -f "$CONFIG_FILE" ]]; then
    log "Loading configuration from $CONFIG_FILE"
    LLAMA_MODEL=$(jq -r '.model_path // empty' "$CONFIG_FILE")
    MODEL_CONTEXT=$(jq -r '.context_size // empty' "$CONFIG_FILE")
    MAX_TOKENS=$(jq -r '.max_tokens // empty' "$CONFIG_FILE")
    TEMPERATURE=$(jq -r '.temperature // empty' "$CONFIG_FILE")
fi

# Use defaults if not specified
LLAMA_MODEL="${LLAMA_MODEL:-/models/llama-3-8b-instruct.Q4_K_M.gguf}"
MODEL_CONTEXT="${MODEL_CONTEXT:-4096}"
MAX_TOKENS="${MAX_TOKENS:-2048}"
TEMPERATURE="${TEMPERATURE:-0.7}"

log "Configuration:"
log "  Model: $LLAMA_MODEL"
log "  Context: $MODEL_CONTEXT"
log "  Max tokens: $MAX_TOKENS"
log "  Temperature: $TEMPERATURE"

# Verify model file exists
if [[ ! -f "$LLAMA_MODEL" ]]; then
    log "Warning: Model file not found: $LLAMA_MODEL"
    log "Please ensure the model file is available on the remote system"
    log "You can download models from: https://huggingface.co/TheBloke/Llama-2-7B-Chat-GGUF"
    
    # Try to find a model in common locations
    for search_path in "/models/*.gguf" "/tmp/*.gguf" "$HOME/*.gguf" "./models/*.gguf"; do
        for model in $search_path; do
            if [[ -f "$model" ]]; then
                log "Found model: $model"
                LLAMA_MODEL="$model"
                break 2
            fi
        done
    done
    
    if [[ ! -f "$LLAMA_MODEL" ]]; then
        error_exit "No GGUF model file found. Please download a model to $LLAMA_MODEL"
    fi
fi

# Create Python script for translation
PYTHON_SCRIPT="$SCRIPT_DIR/llama_translator.py"
cat > "$PYTHON_SCRIPT" << 'EOF'
#!/usr/bin/env python3
import sys
import json
import argparse
from pathlib import Path
from llama_cpp import Llama

def translate_text(input_file, output_file, model_path, context_size, max_tokens, temperature):
    """Translate text using llama.cpp"""
    
    # Initialize Llama model
    llm = Llama(
        model_path=model_path,
        n_ctx=int(context_size),
        n_gpu_layers=-1,  # Use GPU if available
        verbose=False
    )
    
    # Read input text
    with open(input_file, 'r', encoding='utf-8') as f:
        input_text = f.read()
    
    # Split text into chunks if too long
    max_chunk_length = int(max_tokens) // 4  # Rough estimate for tokens
    chunks = []
    current_chunk = ""
    
    for line in input_text.split('\n'):
        if len(current_chunk) + len(line) + 1 < max_chunk_length:
            current_chunk += line + '\n'
        else:
            if current_chunk.strip():
                chunks.append(current_chunk.strip())
            current_chunk = line + '\n'
    
    if current_chunk.strip():
        chunks.append(current_chunk.strip())
    
    print(f"Translating {len(chunks)} chunks...")
    
    # Translation prompt
    system_prompt = """You are a professional Russian to Serbian translator. 
Translate the given text from Russian to Serbian Cyrillic.
Preserve the original formatting, paragraph structure, and any markup.
Keep cultural nuances and literary style.
Return only the translated text without explanations."""

    translated_chunks = []
    
    for i, chunk in enumerate(chunks):
        print(f"Translating chunk {i+1}/{len(chunks)}...")
        
        # Create prompt for this chunk
        prompt = f"{system_prompt}\n\nRussian text:\n{chunk}\n\nSerbian translation:"
        
        try:
            # Generate translation
            output = llm(
                prompt,
                max_tokens=int(max_tokens),
                temperature=float(temperature),
                stop=["\n\n", "Russian text:", "Original text:"],
                echo=False
            )
            
            translated_text = output['choices'][0]['text'].strip()
            translated_chunks.append(translated_text)
            
        except Exception as e:
            print(f"Error translating chunk {i+1}: {e}")
            # Fallback: return original text
            translated_chunks.append(chunk)
    
    # Combine translated chunks
    translated_text = '\n\n'.join(translated_chunks)
    
    # Write output
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(translated_text)
    
    print(f"Translation complete: {output_file}")

def main():
    parser = argparse.ArgumentParser(description='Translate text using llama.cpp')
    parser.add_argument('--input', required=True, help='Input markdown file')
    parser.add_argument('--output', required=True, help='Output markdown file')
    parser.add_argument('--model', required=True, help='GGUF model file')
    parser.add_argument('--context', default='4096', help='Context size')
    parser.add_argument('--max-tokens', default='2048', help='Maximum tokens')
    parser.add_argument('--temperature', default='0.7', help='Temperature')
    
    args = parser.parse_args()
    
    translate_text(
        args.input,
        args.output,
        args.model,
        args.context,
        args.max_tokens,
        args.temperature
    )

if __name__ == "__main__":
    main()
EOF

# Execute translation
log "Starting translation with llama.cpp"
python3 "$PYTHON_SCRIPT" \
    --input "$INPUT_FILE" \
    --output "$OUTPUT_FILE" \
    --model "$LLAMA_MODEL" \
    --context "$MODEL_CONTEXT" \
    --max-tokens "$MAX_TOKENS" \
    --temperature "$TEMPERATURE" || error_exit "Translation failed"

# Verify output file was created
if [[ ! -f "$OUTPUT_FILE" ]]; then
    error_exit "Output file not created: $OUTPUT_FILE"
fi

# Get file sizes for reporting
INPUT_SIZE=$(stat -f%z "$INPUT_FILE" 2>/dev/null || stat -c%s "$INPUT_FILE" 2>/dev/null || echo "0")
OUTPUT_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null || echo "0")

log "Translation completed successfully"
log "Input size: $INPUT_SIZE bytes"
log "Output size: $OUTPUT_SIZE bytes"

# Generate translation stats
STATS_FILE="${OUTPUT_FILE}.stats"
cat > "$STATS_FILE" << EOF
{
  "input_file": "$INPUT_FILE",
  "output_file": "$OUTPUT_FILE",
  "input_size_bytes": $INPUT_SIZE,
  "output_size_bytes": $OUTPUT_SIZE,
  "model_used": "$LLAMA_MODEL",
  "context_size": $MODEL_CONTEXT,
  "max_tokens": $MAX_TOKENS,
  "temperature": $TEMPERATURE,
  "timestamp": "$(date -Iseconds)"
}
EOF

log "Translation stats saved to: $STATS_FILE"

# Cleanup
rm -f "$PYTHON_SCRIPT"

log "Production translation script completed successfully"