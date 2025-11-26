#!/bin/bash
# monitor-deployment.sh - Continuous health monitoring

set -e

CLI_BINARY="${CLI_BINARY:-./build/deployment-cli}"
LOG_FILE="${LOG_FILE:-deployment-health.log}"
INTERVAL="${INTERVAL:-30}"

echo "Starting deployment health monitoring..."
echo "Log file: $LOG_FILE"
echo "Check interval: ${INTERVAL}s"
echo ""

# Create log file
touch "$LOG_FILE"

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    echo "[$timestamp] Checking deployment health..." | tee -a "$LOG_FILE"

    if "$CLI_BINARY" -action status >> "$LOG_FILE" 2>&1; then
        echo "[$timestamp] ✓ Health check passed" | tee -a "$LOG_FILE"
    else
        echo "[$timestamp] ✗ Health check failed" | tee -a "$LOG_FILE"

        # Send alert (customize based on your alerting system)
        echo "[$timestamp] ALERT: Deployment health check failed" >&2
    fi

    echo "" >> "$LOG_FILE"
    sleep "$INTERVAL"
done</content>
</xai:function_call">  
<xai:function_call name="write">
<parameter name="filePath">scripts/check-health.sh