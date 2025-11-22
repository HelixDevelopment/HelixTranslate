# Automated Distributed Deployment System

This document provides comprehensive guidance for the fully automated deployment and management of the distributed Universal Ebook Translator system.

## Overview

The automated deployment system enables:

- **Zero-touch deployment** of distributed instances across multiple hosts
- **Docker-based containerization** for consistent environments
- **Automatic port binding** to available ports
- **Network discovery and broadcasting** for service location
- **Comprehensive API communication logging** between nodes
- **Full autodiscovery mechanisms** for worker nodes
- **Enterprise-grade testing** and validation

## Architecture

### Core Components

#### 1. Deployment Orchestrator (`pkg/deployment/orchestrator.go`)
Central coordinator that manages the entire deployment lifecycle:

- **SSH-based deployment** to remote hosts
- **Docker container management** via docker-compose
- **Health monitoring** and automatic recovery
- **Network discovery integration**
- **API communication logging**

#### 2. SSH Deployer (`pkg/deployment/ssh_deployer.go`)
Handles secure deployment to remote machines:

- **Key-based authentication** for security
- **Docker image management** (pull, run, cleanup)
- **Container lifecycle management**
- **Health checking** of deployed instances

#### 3. Docker Orchestrator (`pkg/deployment/docker_orchestrator.go`)
Manages Docker-based deployments:

- **docker-compose file generation**
- **Multi-service orchestration**
- **Health monitoring** with automatic retries
- **Scaling operations**

#### 4. Network Discovery (`pkg/deployment/network_discovery.go`)
Provides service discovery and broadcasting:

- **UDP-based broadcasting** of service configurations
- **Automatic service discovery** across the network
- **TTL-based cleanup** of stale services
- **Real-time status updates**

#### 5. API Communication Logger (`pkg/deployment/api_logger.go`)
Logs all REST API communications:

- **Structured JSON logging** for analysis
- **Request/response tracking** with timing
- **Error logging** and statistics
- **Separate log file** as requested: `workers_api_communication.log`

## Quick Start

### Prerequisites

1. **SSH Access**: Key-based SSH access to all target hosts
2. **Docker**: Docker and docker-compose installed on all hosts
3. **Go**: Go 1.21+ for building the deployment tools
4. **TLS Certificates**: Generated certificates for HTTPS

### Basic Deployment

```bash
# 1. Build the deployment tool
make build-deployment

# 2. Generate a deployment plan
./build/deployment-cli -action generate-plan -config config.distributed.json

# 3. Deploy the system
./build/deployment-cli -action deploy -plan deployment-plan.json

# 4. Check status
./build/deployment-cli -action status
```

## Configuration

### Deployment Plan Format

```json
{
  "main": {
    "host": "main-server.example.com",
    "user": "translator",
    "ssh_key_path": "/path/to/ssh/key",
    "docker_image": "translator:latest",
    "container_name": "translator-main",
    "ports": [
      {
        "host_port": 8443,
        "container_port": 8443,
        "protocol": "tcp"
      }
    ],
    "environment": {
      "JWT_SECRET": "main-server-secret",
      "MAIN_INSTANCE": "true"
    },
    "volumes": [
      {
        "host_path": "./certs",
        "container_path": "/app/certs",
        "read_only": true
      }
    ],
    "networks": ["translator-network"],
    "restart_policy": "unless-stopped",
    "health_check": {
      "test": ["CMD", "curl", "-f", "https://localhost:8443/health"],
      "interval": "30s",
      "timeout": "10s",
      "retries": 3
    }
  },
  "workers": [
    {
      "host": "worker-1.example.com",
      "user": "translator",
      "ssh_key_path": "/path/to/ssh/key",
      "docker_image": "translator:latest",
      "container_name": "translator-worker-1",
      "ports": [
        {
          "host_port": 8444,
          "container_port": 8443,
          "protocol": "tcp"
        }
      ],
      "environment": {
        "JWT_SECRET": "worker-1-secret",
        "WORKER_INDEX": "1"
      }
    }
  ]
}
```

### Environment Variables

```bash
# SSH Configuration
export SSH_KEY_PATH="/path/to/private/key"
export SSH_USER="translator"

# Docker Configuration
export DOCKER_REGISTRY="your-registry.com"
export DOCKER_IMAGE_TAG="latest"

# Network Configuration
export DISCOVERY_PORT="9999"
export BROADCAST_ADDRESS="255.255.255.255"

# Logging
export API_LOG_FILE="workers_api_communication.log"
export LOG_LEVEL="info"
```

## Deployment Process

### Phase 1: Preparation

1. **Validate deployment plan**
2. **Check SSH connectivity** to all hosts
3. **Verify Docker availability** on target hosts
4. **Generate docker-compose files**

### Phase 2: Main Instance Deployment

1. **Find available port** (starting from 8443)
2. **Deploy main container** via SSH
3. **Wait for health checks** to pass
4. **Initialize network discovery**

### Phase 3: Worker Instance Deployment

