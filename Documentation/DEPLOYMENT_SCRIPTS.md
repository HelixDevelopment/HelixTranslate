# Deployment Scripts and Automation

This document provides detailed information about the deployment scripts and automation tools for the distributed Universal Ebook Translator system.

## Overview

The deployment system includes several automation scripts and tools:

- **Deployment CLI** (`cmd/deployment/main.go`) - Main deployment interface
- **Automated SSH deployment** - Zero-touch remote host setup
- **Docker orchestration** - Container management and scaling
- **Network discovery** - Automatic service location and broadcasting
- **API logging** - Comprehensive communication tracking

## Deployment CLI

### Building the CLI

```bash
# Build the deployment CLI
make build-deployment

# Or build manually
cd cmd/deployment
go build -o ../../build/deployment-cli main.go
```

### CLI Usage

```bash
# Show help
./build/deployment-cli -h

# Generate deployment plan from configuration
./build/deployment-cli -action generate-plan -config config.distributed.json

# Deploy using plan
./build/deployment-cli -action deploy -plan deployment-plan.json

# Check deployment status
./build/deployment-cli -action status

# Stop deployment
./build/deployment-cli -action stop

# Cleanup deployment
./build/deployment-cli -action cleanup
```

### CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Configuration file path | `config.distributed.json` |
| `-action` | Action to perform | `deploy` |
| `-plan` | Deployment plan JSON file | - |
| `-verbose` | Enable verbose logging | `false` |

## Automation Scripts

### One-Click Deployment Script

```bash
#!/bin/bash
# deploy-system.sh - Complete automated deployment

set -e

echo "🚀 Starting Universal Ebook Translator Deployment"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONFIG_FILE="${CONFIG_FILE:-config.distributed.json}"
PLAN_FILE="${PLAN_FILE:-deployment-plan.json}"
CLI_BINARY="${CLI_BINARY:-./build/deployment-cli}"

# Validate prerequisites
validate_prerequisites() {
    echo -e "${YELLOW}Validating prerequisites...${NC}"

    # Check if CLI binary exists
    if [[ ! -f "$CLI_BINARY" ]]; then
        echo -e "${RED}Error: CLI binary not found at $CLI_BINARY${NC}"
        echo "Run 'make build-deployment' first"
        exit 1
    fi

    # Check if config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo -e "${RED}Error: Configuration file not found at $CONFIG_FILE${NC}"
        exit 1
    fi

    # Check SSH keys
    if [[ -z "$SSH_KEY_PATH" ]]; then
        echo -e "${YELLOW}Warning: SSH_KEY_PATH not set, using default key locations${NC}"
    fi

    echo -e "${GREEN}Prerequisites validated${NC}"
}

# Generate deployment plan
generate_plan() {
    echo -e "${YELLOW}Generating deployment plan...${NC}"

    if [[ -f "$PLAN_FILE" ]]; then
        echo -e "${YELLOW}Plan file already exists, backing up...${NC}"
        mv "$PLAN_FILE" "${PLAN_FILE}.backup.$(date +%s)"
    fi

    "$CLI_BINARY" -action generate-plan -config "$CONFIG_FILE" -verbose

    if [[ ! -f "$PLAN_FILE" ]]; then
        echo -e "${RED}Error: Failed to generate deployment plan${NC}"
        exit 1
    fi

    echo -e "${GREEN}Deployment plan generated: $PLAN_FILE${NC}"
}

# Deploy the system
deploy_system() {
    echo -e "${YELLOW}Deploying distributed system...${NC}"

    "$CLI_BINARY" -action deploy -plan "$PLAN_FILE" -verbose

    echo -e "${GREEN}System deployment initiated${NC}"
}

# Wait for deployment completion
wait_for_deployment() {
    echo -e "${YELLOW}Waiting for deployment to complete...${NC}"

    local max_attempts=60  # 10 minutes with 10s intervals
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        echo -e "${YELLOW}Checking deployment status (attempt $attempt/$max_attempts)...${NC}"

        # Check if deployment is complete
        if "$CLI_BINARY" -action status >/dev/null 2>&1; then
            local status_output
            status_output=$("$CLI_BINARY" -action status 2>/dev/null)

            if echo "$status_output" | grep -q "healthy"; then
                echo -e "${GREEN}Deployment completed successfully!${NC}"
                return 0
            fi
        fi

        sleep 10
        ((attempt++))
    done

    echo -e "${RED}Deployment did not complete within timeout${NC}"
    return 1
}

# Validate deployment
validate_deployment() {
    echo -e "${YELLOW}Validating deployment...${NC}"

    local status_output
    status_output=$("$CLI_BINARY" -action status)

    echo "$status_output"

    # Check for healthy instances
    local healthy_count
    healthy_count=$(echo "$status_output" | grep -c "healthy" || true)

    if [[ $healthy_count -eq 0 ]]; then
        echo -e "${RED}Error: No healthy instances found${NC}"
        return 1
    fi

    echo -e "${GREEN}Validation successful: $healthy_count healthy instances${NC}"
}

# Main deployment flow
main() {
    echo -e "${GREEN}=== Universal Ebook Translator Deployment ===${NC}"

    validate_prerequisites
    generate_plan
    deploy_system

    if wait_for_deployment; then
        validate_deployment
        echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
        echo ""
        echo "Next steps:"
        echo "1. Check API logs: tail -f workers_api_communication.log"
        echo "2. Monitor health: $CLI_BINARY -action status"
        echo "3. Scale workers: docker-compose up -d --scale translator-worker-1=3"
    else
        echo -e "${RED}❌ Deployment failed${NC}"
        echo ""
        echo "Troubleshooting:"
        echo "1. Check logs: $CLI_BINARY -action status"
        echo "2. View container logs: docker-compose logs"
        echo "3. Cleanup and retry: $CLI_BINARY -action cleanup"
        exit 1
    fi
}

# Run main function
main "$@"
```

