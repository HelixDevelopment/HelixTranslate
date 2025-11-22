# Health Monitoring and Automatic Recovery

This document describes the comprehensive health monitoring and automatic recovery system for the distributed Universal Ebook Translator deployment.

## Overview

The health monitoring system provides:

- **Real-time health checks** for all deployed instances
- **Automatic recovery** from failures
- **Load balancing** based on health status
- **Alerting and notifications** for critical issues
- **Performance metrics** and trend analysis
- **Zero-downtime maintenance** capabilities

## Health Check Architecture

### Health Check Types

#### 1. Container Health Checks
- **Docker health checks** built into container definitions
- **Application health endpoints** (`/health`, `/ready`, `/metrics`)
- **Database connectivity** checks
- **External service dependencies**

#### 2. System Health Checks
- **CPU and memory usage** monitoring
- **Disk space availability**
- **Network connectivity** validation
- **SSL certificate expiration**

#### 3. Application Health Checks
- **API endpoint responsiveness**
- **Queue processing status**
- **Background job health**
- **Cache hit rates**

### Health Check Configuration

```yaml
# docker-compose.yml health checks
services:
  translator-main:
    healthcheck:
      test: ["CMD", "curl", "-f", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  postgres:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U translator -d translator"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Monitoring Components

### Health Monitor Service

```go
type HealthMonitor struct {
    instances   map[string]*InstanceHealth
    checkers    []HealthChecker
    alertManager *AlertManager
    metrics     *MetricsCollector
    mu          sync.RWMutex
}

type InstanceHealth struct {
    ID          string
    Status      HealthStatus
    LastCheck   time.Time
    Checks      map[string]*HealthCheckResult
    Score       float64
    IncidentCount int
}

type HealthCheckResult struct {
    Name        string
    Status      HealthStatus
    Duration    time.Duration
    Error       string
    Timestamp   time.Time
    Metadata    map[string]interface{}
}
```

### Health Status Definitions

```go
type HealthStatus int

const (
    StatusUnknown HealthStatus = iota
    StatusHealthy
    StatusDegraded
    StatusUnhealthy
    StatusDown
)
```

## Automatic Recovery Mechanisms

### Recovery Strategies

#### 1. Container Restart
- **Automatic restart** on container failure
- **Exponential backoff** for repeated failures
- **Maximum retry limits** to prevent infinite loops

#### 2. Instance Replacement
- **Deploy new instance** when health checks fail
- **Load balancing updates** to route traffic away
- **Graceful shutdown** of unhealthy instances

#### 3. Service Scaling
- **Horizontal scaling** based on load and health
- **Vertical scaling** for resource-intensive tasks
- **Auto-scaling policies** based on metrics

### Recovery Configuration

```json
{
  "recovery": {
    "strategies": {
      "container_restart": {
        "enabled": true,
        "max_attempts": 3,
        "backoff_multiplier": 2.0,
        "max_backoff": "300s"
      },
      "instance_replacement": {
        "enabled": true,
        "cooldown_period": "60s",
        "max_concurrent_replacements": 2
      },
      "service_scaling": {
        "enabled": true,
        "scale_up_threshold": 80,
        "scale_down_threshold": 30,
        "cooldown_period": "300s"
      }
    }
  }
}
```

## Alerting System

### Alert Types

#### 1. Health Alerts
- **Instance down** - Critical alert
- **Degraded performance** - Warning alert
- **High error rates** - Error alert

#### 2. System Alerts
- **Resource exhaustion** (CPU, memory, disk)
- **Network issues** (connectivity, latency)
- **Security events** (failed authentications)

#### 3. Business Alerts
- **Queue backlog** growing
- **Translation failures** increasing
- **API rate limits** exceeded

### Alert Channels

```go
type AlertChannel interface {
    Send(alert *Alert) error
}

type AlertChannels struct {
    Email     *EmailChannel
    Slack     *SlackChannel
    Webhook   *WebhookChannel
    PagerDuty *PagerDutyChannel
}
```

### Alert Configuration

```yaml
alerting:
  channels:
    - type: email
      recipients: ["admin@example.com", "devops@example.com"]
      severity: [critical, error]

    - type: slack
      webhook_url: "https://hooks.slack.com/services/..."
      channel: "#alerts"
      severity: [warning, error, critical]

    - type: pagerduty
      integration_key: "your-pagerduty-key"
      severity: [critical]