1. **Deploy each worker** in parallel
2. **Automatic port assignment** (8444, 8445, etc.)
3. **Configure worker-specific environment**
4. **Register with main instance**

### Phase 4: System Validation

1. **Health checks** on all instances
2. **Network connectivity** verification
3. **API endpoint testing**
4. **Load balancing validation**

## Network Discovery

### Broadcasting Mechanism

Each deployed instance broadcasts its configuration every 30 seconds:

```json
{
  "service_id": "translator-main",
  "type": "coordinator",
  "host": "192.168.1.100",
  "port": 8443,
  "protocol": "https",
  "capabilities": {
    "container_id": "abc123...",
    "status": "healthy",
    "version": "2.0"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Discovery Process

1. **UDP Listening**: All instances listen on port 9999
2. **Broadcast Reception**: Process incoming service announcements
3. **TTL Management**: Clean up stale services after 90 seconds
4. **Capability Caching**: Store service capabilities for load balancing

### Service Types

- **coordinator**: Main translation server
- **worker**: Worker translation instances
- **unknown**: Unidentified services

## API Communication Logging

### Log Format

All API communications are logged in Retrofit-style format to `workers_api_communication.log` for impeccable readability and easy investigation:

```
# Outgoing requests (send)
2025/11/22 17:24:58.564 --> POST https://192.168.1.101:8444/api/v1/translate (1024-byte body)

# Incoming responses (received)
2025/11/22 17:24:58.714 <-- 200 OK https://192.168.1.101:8444/api/v1/translate (150ms, 2048-byte body)

# Error responses
2025/11/22 17:24:58.714 <-- HTTP FAILED: connection timeout
```

**Format Legend:**
- `--> ` = Outgoing request (send direction)
- `<-- ` = Incoming response (received direction)
- Timestamp format: `YYYY/MM/DD HH:MM:SS.mmm`
- Duration: `150ms` or `2.5s`
- Body size: `(1024-byte body)` for requests, `(2048-byte body)` for responses

### Log Analysis

```bash
# View recent communications
tail -f workers_api_communication.log

# Count requests by HTTP method
grep " --> " workers_api_communication.log | awk '{print $4}' | sort | uniq -c

# Count responses by status code
grep " <-- " workers_api_communication.log | awk '{print $4}' | sort | uniq -c

# Find slow responses (>1 second)
grep " <-- " workers_api_communication.log | grep -E "\([0-9]+\.[0-9]+s" | grep -v "(0\."

# Find failed requests
grep "HTTP FAILED" workers_api_communication.log

# Calculate average response time (responses only)
grep " <-- " workers_api_communication.log | grep -o -E "\([0-9]+ms|\([0-9]+\.[0-9]+s" | sed 's/[ms()]//g' | awk '
{
  if ($0 ~ /\./) {
    # Convert seconds to milliseconds
    printf "%.0f\n", $0 * 1000
  } else {
    print $0
  }
}
' | awk '{sum+=$1; count++} END {print "Average response time: " sum/count "ms"}'

# Show request/response pairs (last 10)
tail -20 workers_api_communication.log | grep -E " --> | <-- " | tail -20
```

## Port Management

### Automatic Port Binding

The system automatically finds available ports:

1. **Start from preferred port** (8443 for main, 8444+ for workers)
2. **Check port availability** using TCP connection attempts
3. **Increment and retry** up to 100 ports
4. **Update configurations** with assigned ports

### Port Conflict Resolution

```go
func (do *DeploymentOrchestrator) findAvailablePort(host string, preferredPort int) (int, error) {
    for port := preferredPort; port < preferredPort+100; port++ {
        if do.isPortAvailable(host, port) {
            return port, nil
        }
    }
    return 0, fmt.Errorf("no available ports found")
}
```

## Docker Integration

### Generated docker-compose.yml

```yaml
version: '3.8'
services:
  translator-main:
    image: translator:latest
    container_name: translator-main
    ports:
      - "8443:8443"
    environment:
      - JWT_SECRET=main-server-secret
    volumes:
      - ./certs:/app/certs:ro
    networks:
      - translator-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: translator
      POSTGRES_PASSWORD: secure_password
      POSTGRES_DB: translator
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - translator-network
    ports:
      - "5432:5432"

networks:
  translator-network:
    driver: bridge

volumes:
  postgres-data:
```

### Container Management

- **Automatic cleanup** of failed deployments
- **Health-based restarts** using Docker health checks
- **Resource limits** and monitoring
- **Log aggregation** and rotation

## Security Considerations

### SSH Security

- **Key-based authentication** only (no passwords)
- **Restricted SSH user** permissions
- **Regular key rotation** recommended
- **SSH bastion hosts** for complex networks

### Network Security

- **HTTPS-only communication** with TLS 1.3
- **Firewall configuration** limiting access
- **VPN usage** for untrusted networks
- **Service isolation** using Docker networks

### API Security

- **JWT authentication** between instances
- **API key validation** for external access
- **Rate limiting** and request validation
- **Audit logging** of all operations

## Monitoring and Troubleshooting

### Health Checks

```bash
# Check all container health
docker-compose ps

