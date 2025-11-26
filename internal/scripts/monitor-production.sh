#!/bin/bash
# monitor-production.sh - Production monitoring and alerting system

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${PROJECT_DIR}/logs/monitor.log"
ALERT_WEBHOOK_URL="${ALERT_WEBHOOK_URL:-}"
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
EMAIL_RECIPIENTS="${EMAIL_RECIPIENTS:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - ${RED}ERROR:${NC} $*" >&2 | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - ${YELLOW}WARNING:${NC} $*" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - ${GREEN}SUCCESS:${NC} $*" | tee -a "$LOG_FILE"
}

# Alert functions
send_alert() {
    local severity="$1"
    local message="$2"
    local details="$3"

    log "Sending $severity alert: $message"

    # Send to webhook if configured
    if [[ -n "$ALERT_WEBHOOK_URL" ]]; then
        curl -X POST "$ALERT_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"severity\":\"$severity\",\"message\":\"$message\",\"details\":\"$details\",\"timestamp\":\"$(date -Iseconds)\"}" \
            --max-time 10 \
            --silent \
            --show-error || log_warning "Failed to send webhook alert"
    fi

    # Send to Slack if configured
    if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
        local color="good"
        [[ "$severity" == "error" ]] && color="danger"
        [[ "$severity" == "warning" ]] && color="warning"

        curl -X POST "$SLACK_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"attachments\":[{\"color\":\"$color\",\"title\":\"Translator Monitor Alert\",\"text\":\"$message\",\"fields\":[{\"title\":\"Details\",\"value\":\"$details\"}]}]}" \
            --max-time 10 \
            --silent \
            --show-error || log_warning "Failed to send Slack alert"
    fi

    # Send email if configured
    if [[ -n "$EMAIL_RECIPIENTS" ]]; then
        echo "Subject: [$severity] Translator Monitor Alert
From: monitor@translator.local
To: $EMAIL_RECIPIENTS

$message

Details:
$details

Timestamp: $(date -Iseconds)
Host: $(hostname)
" | sendmail -t || log_warning "Failed to send email alert"
    fi
}

# Health check functions
check_service_health() {
    local service_name="$1"
    local url="$2"
    local timeout="${3:-10}"

    log "Checking $service_name health at $url"

    if curl -f -k --max-time "$timeout" "$url" >/dev/null 2>&1; then
        log_success "$service_name is healthy"
        return 0
    else
        log_error "$service_name is unhealthy"
        return 1
    fi
}

check_database_health() {
    log "Checking database health"

    # Check if PostgreSQL container is running
    if docker ps --filter "name=postgres" --format "{{.Names}}" | grep -q "translator-postgres"; then
        log_success "PostgreSQL container is running"
    else
        log_error "PostgreSQL container is not running"
        return 1
    fi

    # Try to connect to database
    if docker exec translator-postgres pg_isready -U translator -d translator >/dev/null 2>&1; then
        log_success "Database connection is healthy"
        return 0
    else
        log_error "Database connection failed"
        return 1
    fi
}

check_redis_health() {
    log "Checking Redis health"

    # Check if Redis container is running
    if docker ps --filter "name=redis" --format "{{.Names}}" | grep -q "translator-redis"; then
        log_success "Redis container is running"
    else
        log_error "Redis container is not running"
        return 1
    fi

    # Try Redis ping
    if docker exec translator-redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis connection is healthy"
        return 0
    else
        log_error "Redis connection failed"
        return 1
    fi
}

check_disk_space() {
    local threshold="${1:-90}"

    log "Checking disk space (threshold: ${threshold}%)"

    local usage
    usage=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')

    if [[ $usage -gt $threshold ]]; then
        log_error "Disk usage is ${usage}%, exceeds threshold of ${threshold}%"
        return 1
    else
        log_success "Disk usage is ${usage}%"
        return 0
    fi
}

check_memory_usage() {
    local threshold="${1:-90}"

    log "Checking memory usage (threshold: ${threshold}%)"

    local usage
    usage=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100.0}')

    if [[ $usage -gt $threshold ]]; then
        log_error "Memory usage is ${usage}%, exceeds threshold of ${threshold}%"
        return 1
    else
        log_success "Memory usage is ${usage}%"
        return 0
    fi
}

check_cpu_usage() {
    local threshold="${1:-90}"

    log "Checking CPU usage (threshold: ${threshold}%)"

    local usage
    usage=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}')

    if (( $(echo "$usage > $threshold" | bc -l) )); then
        log_error "CPU usage is ${usage}%, exceeds threshold of ${threshold}%"
        return 1
    else
        log_success "CPU usage is ${usage}%"
        return 0
    fi
}

check_container_logs() {
    local container_name="$1"
    local error_patterns="${2:-"ERROR|FATAL|panic"}"
    local time_window="${3:-300}"  # 5 minutes in seconds

    log "Checking $container_name logs for errors in last ${time_window}s"

    local since_time
    since_time=$(date -d "@$(($(date +%s) - time_window))" -Iseconds 2>/dev/null || date -v-${time_window}S -Iseconds 2>/dev/null || echo "5m ago")

    local error_count
    error_count=$(docker logs --since "$since_time" "$container_name" 2>&1 | grep -c -E "$error_patterns" || true)

    if [[ $error_count -gt 0 ]]; then
        log_warning "Found $error_count error(s) in $container_name logs"
        return 1
    else
        log_success "No errors found in $container_name logs"
        return 0
    fi
}