### Health Monitoring Script

```bash
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
done
```

### Auto-Scaling Script

```bash
#!/bin/bash
# autoscale-workers.sh - Automatic worker scaling based on load

set -e

CLI_BINARY="${CLI_BINARY:-./build/deployment-cli}"
MAIN_HOST="${MAIN_HOST:-localhost}"
MAIN_PORT="${MAIN_PORT:-8443}"
CHECK_INTERVAL="${CHECK_INTERVAL:-60}"
CPU_THRESHOLD="${CPU_THRESHOLD:-70}"
MEMORY_THRESHOLD="${MEMORY_THRESHOLD:-80}"

echo "Starting auto-scaling monitor..."
echo "Main host: $MAIN_HOST:$MAIN_PORT"
echo "Check interval: ${CHECK_INTERVAL}s"
echo "CPU threshold: ${CPU_THRESHOLD}%"
echo "Memory threshold: ${MEMORY_THRESHOLD}%"
echo ""

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Get system metrics from main instance
    metrics=$(curl -s -k "https://$MAIN_HOST:$MAIN_PORT/api/v1/system/metrics" 2>/dev/null || echo "{}")

    # Extract CPU and memory usage (simplified example)
    cpu_usage=$(echo "$metrics" | jq -r '.cpu_usage // 0')
    memory_usage=$(echo "$metrics" | jq -r '.memory_usage // 0')

    echo "[$timestamp] CPU: ${cpu_usage}%, Memory: ${memory_usage}%"

    # Check if scaling is needed
    if (( $(echo "$cpu_usage > $CPU_THRESHOLD" | bc -l) )) || \
       (( $(echo "$memory_usage > $MEMORY_THRESHOLD" | bc -l) )); then

        echo "[$timestamp] Load threshold exceeded, scaling up workers..."

        # Scale up workers (example: increase worker-1 replicas)
        docker-compose up -d --scale translator-worker-1=3

        echo "[$timestamp] Scaled up to 3 worker instances"

    elif (( $(echo "$cpu_usage < 30" | bc -l) )) && \
         (( $(echo "$memory_usage < 40" | bc -l) )); then

        echo "[$timestamp] Load is low, scaling down workers..."

        # Scale down workers
        docker-compose up -d --scale translator-worker-1=1

        echo "[$timestamp] Scaled down to 1 worker instance"
    fi

    sleep "$CHECK_INTERVAL"
done
```

