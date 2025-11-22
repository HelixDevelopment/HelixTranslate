# Container Update and Restart Mechanisms

This document describes the mechanisms for updating and restarting Docker containers in the distributed Universal Ebook Translator system.

## Overview

The system provides comprehensive mechanisms for updating and restarting Docker containers on both host and worker machines, ensuring zero-downtime deployments and reliable service management.

## Update Mechanisms

### 1. Automated Deployment System Updates

The deployment CLI (`./build/deployment-cli`) supports update and restart operations:

```bash
# Update all services to latest images
./build/deployment-cli -action update

# Update specific service to new image
./build/deployment-cli -action update -service translator-main -image translator:latest

# Restart all services
./build/deployment-cli -action restart

# Restart specific service
./build/deployment-cli -action restart -service translator-worker-1
```

### 2. Container Update Script

The `scripts/update_containers.sh` script provides comprehensive container management:

```bash
# Update all services
./scripts/update_containers.sh update-all

# Update specific service to new image
./scripts/update_containers.sh -i translator:v2.1.0 update translator-main

# Restart all services
./scripts/update_containers.sh restart-all

# Restart specific service
./scripts/update_containers.sh restart translator-worker-1

# Update without waiting for health checks
./scripts/update_containers.sh --no-wait update-all

# Skip backup creation
./scripts/update_containers.sh --no-backup restart-all
```

### 3. Manual Docker Compose Operations

For direct control, you can use Docker Compose commands:

```bash
# Update all services
docker-compose pull
docker-compose up -d

# Update specific service
docker-compose pull translator-main
docker-compose up -d translator-main

# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart translator-worker-1
```

## Architecture Components

### Deployment Orchestrator Updates

The `DeploymentOrchestrator` provides high-level update operations:

- **UpdateService**: Updates a specific service to a new image version
- **UpdateAllServices**: Updates all deployed services to their latest images
- **RestartService**: Restarts a specific service
- **RestartAllServices**: Restarts all deployed services

### Docker Orchestrator Updates

The `DockerOrchestrator` handles Docker Compose-based updates:

- **UpdateService**: Updates a service to a new image and restarts it
- **UpdateAllServices**: Updates all services in the compose file
- **RestartService**: Restarts a specific service
- **RestartAllServices**: Restarts all services

### SSH Deployer Updates

The `SSHDeployer` manages remote container updates:

- **UpdateInstance**: Stops, removes, and recreates containers with new images
- **RestartInstance**: Restarts containers on remote hosts

## Update Process Flow

### Automated Update Process

1. **Backup Creation**: Current state is backed up (containers, configs, logs)
2. **Image Pull**: New images are pulled from registry
3. **Service Stop**: Services are gracefully stopped
4. **Container Removal**: Old containers are removed
5. **Service Start**: New containers are started with updated images
6. **Health Checks**: Services are verified to be healthy
7. **Cleanup**: Temporary files and old images are cleaned up

### Rollback Process

If an update fails, the system can rollback:

```bash
# Restore from backup
cp backups/backup_20241122_143000/docker-compose.yml .
cp backups/backup_20241122_143000/.env .

# Restart with previous configuration
docker-compose up -d
```

## Health Monitoring

### Health Check Integration

All update operations include health monitoring:

- **Pre-update**: Verify services are healthy before update
- **Post-update**: Wait for services to become healthy after update
- **Timeout handling**: Automatic rollback if health checks fail
- **Progress reporting**: Real-time status updates during updates

### Health Check Configuration

Health checks are configured in deployment plans:

```json
{
  "health_check": {
    "test": ["CMD", "curl", "-f", "https://localhost:8443/health"],
    "interval": "30s",
    "timeout": "10s",
    "retries": 3
  }
}
```

## Configuration Management

### Environment Variables

Update behavior can be controlled via environment variables:

```bash
# Docker Compose file
export COMPOSE_FILE="docker-compose.prod.yml"

# Backup directory
export BACKUP_DIR="./backups"

# Update timeout
export UPDATE_TIMEOUT="600s"

# Health check timeout
export HEALTH_TIMEOUT="300s"
```

### Configuration Files