check_translation_queue() {
    log "Checking translation queue status"

    # Check if there are any queued translations
    local queue_size
    queue_size=$(curl -s -k "https://localhost:8443/health" | jq -r '.queue_size // 0' 2>/dev/null || echo "0")

    if [[ $queue_size -gt 100 ]]; then
        log_warning "Translation queue size is $queue_size, may indicate performance issues"
        return 1
    else
        log_success "Translation queue size is $queue_size"
        return 0
    fi
}

# Performance monitoring
collect_metrics() {
    log "Collecting system metrics"

    local metrics_file="${PROJECT_DIR}/logs/metrics_$(date +%Y%m%d_%H%M%S).json"

    cat > "$metrics_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "system": {
        "cpu_usage": $(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}'),
        "memory_usage": $(free | grep Mem | awk '{printf "%.2f", $3/$2 * 100.0}'),
        "disk_usage": $(df / | tail -1 | awk '{print $5}' | sed 's/%//'),
        "load_average": "$(uptime | awk -F'load average:' '{ print $2 }' | sed 's/^ *//')"
    },
    "containers": {
        "running": $(docker ps --filter "name=translator" --format "{{.Names}}" | wc -l),
        "total": $(docker ps -a --filter "name=translator" --format "{{.Names}}" | wc -l)
    },
    "services": {
        "api_healthy": $(check_service_health "API" "https://localhost:8443/health" >/dev/null 2>&1 && echo "true" || echo "false"),
        "database_healthy": $(check_database_health >/dev/null 2>&1 && echo "true" || echo "false"),
        "redis_healthy": $(check_redis_health >/dev/null 2>&1 && echo "true" || echo "false")
    }
}
EOF

    log "Metrics collected to $metrics_file"
}

# Main monitoring function
run_monitoring() {
    log "Starting production monitoring cycle"

    local alerts_sent=0
    local checks_failed=0

    # System health checks
    if ! check_disk_space 90; then
        send_alert "warning" "High disk usage detected" "Disk usage exceeds 90%"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    if ! check_memory_usage 90; then
        send_alert "warning" "High memory usage detected" "Memory usage exceeds 90%"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    if ! check_cpu_usage 90; then
        send_alert "warning" "High CPU usage detected" "CPU usage exceeds 90%"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    # Service health checks
    if ! check_service_health "API Server" "https://localhost:8443/health"; then
        send_alert "error" "API Server is unhealthy" "Failed to connect to https://localhost:8443/health"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    if ! check_database_health; then
        send_alert "error" "Database is unhealthy" "PostgreSQL connection failed"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    if ! check_redis_health; then
        send_alert "error" "Redis is unhealthy" "Redis connection failed"
        ((alerts_sent++))
        ((checks_failed++))
    fi

    # Container log checks
    for container in translator-api translator-postgres translator-redis; do
        if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
            if ! check_container_logs "$container"; then
                send_alert "warning" "Errors found in $container logs" "Check container logs for details"
                ((alerts_sent++))
            fi
        fi
    done

    # Application-specific checks
    if ! check_translation_queue; then
        send_alert "warning" "Large translation queue detected" "Queue size may indicate performance issues"
        ((alerts_sent++))
    fi

    # Collect metrics
    collect_metrics

    # Summary
    if [[ $checks_failed -gt 0 ]]; then
        log_warning "Monitoring cycle completed with $checks_failed failed checks and $alerts_sent alerts sent"
    else
        log_success "Monitoring cycle completed successfully - all checks passed"
    fi
}

# Continuous monitoring mode
continuous_monitoring() {
    local interval="${1:-300}"  # Default 5 minutes

    log "Starting continuous monitoring (interval: ${interval}s)"

    while true; do
        run_monitoring
        log "Waiting ${interval} seconds until next check..."
        sleep "$interval"
    done
}

# Parse command line arguments
case "${1:-}" in
    "once"|"-o"|"--once")
        run_monitoring
        ;;
    "continuous"|"-c"|"--continuous")
        continuous_monitoring "${2:-300}"
        ;;
    "health"|"-h"|"--health")
        echo "=== Health Check ==="
        check_service_health "API Server" "https://localhost:8443/health" && echo "✓ API healthy" || echo "✗ API unhealthy"
        check_database_health && echo "✓ Database healthy" || echo "✗ Database unhealthy"
        check_redis_health && echo "✓ Redis healthy" || echo "✗ Redis unhealthy"
        check_disk_space 90 && echo "✓ Disk space OK" || echo "✗ Disk space low"
        check_memory_usage 90 && echo "✓ Memory OK" || echo "✗ Memory high"
        check_cpu_usage 90 && echo "✓ CPU OK" || echo "✗ CPU high"
        ;;
    "metrics"|"-m"|"--metrics")
        collect_metrics
        ;;
    *)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  once        - Run monitoring checks once"
        echo "  continuous  - Run continuous monitoring (default 5min intervals)"
        echo "  health      - Quick health check"
        echo "  metrics     - Collect and save metrics"
        echo ""
        echo "Environment variables:"
        echo "  ALERT_WEBHOOK_URL  - Webhook URL for alerts"
        echo "  SLACK_WEBHOOK_URL  - Slack webhook URL for alerts"
        echo "  EMAIL_RECIPIENTS   - Email addresses for alerts"
        echo ""
        echo "Examples:"
        echo "  $0 once"
        echo "  $0 continuous 600  # Check every 10 minutes"
        echo "  ALERT_WEBHOOK_URL=https://hooks.example.com/webhook $0 continuous"
        exit 1
        ;;
esac