### Backup and Recovery Script

```bash
#!/bin/bash
# backup-deployment.sh - Backup deployment configuration and data

set -e

BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="deployment_backup_$TIMESTAMP"

echo "Creating deployment backup: $BACKUP_NAME"

# Create backup directory
mkdir -p "$BACKUP_DIR/$BACKUP_NAME"

# Backup configuration files
echo "Backing up configuration files..."
cp -r config.* "$BACKUP_DIR/$BACKUP_NAME/" 2>/dev/null || true
cp -r certs/ "$BACKUP_DIR/$BACKUP_NAME/" 2>/dev/null || true
cp deployment-plan.json "$BACKUP_DIR/$BACKUP_NAME/" 2>/dev/null || true

# Backup API communication logs
echo "Backing up API logs..."
cp workers_api_communication.log "$BACKUP_DIR/$BACKUP_NAME/" 2>/dev/null || true

# Backup Docker volumes (if using named volumes)
echo "Backing up Docker volumes..."
docker run --rm -v translator_postgres-data:/data -v "$PWD/$BACKUP_DIR/$BACKUP_NAME:/backup" alpine tar czf /backup/postgres-data.tar.gz -C / data 2>/dev/null || true

# Create backup manifest
cat > "$BACKUP_DIR/$BACKUP_NAME/manifest.txt" << EOF
Backup created: $TIMESTAMP
Included:
- Configuration files (config.*)
- SSL certificates (certs/)
- Deployment plan (deployment-plan.json)
- API communication logs (workers_api_communication.log)
- PostgreSQL data (postgres-data.tar.gz)

To restore:
1. Stop the deployment: ./build/deployment-cli -action stop
2. Restore files: cp -r $BACKUP_NAME/* ./
3. Restore volumes: docker run --rm -v translator_postgres-data:/data -v \$PWD:/backup alpine sh -c "cd /data && tar xzf /backup/$BACKUP_NAME/postgres-data.tar.gz"
4. Restart deployment: ./build/deployment-cli -action deploy
EOF

# Compress backup
echo "Compressing backup..."
cd "$BACKUP_DIR"
tar czf "${BACKUP_NAME}.tar.gz" "$BACKUP_NAME"
rm -rf "$BACKUP_NAME"

echo "Backup completed: $BACKUP_DIR/${BACKUP_NAME}.tar.gz"
echo "Size: $(du -h "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" | cut -f1)"
```

### Recovery Script

```bash
#!/bin/bash
# restore-deployment.sh - Restore deployment from backup

set -e

BACKUP_FILE="$1"

if [[ -z "$BACKUP_FILE" ]]; then
    echo "Usage: $0 <backup-file.tar.gz>"
    echo "Available backups:"
    ls -la ./backups/*.tar.gz 2>/dev/null || echo "No backups found"
    exit 1
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "Restoring deployment from: $BACKUP_FILE"

# Stop current deployment
echo "Stopping current deployment..."
./build/deployment-cli -action stop 2>/dev/null || true

# Extract backup
echo "Extracting backup..."
BACKUP_NAME=$(basename "$BACKUP_FILE" .tar.gz)
mkdir -p "/tmp/$BACKUP_NAME"
tar xzf "$BACKUP_FILE" -C "/tmp/$BACKUP_NAME"

# Restore files
echo "Restoring configuration files..."
cp -r "/tmp/$BACKUP_NAME"/* ./ 2>/dev/null || true

# Restore Docker volumes
if [[ -f "/tmp/$BACKUP_NAME/postgres-data.tar.gz" ]]; then
    echo "Restoring PostgreSQL data..."
    docker run --rm -v translator_postgres-data:/data -v "$PWD:/backup" alpine sh -c "cd /data && tar xzf /backup/tmp/$BACKUP_NAME/postgres-data.tar.gz" 2>/dev/null || true
fi

# Cleanup
rm -rf "/tmp/$BACKUP_NAME"

# Restart deployment
echo "Restarting deployment..."
./build/deployment-cli -action deploy

echo "Restore completed successfully!"
```

