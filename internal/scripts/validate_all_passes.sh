#!/bin/bash

# Comprehensive validation of all translation passes

echo "========================================="
echo "Multi-Pass Translation Validation"
echo "========================================="
echo ""

PASS1="Books/Translated/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi_sr_pass1.epub"
PASS2="Books/Translated/Stepanova_p2.epub"
PASS3="Books/Translated/Stepanova_p3_final.epub"

# Validate Pass 1
echo "=== Pass 1: Initial Translation ==="
if [ -f "$PASS1" ]; then
    echo "✓ File exists: $PASS1"
    echo "  Size: $(ls -lh "$PASS1" | awk '{print $5}')"
    echo ""
    echo "Verification results:"
    go run /tmp/verify_translation.go "$PASS1" 2>&1 | grep -v "^$"
    echo ""
else
    echo "❌ Pass 1 file not found!"
    echo ""
fi

# Validate Pass 2
echo "=== Pass 2: Verification ==="
if [ -f "$PASS2" ]; then
    echo "✓ File exists: $PASS2"
    echo "  Size: $(ls -lh "$PASS2" | awk '{print $5}')"
    echo ""
    echo "Verification results:"
    go run /tmp/verify_translation.go "$PASS2" 2>&1 | grep -v "^$"
    echo ""
else
    echo "⏳ Pass 2 not yet complete"
    if [ -f "/tmp/translation_logs/pass2.log" ]; then
        last_chapter=$(grep "Translating chapter" /tmp/translation_logs/pass2.log | tail -1)
        echo "  Current status: $last_chapter"
    fi
    echo ""
fi

# Validate Pass 3
echo "=== Pass 3: Polishing ==="
if [ -f "$PASS3" ]; then
    echo "✓ File exists: $PASS3"
    echo "  Size: $(ls -lh "$PASS3" | awk '{print $5}')"
    echo ""
    echo "Verification results:"
    go run /tmp/verify_translation.go "$PASS3" 2>&1 | grep -v "^$"
    echo ""
else
    echo "⏳ Pass 3 not yet started"
    if [ -f "/tmp/translation_logs/pass3.log" ]; then
        last_chapter=$(grep "Translating chapter" /tmp/translation_logs/pass3.log | tail -1)
        echo "  Current status: $last_chapter"
    fi
    echo ""
fi

# Summary
echo "========================================="
echo "Summary"
echo "========================================="

count=0
if [ -f "$PASS1" ]; then count=$((count+1)); fi
if [ -f "$PASS2" ]; then count=$((count+1)); fi
if [ -f "$PASS3" ]; then count=$((count+1)); fi

echo "Completed passes: $count/3"

if [ $count -eq 3 ]; then
    echo ""
    echo "✅ All passes complete!"
    echo ""
    echo "Final output: $PASS3"
    echo ""
    echo "Quality comparison:"
    echo "  Pass 1 (Multi-LLM):  $(ls -lh "$PASS1" | awk '{print $5}')"
    echo "  Pass 2 (DeepSeek):   $(ls -lh "$PASS2" | awk '{print $5}')"
    echo "  Pass 3 (Zhipu):      $(ls -lh "$PASS3" | awk '{print $5}')"
    echo ""
    echo "Recommendation: Review Pass 3 for publication-ready quality"
elif [ $count -eq 2 ]; then
    echo "⏳ Pass 3 in progress or pending"
elif [ $count -eq 1 ]; then
    echo "🔄 Pass 2 in progress"
else
    echo "❌ Translation pipeline not started"
fi