# View container logs
docker-compose logs translator-main

# Check API health
curl -k https://localhost:8443/health

# Monitor API communications
tail -f workers_api_communication.log
```

### Common Issues

#### SSH Connection Failures
```bash
# Test SSH connection
ssh -i /path/to/key translator@host.example.com

# Check SSH key permissions
ls -la /path/to/ssh/key  # Should be 600

# Verify SSH service
ssh translator@host.example.com systemctl status sshd
```

#### Docker Issues
```bash
# Check Docker service
sudo systemctl status docker

# Verify Docker images
docker images | grep translator

# Check container logs
docker logs translator-main
```

#### Network Discovery Issues
```bash
# Check UDP port availability
netstat -uln | grep 9999

# Test broadcasting
tcpdump -i any udp port 9999

# Verify firewall rules
sudo ufw status
```

#### Port Conflicts
```bash
# Find process using port
lsof -i :8443

# Check port availability
nc -z localhost 8443 || echo "Port available"
```

## Scaling Operations

### Adding Workers

```bash
# Add worker via API
curl -X POST https://main-server:8443/api/v1/deployment/workers \
  -H "Content-Type: application/json" \
  -d '{
    "host": "new-worker.example.com",
    "user": "translator",
    "ssh_key_path": "/path/to/key"
  }'
```

### Scaling Existing Services

```bash
# Scale worker instances
docker-compose up -d --scale translator-worker-1=3

# Update deployment plan
./deployment-cli -action generate-plan
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup deployment configuration
cp deployment-plan.json deployment-plan.json.backup

# Backup SSL certificates
tar -czf certs-backup.tar.gz certs/

# Backup API logs
cp workers_api_communication.log workers_api_communication.log.$(date +%Y%m%d)
```

### Recovery Procedures

1. **Stop all instances**: `docker-compose down`
2. **Restore configurations** from backup
3. **Redeploy system**: `./deployment-cli -action deploy`
4. **Verify health checks** pass
5. **Restore data** if needed

## Performance Optimization

### Connection Pooling

- **SSH connection reuse** across operations
- **HTTP client pooling** for API calls
- **Database connection pooling** in containers

### Caching Strategies

- **DNS resolution caching**
- **Container image layer caching**
- **API response caching**

### Resource Management

- **CPU and memory limits** per container
- **Automatic scaling** based on load
- **Resource monitoring** and alerting

## Testing

### Unit Tests

```bash
# Run deployment tests
go test ./pkg/deployment/... -v

# Run with coverage
go test ./pkg/deployment/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Test full deployment cycle
go test ./test/deployment/integration_test.go -v

# Test network discovery
go test ./test/deployment/network_test.go -v
```

### Performance Tests

```bash
# Load testing
go test ./test/deployment/performance_test.go -v -bench=.

# Stress testing
go test ./test/deployment/stress_test.go -v
```

## API Reference

### Deployment Status

```http
GET /api/v1/deployment/status
```

Returns comprehensive deployment status:

```json
{
  "instances": {
    "translator-main": {
      "host": "192.168.1.100",
      "port": 8443,
      "status": "healthy",
      "container_id": "abc123...",
      "last_seen": "2024-01-15T10:30:00Z"
    }
  },
  "network_services": 3,
  "api_requests_total": 1250,
  "health_score": 98.5
}
```

### Add Worker

```http
POST /api/v1/deployment/workers
Content-Type: application/json

{
  "host": "worker.example.com",
  "user": "translator",
  "ssh_key_path": "/path/to/key"
}
```

### API Communication Stats

```http
GET /api/v1/deployment/api-stats
```

Returns API communication statistics:

```json
{
  "total_requests": 1250,
  "total_responses": 1248,
  "error_count": 2,
  "avg_duration": "45ms",
  "requests_by_endpoint": {
    "/api/v1/translate": 1200,
    "/api/v1/health": 50
  }
}
```

## Future Enhancements

- **Kubernetes integration** for cloud deployments
- **Terraform integration** for infrastructure as code
- **Rolling updates** with zero downtime
- **Multi-region support** with geo-distribution
- **Advanced monitoring** with Prometheus/Grafana
- **Automated scaling** based on AI/ML predictions

---

## Summary

The automated deployment system provides enterprise-grade capabilities for deploying and managing distributed Universal Ebook Translator instances:

✅ **Fully automated SSH deployment** to remote hosts
✅ **Docker-based containerization** with docker-compose
✅ **Automatic port binding** to available ports
✅ **Network discovery and broadcasting** for service location
✅ **Comprehensive API communication logging** to separate file
✅ **Full autodiscovery mechanisms** for worker nodes
✅ **Enterprise-grade testing** and validation
✅ **Detailed documentation** and troubleshooting guides

The system ensures **zero-touch deployment**, **high availability**, and **comprehensive monitoring** for production environments.</content>
</xai:function_call"></xai:function_call"  
<xai:function_call name="write">
<parameter name="filePath">Documentation/DEPLOYMENT_SCRIPTS.md