```

## API Communication Logging

### Retrofit-Style Logging

The system implements comprehensive API communication logging between distributed nodes using a Retrofit-style format for impeccable readability and debugging.

#### Log Format

**Outgoing Requests:**
```
2025/11/22 17:28:39.302 --> GET https://thinker.local:8443/api/v1/providers
```

**Incoming Responses:**
```
2025/11/22 17:28:39.321 <-- 200 OK https://thinker.local:8443/api/v1/providers (19ms, 811-byte body)
```

**Failed Requests:**
```
2025/11/22 17:28:39.302 --> GET https://thinker.local:8443/api/v1/providers
2025/11/22 17:28:39.321 <-- HTTP FAILED: connection refused
```

#### Log File Location

- **File:** `workers_api_communication.log`
- **Location:** Project root directory
- **Format:** Human-readable text (one line per request/response)

#### Analyzing API Communications

```bash
# View recent API communications
tail -20 workers_api_communication.log

# Count requests by endpoint
grep " --> " workers_api_communication.log | cut -d' ' -f6 | sort | uniq -c | sort -nr

# Find slow requests (>1000ms)
grep " <-- " workers_api_communication.log | grep -E "\([0-9]{4,}ms|[0-9]+s" | sort -k5 -n

# Check for failed requests
grep "HTTP FAILED" workers_api_communication.log

# Monitor real-time API communications
tail -f workers_api_communication.log
```

#### Log Fields

| Field | Description | Example |
|-------|-------------|---------|
| Timestamp | Request/response time | `2025/11/22 17:28:39.302` |
| Direction | `--> ` (outgoing) or `<-- ` (incoming) | `--> ` |
| Method | HTTP method | `GET` |
| Status | HTTP status code and text | `200 OK` |
| URL | Full request URL | `https://thinker.local:8443/api/v1/providers` |
| Duration | Response time | `(19ms` |
| Body Size | Response body size | `811-byte body)` |
| Error | Failure reason (if applicable) | `connection refused` |

#### Integration Points

API logging is automatically enabled when:
- Distributed mode is enabled in configuration
- Worker discovery operations are performed
- Translation requests are distributed across workers
- Health checks and status queries occur

## Metrics Collection

### System Metrics

```go
type SystemMetrics struct {
    CPUUsage        float64
    MemoryUsage     float64
    DiskUsage       float64
    NetworkRX       int64
    NetworkTX       int64
    LoadAverage     float64
    Uptime          time.Duration
}
```

### Application Metrics

```go
type ApplicationMetrics struct {
    ActiveConnections   int
    RequestsPerSecond   float64
    AverageResponseTime time.Duration
    ErrorRate           float64
    QueueDepth          int
    CacheHitRate        float64
}
```

### Custom Metrics

```go
type CustomMetrics struct {
    TranslationsCompleted int64
    AverageTranslationTime time.Duration
    APIKeysUsed           int
    LanguagesProcessed    map[string]int64
}
```

## Monitoring Dashboard

### Real-time Dashboard

```bash
# Start monitoring dashboard
./build/deployment-cli -action monitor

# Or access via web interface
open http://localhost:8080/monitoring
```

### Dashboard Features

- **Live health status** of all instances
- **Real-time metrics** graphs
- **Alert history** and active alerts
- **Performance trends** over time
- **Log streaming** from instances
- **Configuration management**

### Dashboard API

```http
GET /api/v1/monitoring/health
GET /api/v1/monitoring/metrics
GET /api/v1/monitoring/alerts
GET /api/v1/monitoring/logs
```

## Automated Maintenance

### Maintenance Windows

```json
{
  "maintenance": {
    "windows": [
      {
        "name": "nightly-maintenance",
        "schedule": "0 2 * * *",
        "duration": "2h",
        "services": ["postgres", "redis"],
        "actions": ["backup", "optimize", "restart"]
      }
    ]
  }
}
```

### Maintenance Operations

#### 1. Database Maintenance
- **Automatic backups** during maintenance windows
- **Index optimization** and statistics updates
- **Vacuum operations** for PostgreSQL
- **Cache clearing** and memory defragmentation

#### 2. Log Rotation
- **Automatic log rotation** based on size/time
- **Compression** of old log files
- **Archive storage** to remote locations
- **Log analysis** and reporting

#### 3. Certificate Management
- **SSL certificate renewal** before expiration
- **Automatic deployment** of new certificates
- **Service restart** with zero downtime
- **Certificate validation** and monitoring

## Performance Optimization

### Auto-tuning

#### 1. Resource Allocation
- **Dynamic CPU/memory limits** based on usage
- **Workload-based scaling** decisions
- **Resource quota management**

#### 2. Connection Pooling
- **Database connection optimization**
- **HTTP client connection reuse**
- **Cache connection management**

#### 3. Query Optimization
- **Slow query detection** and alerting
- **Automatic index recommendations**
- **Query plan analysis**

### Performance Baselines

```json
{
  "baselines": {
    "response_time_p95": "100ms",
    "error_rate": "0.1%",
    "cpu_usage": "70%",
    "memory_usage": "80%",
    "disk_usage": "85%"
  }
}
```

