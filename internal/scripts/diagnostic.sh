#!/bin/bash

# Diagnostic script to check LLM setup on remote worker
set -euo pipefail

echo "=== LLM Environment Diagnostic ==="
echo "Current directory: $(pwd)"
echo "Files in current directory:"
ls -la

echo -e "\n=== Checking llama.cpp ==="
if command -v llama-cli >/dev/null 2>&1; then
    echo "✅ llama-cli found: $(which llama-cli)"
    echo "llama-cli version:"
    llama-cli --help 2>&1 | head -5
else
    echo "❌ llama-cli NOT found"
    echo "Available binaries in /usr/local/bin:"
    ls /usr/local/bin/ | grep -i llama || echo "No llama binaries found"
    echo "Available binaries in /usr/bin:"
    ls /usr/bin/ | grep -i llama || echo "No llama binaries found"
fi

echo -e "\n=== Checking Model ==="
MODEL_PATH="/home/milosvasic/llama.cpp/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf"
if [[ -f "$MODEL_PATH" ]]; then
    echo "✅ Model found: $MODEL_PATH"
    echo "Model size: $(du -h "$MODEL_PATH" | cut -f1)"
else
    echo "❌ Model NOT found: $MODEL_PATH"
    echo "Checking alternative paths..."
    for path in /home/milosvasic/llama.cpp/models/ /tmp/models/ /opt/llama.cpp/models/; do
        if [[ -d "$path" ]]; then
            echo "Found model directory: $path"
            ls -la "$path"
        fi
    done
fi

echo -e "\n=== Checking LLM Translation Script ==="
if [[ -f "./llm_translation.sh" ]]; then
    echo "✅ llm_translation.sh found"
    echo "File permissions:"
    ls -la llm_translation.sh
else
    echo "❌ llm_translation.sh NOT found"
fi

echo -e "\n=== Testing Simple Translation ==="
echo "Creating test input..."
echo "Простой текст для перевода" > test_input.txt

echo "Running llm_translation.sh on test input..."
timeout 30 ./llm_translation.sh test_input.txt test_output.txt config.json || echo "Script failed or timed out"

if [[ -f "test_output.txt" ]]; then
    echo "✅ Output file created"
    echo "Output content:"
    cat test_output.txt
else
    echo "❌ No output file created"
fi

echo -e "\n=== System Information ==="
echo "RAM info:"
free -h || echo "free command not available"
echo "CPU info:"
nproc || echo "nproc command not available"
echo "Python version:"
python3 --version || echo "python3 not available"

echo -e "\n=== Diagnostic Complete ==="