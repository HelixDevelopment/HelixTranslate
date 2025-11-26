#!/bin/bash

# Auto-start Pass 3 when Pass 2 completes
# This script monitors Pass 2 and automatically starts Pass 3 polishing

echo "Monitoring Pass 2 completion..."

# Wait for Pass 2 output file to exist and be complete
while true; do
    if [ -f "Books/Translated/Stepanova_p2.epub" ]; then
        # Check if translation completed in log
        if grep -q "Translation Statistics" /tmp/translation_logs/pass2.log 2>/dev/null; then
            echo "✓ Pass 2 completed! Starting Pass 3..."

            # Start Pass 3 with Zhipu for polishing (or DeepSeek as fallback)
            # API keys should be set in environment before running this script
            # export ZHIPU_API_KEY="your-key-here"
            # export DEEPSEEK_API_KEY="your-key-here"

            # Try Zhipu first for literary polishing
            echo "Using Zhipu AI for Pass 3 polishing..."
            nohup ./build/translator \
                -input "Books/Translated/Stepanova_p2.epub" \
                -output "Books/Translated/Stepanova_p3_final.epub" \
                -locale sr \
                -provider zhipu \
                -format epub \
                -script cyrillic \
                > /tmp/translation_logs/pass3.log 2>&1 &

            echo "Pass 3 started! PID: $!"
            echo "Monitor with: tail -f /tmp/translation_logs/pass3.log"
            exit 0
        fi
    fi

    # Check progress every 60 seconds
    sleep 60

    # Show current progress
    if [ -f "/tmp/translation_logs/pass2.log" ]; then
        last_chapter=$(grep "Translating chapter" /tmp/translation_logs/pass2.log | tail -1)
        echo "[$(date +%H:%M:%S)] Pass 2 status: $last_chapter"
    fi
done