Update settings can be specified in configuration files:

```yaml
# docker-compose.yml
services:
  translator-main:
    image: translator:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

## Monitoring and Logging

### Update Logging

All update operations are logged:

- **Update progress**: Step-by-step progress reporting
- **Error details**: Comprehensive error information
- **Timing information**: Duration of each update phase
- **Health status**: Before/after health check results

### Event Integration

Update events are published to the event bus:

```go
// Service updated event
event := events.Event{
    Type:      "service_updated",
    SessionID: "system",
    Message:   fmt.Sprintf("Service %s updated to %s", serviceName, newImage),
    Data: map[string]interface{}{
        "service": serviceName,
        "image":   newImage,
        "container_id": containerID,
    },
}
```

## Security Considerations

### Image Verification

- **Image signing**: Verify image signatures before deployment
- **Registry security**: Use secure private registries
- **Vulnerability scanning**: Scan images for security vulnerabilities

### Access Control

- **SSH key management**: Secure SSH key handling for remote updates
- **API authentication**: Secure API endpoints for update operations
- **Audit logging**: Comprehensive logging of all update operations

## Troubleshooting

### Common Issues

#### Update Fails Due to Port Conflicts

```bash
# Check port usage
netstat -tlnp | grep :8443

# Stop conflicting service
sudo systemctl stop apache2

# Retry update
./scripts/update_containers.sh update-all
```

#### Health Checks Fail After Update

```bash
# Check service logs
docker-compose logs translator-main

# Manual health check
curl -f https://localhost:8443/health

# Force restart
docker-compose restart translator-main
```

#### Image Pull Fails

```bash
# Check registry access
docker login registry.example.com

# Pull manually
docker pull translator:latest

# Retry update
./scripts/update_containers.sh update translator-main
```

### Recovery Procedures

#### Emergency Rollback

```bash
# Stop all services
docker-compose down

# Restore backup
cp backups/latest_backup/docker-compose.yml .
cp backups/latest_backup/.env .

# Start services
docker-compose up -d

# Verify health
docker-compose ps
```

#### Force Update

```bash
# Force pull images
docker-compose pull --no-cache

# Force recreate containers
docker-compose up -d --force-recreate

# Remove unused images
docker image prune -f
```

## Best Practices

### Update Strategy

1. **Test in staging**: Always test updates in a staging environment first
2. **Gradual rollout**: Update services incrementally, not all at once
3. **Monitor closely**: Watch logs and metrics during updates
4. **Have rollback plan**: Ensure quick rollback capability
5. **Automate where possible**: Use scripts and automation for consistency

### Maintenance Windows

- **Schedule updates**: Plan updates during low-traffic periods
- **Notify stakeholders**: Inform users of planned maintenance
- **Monitor post-update**: Watch for issues after updates
- **Document changes**: Keep records of all updates and changes

### Performance Optimization

- **Parallel updates**: Update multiple services simultaneously where safe
- **Image layer caching**: Leverage Docker layer caching for faster updates
- **Resource limits**: Set appropriate resource limits for containers
- **Network optimization**: Use local registries to reduce pull times

## Integration with CI/CD

### Automated Updates

The system can be integrated with CI/CD pipelines:

```yaml
# .github/workflows/update.yml
name: Update Containers
on:
  push:
    tags:
      - 'v*'

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Update containers
        run: ./scripts/update_containers.sh update-all
      - name: Health check
        run: ./scripts/health_check.sh
```

### Blue-Green Deployments

For zero-downtime updates:

```bash
# Create new compose file
cp docker-compose.yml docker-compose.green.yml

# Update image version
sed -i 's/translator:latest/translator:v2.1.0/' docker-compose.green.yml

# Start new services
docker-compose -f docker-compose.green.yml up -d

# Wait for health
./scripts/wait_healthy.sh -f docker-compose.green.yml

# Switch traffic (if using load balancer)
# ...

# Stop old services
docker-compose down

# Rename files
mv docker-compose.green.yml docker-compose.yml
```

This comprehensive update system ensures reliable, secure, and efficient container management across the distributed translation infrastructure.</content>
</xai:function_call">/dev/null