## Makefile Integration

Add these targets to your `Makefile`:

```makefile
.PHONY: build-deployment deploy stop-deployment cleanup-deployment

# Build deployment CLI
build-deployment:
	cd cmd/deployment && go build -o ../../build/deployment-cli main.go

# Deploy system
deploy: build-deployment
	./build/deployment-cli -action deploy -plan deployment-plan.json

# Stop deployment
stop-deployment:
	./build/deployment-cli -action stop

# Cleanup deployment
cleanup-deployment:
	./build/deployment-cli -action cleanup

# Generate deployment plan
plan: build-deployment
	./build/deployment-cli -action generate-plan

# Check deployment status
status: build-deployment
	./build/deployment-cli -action status

# Full deployment with monitoring
deploy-full: deploy
	@echo "Starting health monitoring..."
	./scripts/monitor-deployment.sh &
	@echo "Deployment completed with monitoring"
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy Distributed System

on:
  push:
    branches: [ main ]
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Build deployment CLI
      run: make build-deployment

    - name: Generate deployment plan
      run: make plan

    - name: Deploy to staging
      if: github.ref == 'refs/heads/develop'
      run: |
        export SSH_KEY_PATH=${{ secrets.STAGING_SSH_KEY }}
        make deploy

    - name: Deploy to production
      if: github.ref == 'refs/heads/main'
      run: |
        export SSH_KEY_PATH=${{ secrets.PROD_SSH_KEY }}
        make deploy

    - name: Health check
      run: |
        sleep 30
        make status
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any

    environment {
        SSH_KEY_PATH = credentials('translator-ssh-key')
        CONFIG_FILE = 'config.distributed.json'
    }

    stages {
        stage('Build') {
            steps {
                sh 'make build-deployment'
            }
        }

        stage('Plan') {
            steps {
                sh 'make plan'
            }
        }

        stage('Deploy') {
            steps {
                sh 'make deploy'
            }
        }

        stage('Health Check') {
            steps {
                sh 'sleep 30 && make status'
            }
        }

        stage('Monitor') {
            steps {
                sh './scripts/monitor-deployment.sh &'
            }
        }
    }

    post {
        failure {
            sh 'make stop-deployment || true'
        }
    }
}
```

## Troubleshooting Scripts

### Quick Diagnostic Script

```bash
#!/bin/bash
# diagnose-deployment.sh - Quick deployment diagnostics

echo "=== Deployment Diagnostics ==="
echo ""

echo "1. CLI Binary Status:"
if [[ -f "./build/deployment-cli" ]]; then
    echo "✓ CLI binary exists"
    ./build/deployment-cli -action status >/dev/null 2>&1 && echo "✓ CLI is functional" || echo "✗ CLI is not responding"
else
    echo "✗ CLI binary not found"
fi
echo ""

echo "2. Docker Status:"
docker --version >/dev/null 2>&1 && echo "✓ Docker is installed" || echo "✗ Docker not found"
docker-compose --version >/dev/null 2>&1 && echo "✓ Docker Compose is installed" || echo "✗ Docker Compose not found"
echo ""

echo "3. Configuration Files:"
[[ -f "config.distributed.json" ]] && echo "✓ Distributed config exists" || echo "✗ Distributed config missing"
[[ -f "deployment-plan.json" ]] && echo "✓ Deployment plan exists" || echo "✗ Deployment plan missing"
[[ -f "workers_api_communication.log" ]] && echo "✓ API log exists" || echo "✗ API log missing"
echo ""

echo "4. Network Status:"
timeout 5 bash -c "</dev/tcp/localhost/8443" >/dev/null 2>&1 && echo "✓ Main port (8443) is accessible" || echo "✗ Main port (8443) not accessible"
echo ""

echo "5. Recent API Activity:"
if [[ -f "workers_api_communication.log" ]]; then
    lines=$(wc -l < workers_api_communication.log)
    echo "✓ API log has $lines entries"
    echo "Recent activity:"
    tail -5 workers_api_communication.log | sed 's/^/  /'
else
    echo "✗ No API activity log"
fi
echo ""

echo "6. Container Status:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep translator || echo "No translator containers running"
echo ""

echo "=== End Diagnostics ==="
```

