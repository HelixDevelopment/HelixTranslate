#!/bin/bash

# LLM-based Translation System using llama.cpp
# Russian to Serbian translation using local LLM inference
set -euo pipefail

INPUT_FILE="$1"
OUTPUT_FILE="$2"
CONFIG_FILE="$3"

LOG_FILE="llm_translation.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $*"
    exit 1
}

log "Starting LLM-based Translation System with llama.cpp"
log "Input: $INPUT_FILE"
log "Output: $OUTPUT_FILE"

# Check if input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    error_exit "Input file not found: $INPUT_FILE"
fi

# Configuration for llama.cpp
LLAMA_BINARY="${LLAMA_BINARY:-llama-cli}"
MODEL_PATH="${MODEL_PATH:-/home/milosvasic/llama.cpp/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf}"
THREADS="${THREADS:-8}"
CONTEXT_SIZE="${CONTEXT_SIZE:-4096}"
TEMPERATURE="${TEMPERATURE:-0.3}"
MAX_TOKENS="${MAX_TOKENS:-2048}"

# Check if llama.cpp binary exists
if ! command -v "$LLAMA_BINARY" >/dev/null 2>&1; then
    error_exit "llama.cpp binary not found: $LLAMA_BINARY. Install with: apt install llama.cpp or build from source"
fi

# Check if model exists
if [[ ! -f "$MODEL_PATH" ]]; then
    error_exit "Model file not found: $MODEL_PATH. Download the model first."
fi

log "Using llama.cpp binary: $LLAMA_BINARY"
log "Using model: $MODEL_PATH"
log "Configuration: $THREADS threads, $CONTEXT_SIZE context, temperature $TEMPERATURE"

# Create translation prompt template
TRANSLATION_PROMPT_TEMPLATE="Translate the following Russian text to Serbian Cyrillic. Maintain the original meaning, tone, and style. Only return the translated text without any explanations or additional comments.

Russian text:
%s

Serbian translation:"

# Create a Python script for LLM-based translation
cat > translate_llm.py << 'EOF'
#!/usr/bin/env python3
import sys
import json
import re
import subprocess
import tempfile
import os
from pathlib import Path

class LLMTranslator:
    def __init__(self):
        self.llama_binary = os.environ.get('LLAMA_BINARY', 'llama-cli')
        self.model_path = os.environ.get('MODEL_PATH', '/home/milosvasic/llama.cpp/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf')
        self.threads = int(os.environ.get('THREADS', '8'))
        self.context_size = int(os.environ.get('CONTEXT_SIZE', '4096'))
        self.temperature = float(os.environ.get('TEMPERATURE', '0.3'))
        self.max_tokens = int(os.environ.get('MAX_TOKENS', '2048'))
        
        # Translation prompt template
        self.prompt_template = """Translate the following Russian text to Serbian Cyrillic. Maintain the original meaning, tone, and style. Only return the translated text without any explanations or additional comments.

Russian text:
%s

Serbian translation:"""
    
    def check_dependencies(self):
        """Check if llama.cpp and model are available"""
        try:
            # Check llama.cpp binary
            result = subprocess.run([self.llama_binary, '--help'], 
                                  capture_output=True, text=True, timeout=10)
            if result.returncode != 0:
                raise Exception(f"llama.cpp binary not working: {self.llama_binary}")
        except (subprocess.TimeoutExpired, FileNotFoundError) as e:
            raise Exception(f"llama.cpp binary not found or not working: {e}")
        
        if not os.path.exists(self.model_path):
            raise Exception(f"Model file not found: {self.model_path}")
    
    def translate_text(self, text):
        """Translate text using llama.cpp"""
        if not text.strip():
            return text
        
        # Skip FB2/XML tags - don't translate them
        if text.startswith('<') or text.startswith('/') or '=' in text or '|' in text:
            return text
        
        # Prepare prompt
        prompt = self.prompt_template % text.strip()
        
        # Build llama.cpp command
        cmd = [
            self.llama_binary,
            '-m', self.model_path,
            '-p', prompt,
            '-n', str(self.max_tokens),
            '-t', str(self.threads),
            '-c', str(self.context_size),
            '--temp', str(self.temperature),
            '--top-p', '0.9',
            '--top-k', '40',
            '--repeat-penalty', '1.1',
            '--no-display-prompt'
        ]
        
        try:
            # Run llama.cpp
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
            
            if result.returncode != 0:
                print(f"llama.cpp error: {result.stderr}", file=sys.stderr)
                # Fallback to original text on error
                return text
            
            # Extract translation
            translation = result.stdout.strip()
            
            # Clean up the output - remove any prompt echo
            if prompt in translation:
                translation = translation.replace(prompt, '').strip()
            
            # Remove common llama.cpp artifacts
            translation = re.sub(r'^Serbian translation:\s*', '', translation, flags=re.IGNORECASE)
            translation = re.sub(r'^Translation:\s*', '', translation, flags=re.IGNORECASE)
            
            # If result is empty, return original
            if not translation.strip():
                return text
            
            return translation
            
        except subprocess.TimeoutExpired:
            print(f"Translation timeout for text: {text[:50]}...", file=sys.stderr)
            return text
        except Exception as e:
            print(f"Translation error: {e}", file=sys.stderr)
            return text
    
    def translate_paragraphs(self, input_file, output_file):
        """Translate file paragraph by paragraph"""
        with open(input_file, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Split by paragraphs (double newlines)
        paragraphs = content.split('\n\n')
        translated_paragraphs = []
        
        total_paragraphs = len(paragraphs)
        for i, paragraph in enumerate(paragraphs, 1):
            if paragraph.strip():
                print(f"Translating paragraph {i}/{total_paragraphs}...", file=sys.stderr)
                translated_para = self.translate_text(paragraph)
                translated_paragraphs.append(translated_para)
            else:
                translated_paragraphs.append(paragraph)
        
        # Write translated content
        translated_content = '\n\n'.join(translated_paragraphs)
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(translated_content)
        
        return len(content), len(translated_content)

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 translate_llm.py <input> <output>", file=sys.stderr)
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        translator = LLMTranslator()
        translator.check_dependencies()
        
        original_len, translated_len = translator.translate_paragraphs(input_file, output_file)
        
        print(f"LLM translation completed: {original_len} -> {translated_len} characters")
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
EOF

# Set environment variables for the Python script
export LLM_BINARY="$LLAMA_BINARY"
export MODEL_PATH="$MODEL_PATH"
export THREADS="$THREADS"
export CONTEXT_SIZE="$CONTEXT_SIZE"
export TEMPERATURE="$TEMPERATURE"
export MAX_TOKENS="$MAX_TOKENS"

# Run LLM translation
python3 translate_llm.py "$INPUT_FILE" "$OUTPUT_FILE" || error_exit "LLM translation failed"

# Verify output was created
if [[ ! -f "$OUTPUT_FILE" ]]; then
    error_exit "Translation output file not created"
fi

log "LLM translation completed successfully"
log "Output file size: $(wc -c < "$OUTPUT_FILE") bytes"

# Quick verification - check for Serbian characters
if command -v grep >/dev/null 2>&1; then
    serbian_chars=$(grep -o '[љњшђжчћ]' "$OUTPUT_FILE" | wc -l || echo "0")
    log "Serbian Cyrillic characters found: $serbian_chars"
fi

log "LLM-based translation process completed"