## Incident Response

### Automated Incident Response

```yaml
incident_response:
  rules:
    - name: high_error_rate
      condition: "error_rate > 5%"
      actions:
        - type: alert
          severity: critical
        - type: scale_up
          service: worker
          replicas: 2
        - type: restart
          service: affected

    - name: instance_down
      condition: "health_status == 'down'"
      actions:
        - type: alert
          severity: critical
        - type: replace_instance
          cooldown: 30s
        - type: notify
          channels: [slack, email]
```

### Incident Playbooks

#### Instance Failure Playbook
1. **Detection**: Health check failure
2. **Alert**: Immediate notification to on-call engineer
3. **Diagnosis**: Automatic log collection and analysis
4. **Recovery**: Deploy replacement instance
5. **Verification**: Health checks pass
6. **Post-mortem**: Generate incident report

#### Performance Degradation Playbook
1. **Detection**: Performance metrics exceed thresholds
2. **Alert**: Warning notification
3. **Analysis**: Identify bottleneck (CPU, memory, I/O)
4. **Mitigation**: Scale resources or optimize queries
5. **Recovery**: Performance returns to normal
6. **Prevention**: Implement permanent fixes

## Testing and Validation

### Health Check Testing

```bash
# Test health checks
go test ./pkg/deployment/health_test.go -v

# Integration testing
go test ./test/deployment/health_integration_test.go -v
```

### Recovery Testing

```bash
# Test automatic recovery
go test ./test/deployment/recovery_test.go -v

# Chaos engineering
go test ./test/deployment/chaos_test.go -v
```

### Load Testing

```bash
# Performance testing
go test ./test/deployment/performance_test.go -v -bench=.

# Stress testing
go test ./test/deployment/stress_test.go -v
```

## Monitoring Scripts

### Health Check Script

```bash
#!/bin/bash
# check-health.sh - Comprehensive health checking

SERVICES=("translator-main" "translator-worker-1" "postgres" "redis")
HEALTH_ENDPOINTS=(
    "https://localhost:8443/health"
    "https://localhost:8444/health"
    "http://localhost:5432/health"
    "http://localhost:6379/health"
)

echo "=== Health Check Report ==="
echo "Timestamp: $(date)"
echo ""

all_healthy=true

for i in "${!SERVICES[@]}"; do
    service="${SERVICES[$i]}"
    endpoint="${HEALTH_ENDPOINTS[$i]}"

    echo -n "Checking $service ($endpoint)... "

    if curl -f -k --max-time 10 "$endpoint" >/dev/null 2>&1; then
        echo "✓ HEALTHY"
    else
        echo "✗ UNHEALTHY"
        all_healthy=false
    fi
done

echo ""
if $all_healthy; then
    echo "✓ All services are healthy"
    exit 0
else
    echo "✗ Some services are unhealthy"
    exit 1
fi
```

### Metrics Collection Script

```bash
#!/bin/bash
# collect-metrics.sh - Collect system and application metrics

OUTPUT_FILE="${1:-metrics.json}"
INTERVAL="${2:-60}"

echo "Collecting metrics every ${INTERVAL}s to $OUTPUT_FILE"

while true; do
    timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    # System metrics
    cpu_usage=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}')
    memory_usage=$(free | grep Mem | awk '{printf "%.2f", $3/$2 * 100.0}')

    # Docker metrics
    container_count=$(docker ps | wc -l)
    container_count=$((container_count - 1))  # Subtract header

    # Application metrics (example)
    api_requests=$(curl -s -k https://localhost:8443/metrics | jq -r '.requests_total // 0' 2>/dev/null || echo "0")

    # Create metrics JSON
    metrics=$(cat <<EOF
{
  "timestamp": "$timestamp",
  "system": {
    "cpu_usage_percent": $cpu_usage,
    "memory_usage_percent": $memory_usage
  },
  "docker": {
    "running_containers": $container_count
  },
  "application": {
    "api_requests_total": $api_requests
  }
}
EOF
)

    echo "$metrics" >> "$OUTPUT_FILE"
    echo "[$timestamp] Metrics collected"

    sleep "$INTERVAL"
done
```

## Summary

The health monitoring and automatic recovery system ensures:

✅ **Comprehensive health monitoring** across all system components
✅ **Automatic failure recovery** with multiple strategies
✅ **Intelligent alerting** with escalation paths
✅ **Performance optimization** through auto-tuning
✅ **Zero-downtime maintenance** windows
✅ **Incident response automation** with playbooks
✅ **Extensive testing** and validation coverage

This system provides **enterprise-grade reliability** and **self-healing capabilities** for production deployments.</content>
</xai:function_call">  
<xai:function_call name="write">
<parameter name="filePath">scripts/deploy-system.sh