### Log Analysis Script

```bash
#!/bin/bash
# analyze-api-logs.sh - Analyze API communication logs

LOG_FILE="${1:-workers_api_communication.log}"

if [[ ! -f "$LOG_FILE" ]]; then
    echo "Error: Log file not found: $LOG_FILE"
    exit 1
fi

echo "=== API Communication Analysis ==="
echo "Log file: $LOG_FILE"
echo "Total entries: $(wc -l < "$LOG_FILE")"
echo ""

echo "1. HTTP Status Code Distribution:"
jq -r '.status_code // "unknown"' "$LOG_FILE" 2>/dev/null | sort | uniq -c | sort -nr || echo "Unable to parse JSON logs"
echo ""

echo "2. Most Active Endpoints:"
jq -r '.url // "unknown"' "$LOG_FILE" 2>/dev/null | sort | uniq -c | sort -nr | head -10 || echo "Unable to parse JSON logs"
echo ""

echo "3. Response Time Statistics:"
durations=$(jq -r '.duration // "0ms"' "$LOG_FILE" 2>/dev/null | sed 's/ms//' | sed 's/s/*1000/' | bc 2>/dev/null || echo "0")
if [[ -n "$durations" ]]; then
    echo "$durations" | awk '
        BEGIN { count=0; sum=0; min=999999; max=0 }
        { count++; sum+=$1; if($1<min) min=$1; if($1>max) max=$1 }
        END {
            if(count>0) {
                avg=sum/count
                print "Count: " count
                print "Average: " avg "ms"
                print "Min: " min "ms"
                print "Max: " max "ms"
            } else {
                print "No valid duration data found"
            }
        }'
else
    echo "Unable to parse duration data"
fi
echo ""

echo "4. Error Analysis:"
error_count=$(jq 'select(.status_code >= 400)' "$LOG_FILE" 2>/dev/null | wc -l)
echo "Total errors: $error_count"

if [[ $error_count -gt 0 ]]; then
    echo "Top error codes:"
    jq -r 'select(.status_code >= 400) | .status_code' "$LOG_FILE" 2>/dev/null | sort | uniq -c | sort -nr || echo "Unable to parse error data"
fi
echo ""

echo "5. Communication Partners:"
jq -r '"\(.source_host):\(.source_port) -> \(.target_host):\(.target_port)"' "$LOG_FILE" 2>/dev/null | sort | uniq -c | sort -nr | head -10 || echo "Unable to parse communication data"
echo ""

echo "=== End Analysis ==="
```

## Summary

These scripts provide comprehensive automation for:

✅ **One-click deployment** with validation and monitoring
✅ **Health monitoring** with alerting capabilities
✅ **Auto-scaling** based on system load
✅ **Backup and recovery** with data preservation
✅ **CI/CD integration** for automated deployments
✅ **Troubleshooting tools** for quick diagnostics
✅ **Log analysis** for performance insights

The scripts ensure **zero-touch operations**, **high availability**, and **comprehensive monitoring** for production deployments.</content>
</xai:function_call">  
<xai:function_call name="write">
<parameter name="filePath">Documentation/HEALTH_MONITORING.md