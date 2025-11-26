#!/bin/bash

# Monitor llama.cpp translation progress
# Usage: ./monitor_llamacpp_translation.sh [log_file]

LOG_FILE="${1:-translation_llamacpp.log}"

if [ ! -f "$LOG_FILE" ]; then
    echo "Error: Log file '$LOG_FILE' not found"
    exit 1
fi

echo "=== LLama.cpp Translation Monitor ==="
echo "Monitoring: $LOG_FILE"
echo "Press Ctrl+C to stop monitoring"
echo ""

# Function to extract stats
print_stats() {
    echo "=== Translation Progress ==="
    echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # Count completed inferences
    COMPLETED=$(grep -c "Inference complete" "$LOG_FILE")
    echo "Completed inferences: $COMPLETED"

    # Get current chapter if available
    CURRENT_CHAPTER=$(grep "translation_progress" "$LOG_FILE" | tail -1)
    if [ -n "$CURRENT_CHAPTER" ]; then
        echo "Latest progress: $CURRENT_CHAPTER"
    fi

    # Calculate average speed
    AVG_SPEED=$(grep "Inference complete" "$LOG_FILE" | \
                awk -F': ' '{print $2}' | \
                awk '{sum+=$1; count++} END {if(count>0) printf "%.1f", sum/count}')
    if [ -n "$AVG_SPEED" ]; then
        echo "Average speed: ${AVG_SPEED} tokens/sec"
    fi

    # Show last few lines
    echo ""
    echo "=== Recent Activity ==="
    tail -5 "$LOG_FILE"

    # Check process status
    echo ""
    LLAMA_PID=$(ps aux | grep llama-cli | grep -v grep | awk '{print $2}')
    if [ -n "$LLAMA_PID" ]; then
        echo "Status: Running (PID: $LLAMA_PID)"
        ps -p "$LLAMA_PID" -o %cpu,%mem,rss,etime | tail -1 | \
            awk '{printf "CPU: %s%% | MEM: %s%% | RAM: %.1f GB | Uptime: %s\n", $1, $2, $3/1024/1024, $4}'
    else
        echo "Status: Not running or completed"
    fi

    echo ""
    echo "================================"
}

# Main monitoring loop
while true; do
    clear
    print_stats
    sleep 30
done
