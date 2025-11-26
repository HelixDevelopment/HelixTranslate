#!/bin/bash

# Monitor multi-pass translation progress

echo "=== Multi-Pass Translation Progress ==="
echo ""

# Check Pass 1
if [ -f "Books/Translated/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi_sr_pass1.epub" ]; then
    size1=$(ls -lh "Books/Translated/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi_sr_pass1.epub" | awk '{print $5}')
    echo "✓ Pass 1 (Initial Translation): COMPLETE ($size1)"
else
    echo "⏳ Pass 1 (Initial Translation): Not started"
fi

# Check Pass 2
if [ -f "Books/Translated/Stepanova_p2.epub" ]; then
    size2=$(ls -lh "Books/Translated/Stepanova_p2.epub" | awk '{print $5}')
    echo "✓ Pass 2 (Verification): COMPLETE ($size2)"
elif [ -f "/tmp/translation_logs/pass2.log" ]; then
    last_chapter=$(grep "Translating chapter" /tmp/translation_logs/pass2.log | tail -1)
    echo "🔄 Pass 2 (Verification): IN PROGRESS - $last_chapter"
else
    echo "⏳ Pass 2 (Verification): Not started"
fi

# Check Pass 3
if [ -f "Books/Translated/Stepanova_p3_final.epub" ]; then
    size3=$(ls -lh "Books/Translated/Stepanova_p3_final.epub" | awk '{print $5}')
    echo "✓ Pass 3 (Polishing): COMPLETE ($size3)"
elif [ -f "/tmp/translation_logs/pass3.log" ]; then
    last_chapter=$(grep "Translating chapter" /tmp/translation_logs/pass3.log | tail -1)
    echo "🔄 Pass 3 (Polishing): IN PROGRESS - $last_chapter"
else
    echo "⏳ Pass 3 (Polishing): Not started"
fi

echo ""
echo "=== Recent Activity ==="
if [ -f "/tmp/translation_logs/pass2.log" ]; then
    echo "Pass 2 latest:"
    tail -5 /tmp/translation_logs/pass2.log | grep -E "chapter|LLM" | tail -2
fi

if [ -f "/tmp/translation_logs/pass3.log" ]; then
    echo "Pass 3 latest:"
    tail -5 /tmp/translation_logs/pass3.log | grep -E "chapter|LLM" | tail -2
fi
