#!/bin/bash

echo "=== TRANSLATION SYSTEM DIAGNOSTIC ==="
echo "Date: $(date)"
echo

# Test 1: Basic connectivity
echo "1. Testing SSH connectivity..."
timeout 10 ssh milosvasic@thinker.local "echo 'SSH connection OK'" 2>/dev/null
if [ $? -eq 0 ]; then
    echo "✓ SSH connection working"
else
    echo "✗ SSH connection failed"
    exit 1
fi

# Test 2: Check remote directory structure
echo
echo "2. Checking remote directory structure..."
ssh milosvasic@thinker.local "ls -la /tmp/translate-ssh/ 2>/dev/null || echo 'Directory not found'"

# Test 3: Check llama.cpp binary
echo
echo "3. Checking llama.cpp..."
ssh milosvasic@thinker.local "find /home/milosvasic/llama.cpp -name 'main' -o -name 'llama' -type f 2>/dev/null || echo 'llama.cpp binary not found'"
ssh milosvasic@thinker.local "ls -la /tmp/translate-ssh/llama.cpp 2>/dev/null || echo 'uploaded llama.cpp not found'"

# Test 4: Check model
echo
echo "4. Checking models..."
ssh milosvasic@thinker.local "ls -la /home/milosvasic/models/*.gguf 2>/dev/null || echo 'No models found'"

# Test 5: Test Python
echo
echo "5. Testing Python..."
ssh milosvasic@thinker.local "cd /tmp/translate-ssh && python3 --version 2>/dev/null || echo 'Python3 not available'"

# Test 6: Simple llama.cpp test
echo
echo "6. Testing simple llama.cpp execution..."
ssh milosvasic@thinker.local "cd /tmp/translate-ssh && timeout 30 python3 -c \"
import subprocess
import sys
try:
    result = subprocess.run(['/home/milosvasic/llama.cpp', '--help'], capture_output=True, text=True, timeout=10)
    print('llama.cpp help output length:', len(result.stdout))
    print('llama.cpp stderr length:', len(result.stderr))
    print('llama.cpp exit code:', result.returncode)
except Exception as e:
    print('Error:', str(e))
    sys.exit(1)
\" 2>/dev/null || echo 'llama.cpp test failed'"

echo
echo "=== DIAGNOSTIC COMPLETE ==="