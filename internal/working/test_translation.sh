#!/bin/bash

# Test translation pipeline with DeepSeek
echo "=== Testing Translation Pipeline ==="
echo ""

# Test 1: Language detection
echo "Test 1: Language Detection"
./build/translator -input Books/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi.epub --detect
echo ""

# Test 2: Translation to TXT (simpler than EPUB)
echo "Test 2: Translation to TXT format (first chapter)"
if [ -z "$DEEPSEEK_API_KEY" ]; then
  echo "ERROR: DEEPSEEK_API_KEY must be set"
  exit 1
fi
timeout 120 ./build/translator \
  -input Books/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi.epub \
  -locale sr \
  -provider deepseek \
  -format txt \
  -output /tmp/test_sr.txt 2>&1 | head -100

if [ -f "/tmp/test_sr.txt" ]; then
  echo ""
  echo "✅ TXT file created successfully!"
  echo "File size: $(wc -c < /tmp/test_sr.txt) bytes"
  echo "First 500 chars:"
  head -c 500 /tmp/test_sr.txt
  echo ""
  echo ""

  # Check for Russian text (indicates untranslated)
  if grep -q "[А-Яа-я]" /tmp/test_sr.txt; then
    echo "⚠️  Warning: Russian text detected in output"
    echo "Sample Russian text:"
    grep -o "[А-Яа-я]\{10,\}" /tmp/test_sr.txt | head -3
  else
    echo "✅ No Russian text detected - translation appears complete"
  fi
else
  echo "❌ TXT file not created!"
fi

echo ""
echo "=== Pipeline Test Complete ==="
