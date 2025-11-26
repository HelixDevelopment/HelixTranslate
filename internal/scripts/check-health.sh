#!/bin/bash
# check-health.sh - Comprehensive health checking

echo "=== Health Check Report ==="
echo "Timestamp: $(date)"
echo ""

all_healthy=true

# Check CLI binary
echo -n "CLI Binary: "
if [[ -f "./build/deployment-cli" ]]; then
    echo "✓ Exists"
else
    echo "✗ Missing"
    all_healthy=false
fi

# Check Docker
echo -n "Docker: "
if docker --version >/dev/null 2>&1; then
    echo "✓ Installed"
else
    echo "✗ Not installed"
    all_healthy=false
fi

# Check Docker Compose
echo -n "Docker Compose: "
if docker-compose --version >/dev/null 2>&1; then
    echo "✓ Installed"
else
    echo "✗ Not installed"
    all_healthy=false
fi

# Check configuration files
echo -n "Config files: "
config_ok=true
[[ -f "config.distributed.json" ]] || config_ok=false
[[ -f "deployment-plan.json" ]] || config_ok=false

if $config_ok; then
    echo "✓ Present"
else
    echo "✗ Missing"
    all_healthy=false
fi

# Check main service port
echo -n "Main service (8443): "
if timeout 5 bash -c "</dev/tcp/localhost/8443" >/dev/null 2>&1; then
    echo "✓ Accessible"
else
    echo "✗ Not accessible"
    all_healthy=false
fi

# Check API communication log
echo -n "API log: "
if [[ -f "workers_api_communication.log" ]]; then
    lines=$(wc -l < workers_api_communication.log)
    echo "✓ Exists ($lines entries)"
else
    echo "✗ Missing"
fi

# Check running containers
echo -n "Running containers: "
container_count=$(docker ps --filter "name=translator" --format "{{.Names}}" | wc -l)
if [[ $container_count -gt 0 ]]; then
    echo "✓ $container_count running"
else
    echo "✗ None running"
    all_healthy=false
fi

echo ""
if $all_healthy; then
    echo "✓ All health checks passed"
    exit 0
else
    echo "✗ Some health checks failed"
    exit 1
fi</content>
</xai:function_call">  
<xai:function_call name="write">
<parameter name="filePath">Makefile