#!/bin/bash

# Monitor translation progress in real-time

LOG_FILE="/tmp/translation_v3.log"
OUTPUT_FILE="Books/Translated_Son_Nad_Bezdnoyi_sr_v3.epub"

echo "=== Translation Monitor ==="
echo ""
echo "Log file: $LOG_FILE"
echo "Output: $OUTPUT_FILE"
echo ""

# Check if translation is running
if ! pgrep -f "translator.*Stepanova.*multi-llm" > /dev/null; then
    echo "❌ Translation not running"
    echo ""
    echo "Last 20 lines of log:"
    tail -20 "$LOG_FILE" 2>/dev/null || echo "No log file found"
    exit 1
fi

echo "✅ Translation running"
echo ""

# Show progress
echo "=== Current Progress ==="
if [ -f "$LOG_FILE" ]; then
    # Count chapters completed
    CHAPTERS_DONE=$(grep -c "Translating chapter" "$LOG_FILE" 2>/dev/null || echo "0")
    echo "Chapters started: $CHAPTERS_DONE / 38"

    # Count successful translations
    SUCCESSES=$(grep -c "translation_success" "$LOG_FILE" 2>/dev/null || echo "0")
    echo "Successful translations: $SUCCESSES"

    # Count failures
    FAILURES=$(grep -c "multi_llm_warning\|translation_error" "$LOG_FILE" 2>/dev/null || echo "0")
    echo "Failures: $FAILURES"

    # Calculate success rate
    if [ $SUCCESSES -gt 0 ]; then
        TOTAL=$((SUCCESSES + FAILURES))
        SUCCESS_RATE=$((SUCCESSES * 100 / TOTAL))
        echo "Success rate: $SUCCESS_RATE%"
    fi

    echo ""
    echo "=== Recent Activity (last 15 lines) ==="
    tail -15 "$LOG_FILE" | grep -E "chapter|attempt|success|warning|error|completed|Writing" || tail -15 "$LOG_FILE"
else
    echo "Log file not found yet..."
fi

echo ""
echo "=== File Status ==="
if [ -f "$OUTPUT_FILE" ]; then
    echo "✅ Output file exists: $(ls -lh "$OUTPUT_FILE" | awk '{print $5}')"
else
    echo "⏳ Output file not created yet"
fi

echo ""
echo "To monitor continuously: watch -n 5 ./monitor_translation.sh"
