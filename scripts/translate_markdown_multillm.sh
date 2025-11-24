#!/bin/bash

# Production Multi-LLM Translation System with llama.cpp
# Russian to Serbian Cyrillic translation with model coordination

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="$SCRIPT_DIR/llama_cpp_production.log"

# Multi-LLM Configuration
declare -a MODELS=(
    "/models/llama-3-8b-instruct.Q4_K_M.gguf"
    "/models/llama-3-8b-instruct.Q5_K_M.gguf" 
    "/models/qwen-7b-instruct.Q4_K_M.gguf"
)
MODEL_CONTEXT=8192
MAX_TOKENS=4096
TEMPERATURE=0.3
TOP_P=0.95
REPEAT_PENALTY=1.1

# Ensemble voting configuration
USE_ENSEMBLE=true
ENSEMBLE_SIZE=3
CONSENSUS_THRESHOLD=0.7

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error_exit() {
    log "ERROR: $*"
    exit 1
}

info() {
    echo "[INFO] $*" | tee -a "$LOG_FILE"
}

warn() {
    echo "[WARN] $*" | tee -a "$LOG_FILE"
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

log "Starting PRODUCTION Multi-LLM Translation with llama.cpp"
log "Input: $INPUT_FILE"
log "Output: $OUTPUT_FILE"

# Verify input file exists
if [[ ! -f "$INPUT_FILE" ]]; then
    error_exit "Input file not found: $INPUT_FILE"
fi

# Parse configuration if provided
if [[ -n "$CONFIG_FILE" && -f "$CONFIG_FILE" ]]; then
    log "Loading configuration from $CONFIG_FILE"
    
    # Use python3 for safe JSON parsing
    if command -v python3 &>/dev/null; then
        MODEL_CONTEXT=$(python3 -c "import json; data=json.load(open('$CONFIG_FILE')); print(data.get('n_ctx', $MODEL_CONTEXT))" 2>/dev/null || echo "$MODEL_CONTEXT")
        MAX_TOKENS=$(python3 -c "import json; data=json.load(open('$CONFIG_FILE')); print(data.get('max_tokens', $MAX_TOKENS))" 2>/dev/null || echo "$MAX_TOKENS")
        TEMPERATURE=$(python3 -c "import json; data=json.load(open('$CONFIG_FILE')); print(data.get('temperature', $TEMPERATURE))" 2>/dev/null || echo "$TEMPERATURE")
        TOP_P=$(python3 -c "import json; data=json.load(open('$CONFIG_FILE')); print(data.get('top_p', $TOP_P))" 2>/dev/null || echo "$TOP_P")
        
        # Parse models array if available
        IFS=$'\n' read -d '' -r -a config_models < <(python3 -c "import json; data=json.load(open('$CONFIG_FILE')); 
models = data.get('models', []);
print('\\n'.join(models))" 2>/dev/null || true)
        
        if [[ ${#config_models[@]} -gt 0 ]]; then
            MODELS=("${config_models[@]}")
        fi
    fi
fi

# Find available models
AVAILABLE_MODELS=()
for model in "${MODELS[@]}"; do
    if [[ -f "$model" ]]; then
        AVAILABLE_MODELS+=("$model")
        log "Found model: $model"
    else
        warn "Model not found: $model"
    fi
done

if [[ ${#AVAILABLE_MODELS[@]} -eq 0 ]]; then
    error_exit "No GGUF model files found. Please download models to /models/ directory"
fi

# Use subset of available models for ensemble
if [[ $USE_ENSEMBLE == true ]]; then
    MODEL_COUNT=$((ENSEMBLE_SIZE < ${#AVAILABLE_MODELS[@]} ? ENSEMBLE_SIZE : ${#AVAILABLE_MODELS[@]}))
    SELECTED_MODELS=("${AVAILABLE_MODELS[@]:0:$MODEL_COUNT}")
    log "Using ensemble of $MODEL_COUNT models"
else
    SELECTED_MODELS=("${AVAILABLE_MODELS[@]:0:1}")
    log "Using single model: ${SELECTED_MODELS[0]}"
fi

# Create virtual environment if needed
VENV_DIR="$HOME/translate_production_env"
if [[ ! -d "$VENV_DIR" ]]; then
    log "Creating production virtual environment at $VENV_DIR"
    python3 -m venv "$VENV_DIR" || error_exit "Failed to create virtual environment"
fi

# Activate virtual environment
log "Activating production virtual environment"
source "$VENV_DIR/bin/activate" || error_exit "Failed to activate virtual environment"

# Upgrade pip and install requirements
log "Upgrading pip and installing production dependencies"
pip install --upgrade pip setuptools wheel --quiet || error_exit "Failed to upgrade pip"

# Install llama-cpp-python with GPU support
log "Installing llama-cpp-python with CUDA support"
pip install llama-cpp-python --extra-index-url https://abetlen.github.io/llama-cpp-python/whl/cu121 || \
pip install llama-cpp-python --extra-index-url https://abetlen.github.io/llama-cpp-python/whl/cpu || \
error_exit "Failed to install llama-cpp-python"

# Install additional dependencies
log "Installing additional dependencies"
pip install numpy --quiet || warn "Failed to install numpy"
pip install sentence-transformers --quiet || warn "Failed to install sentence-transformers"

# Create multi-LLM translation Python script
PYTHON_SCRIPT="$SCRIPT_DIR/multi_llm_translator.py"
cat > "$PYTHON_SCRIPT" << 'EOF'
#!/usr/bin/env python3
import sys
import json
import os
import argparse
import hashlib
from pathlib import Path
from llama_cpp import Llama
from typing import List, Dict, Tuple
import numpy as np

class MultiLLMTranslator:
    def __init__(self, models: List[str], context_size: int, max_tokens: int, 
                 temperature: float, top_p: float, repeat_penalty: float, 
                 use_ensemble: bool, consensus_threshold: float):
        self.models = models
        self.context_size = context_size
        self.max_tokens = max_tokens
        self.temperature = temperature
        self.top_p = top_p
        self.repeat_penalty = repeat_penalty
        self.use_ensemble = use_ensemble
        self.consensus_threshold = consensus_threshold
        self.loaded_models = {}
        
    def load_models(self):
        """Load all specified models"""
        print(f"Loading {len(self.models)} model(s)...")
        
        for i, model_path in enumerate(self.models):
            try:
                print(f"Loading model {i+1}/{len(self.models)}: {model_path}")
                model = Llama(
                    model_path=model_path,
                    n_ctx=self.context_size,
                    n_gpu_layers=-1,  # Use all GPU layers
                    verbose=False,
                    seed=42  # Deterministic output
                )
                self.loaded_models[model_path] = model
                print(f"Successfully loaded: {model_path}")
            except Exception as e:
                print(f"Failed to load model {model_path}: {e}")
                sys.exit(1)
    
    def translate_with_model(self, model: Llama, text: str, model_path: str) -> str:
        """Translate text using a single model"""
        
        system_prompt = """You are a professional translator specializing in Russian to Serbian translation. 
Translate the given Russian text to Serbian Cyrillic while maintaining:
1. Original meaning, context, and nuances
2. Cultural idioms and expressions  
3. Professional literary style and tone
4. Markdown formatting and structure
5. Character voice and narrative flow

Rules:
- Translate accurately to Serbian (Cyrillic script)
- Preserve all markdown formatting
- Maintain paragraph structure
- Keep proper names and technical terms consistent
- Use natural Serbian phrasing
- Return ONLY the translated Serbian text"""

        prompt = f"""<|begin_of_text|><|start_header_id|>system<|end_header_id|>
{system_prompt}<|eot_id|><|start_header_id|>user<|end_header_id|>

Translate the following Russian text to Serbian Cyrillic:

{text}<|eot_id|><|start_header_id|>assistant<|end_header_id|>"""

        try:
            output = model(
                prompt,
                max_tokens=self.max_tokens,
                temperature=self.temperature,
                top_p=self.top_p,
                repeat_penalty=self.repeat_penalty,
                stop=['<|eot_id|>', 'Russian:', 'Original:'],
                echo=False
            )
            
            return output['choices'][0]['text'].strip()
            
        except Exception as e:
            print(f"Translation error with model {model_path}: {e}")
            return text  # Fallback to original text
    
    def calculate_similarity(self, text1: str, text2: str) -> float:
        """Calculate similarity between two translated texts"""
        # Simple word overlap similarity (could be enhanced with embeddings)
        words1 = set(text1.lower().split())
        words2 = set(text2.lower().split())
        
        if not words1 or not words2:
            return 0.0
            
        intersection = words1.intersection(words2)
        union = words1.union(words2)
        
        return len(intersection) / len(union)
    
    def consensus_translation(self, translations: List[Tuple[str, str]]) -> str:
        """Find consensus translation from multiple models"""
        
        if len(translations) == 1:
            return translations[0][1]
        
        # Calculate similarity matrix
        n = len(translations)
        similarity_matrix = np.zeros((n, n))
        
        for i in range(n):
            for j in range(n):
                if i == j:
                    similarity_matrix[i][j] = 1.0
                else:
                    similarity_matrix[i][j] = self.calculate_similarity(
                        translations[i][1], translations[j][1]
                    )
        
        # Calculate consensus scores
        consensus_scores = []
        for i, (model_path, translation) in enumerate(translations):
            score = np.mean(similarity_matrix[i])
            consensus_scores.append((score, translation, model_path))
        
        # Sort by consensus score
        consensus_scores.sort(reverse=True)
        
        best_score, best_translation, best_model = consensus_scores[0]
        
        print(f"Consensus analysis:")
        for i, (score, trans, model) in enumerate(consensus_scores[:3]):
            model_name = os.path.basename(model)
            print(f"  {i+1}. {model_name}: {score:.3f}")
        
        if best_score >= self.consensus_threshold:
            print(f"Selected consensus translation (score: {best_score:.3f})")
            return best_translation
        else:
            print(f"Low consensus score ({best_score:.3f}), using first model result")
            return translations[0][1]
    
    def translate_text(self, text: str) -> str:
        """Translate text using multiple models with consensus"""
        
        # Split text into chunks if too large
        max_chunk_length = self.max_tokens // 4  # Rough estimate
        if len(text) > max_chunk_length:
            return self.translate_chunks(text)
        
        # Translate with each model
        translations = []
        for model_path in self.models:
            if model_path in self.loaded_models:
                model = self.loaded_models[model_path]
                translation = self.translate_with_model(model, text, model_path)
                translations.append((model_path, translation))
                print(f"Translation completed with {os.path.basename(model_path)}")
        
        # Use consensus if multiple models
        if self.use_ensemble and len(translations) > 1:
            return self.consensus_translation(translations)
        else:
            return translations[0][1]
    
    def translate_chunks(self, text: str) -> str:
        """Translate long text by chunking"""
        
        # Split by paragraphs to maintain context
        paragraphs = text.split('\n\n')
        chunks = []
        current_chunk = ""
        
        for paragraph in paragraphs:
            if len(current_chunk) + len(paragraph) + 2 < self.max_tokens // 4:
                current_chunk += paragraph + '\n\n'
            else:
                if current_chunk.strip():
                    chunks.append(current_chunk.strip())
                current_chunk = paragraph + '\n\n'
        
        if current_chunk.strip():
            chunks.append(current_chunk.strip())
        
        print(f"Translating {len(chunks)} chunks...")
        
        # Translate each chunk
        translated_chunks = []
        for i, chunk in enumerate(chunks):
            print(f"Translating chunk {i+1}/{len(chunks)}...")
            translated_chunk = self.translate_text(chunk)
            translated_chunks.append(translated_chunk)
        
        return '\n\n'.join(translated_chunks)

def main():
    parser = argparse.ArgumentParser(description='Multi-LLM Translation with llama.cpp')
    parser.add_argument('--input', required=True, help='Input markdown file')
    parser.add_argument('--output', required=True, help='Output markdown file')
    parser.add_argument('--models', nargs='+', required=True, help='Model file paths')
    parser.add_argument('--context', type=int, default=8192, help='Context size')
    parser.add_argument('--max-tokens', type=int, default=4096, help='Maximum tokens')
    parser.add_argument('--temperature', type=float, default=0.3, help='Temperature')
    parser.add_argument('--top-p', type=float, default=0.95, help='Top P')
    parser.add_argument('--repeat-penalty', type=float, default=1.1, help='Repeat penalty')
    parser.add_argument('--ensemble', action='store_true', help='Use ensemble translation')
    parser.add_argument('--consensus-threshold', type=float, default=0.7, help='Consensus threshold')
    
    args = parser.parse_args()
    
    # Create translator
    translator = MultiLLMTranslator(
        models=args.models,
        context_size=args.context,
        max_tokens=args.max_tokens,
        temperature=args.temperature,
        top_p=args.top_p,
        repeat_penalty=args.repeat_penalty,
        use_ensemble=args.ensemble,
        consensus_threshold=args.consensus_threshold
    )
    
    # Load models
    translator.load_models()
    
    # Read input text
    with open(args.input, 'r', encoding='utf-8') as f:
        input_text = f.read()
    
    print(f"Translating {len(input_text)} characters...")
    
    # Translate text
    translated_text = translator.translate_text(input_text)
    
    # Write output
    with open(args.output, 'w', encoding='utf-8') as f:
        f.write(translated_text)
    
    print(f"Translation completed: {args.output}")

if __name__ == "__main__":
    main()
EOF

# Execute translation with ensemble
log "Starting multi-LLM ensemble translation"
python3 "$PYTHON_SCRIPT" \
    --input "$INPUT_FILE" \
    --output "$OUTPUT_FILE" \
    --models "${SELECTED_MODELS[@]}" \
    --context "$MODEL_CONTEXT" \
    --max-tokens "$MAX_TOKENS" \
    --temperature "$TEMPERATURE" \
    --top-p "$TOP_P" \
    --repeat-penalty "$REPEAT_PENALTY" \
    --ensemble \
    --consensus-threshold "$CONSENSUS_THRESHOLD" || error_exit "Multi-LLM translation failed"

# Verify output file was created
if [[ ! -f "$OUTPUT_FILE" ]]; then
    error_exit "Output file not created: $OUTPUT_FILE"
fi

# Get file sizes for reporting
INPUT_SIZE=$(stat -c%s "$INPUT_FILE" 2>/dev/null || stat -f%z "$INPUT_FILE" 2>/dev/null || echo "0")
OUTPUT_SIZE=$(stat -c%s "$OUTPUT_FILE" 2>/dev/null || stat -f%z "$OUTPUT_FILE" 2>/dev/null || echo "0")

log "Multi-LLM translation completed successfully"
log "Input size: $INPUT_SIZE bytes"
log "Output size: $OUTPUT_SIZE bytes"
log "Models used: ${#SELECTED_MODELS[@]} model(s)"
log "Ensemble: $USE_ENSEMBLE"

# Generate translation statistics
STATS_FILE="${OUTPUT_FILE}.stats"
cat > "$STATS_FILE" << EOF
{
  "input_file": "$INPUT_FILE",
  "output_file": "$OUTPUT_FILE",
  "input_size_bytes": $INPUT_SIZE,
  "output_size_bytes": $OUTPUT_SIZE,
  "models_used": [$(printf '"%s",' "${SELECTED_MODELS[@]}" | sed 's/,$//')],
  "context_size": $MODEL_CONTEXT,
  "max_tokens": $MAX_TOKENS,
  "temperature": $TEMPERATURE,
  "top_p": $TOP_P,
  "repeat_penalty": $REPEAT_PENALTY,
  "ensemble_enabled": $USE_ENSEMBLE,
  "ensemble_size": ${#SELECTED_MODELS[@]},
  "consensus_threshold": $CONSENSUS_THRESHOLD,
  "timestamp": "$(date -Iseconds)"
}
EOF

log "Translation statistics saved to: $STATS_FILE"

# Cleanup
rm -f "$PYTHON_SCRIPT"

log "Production multi-LLM translation system completed successfully"