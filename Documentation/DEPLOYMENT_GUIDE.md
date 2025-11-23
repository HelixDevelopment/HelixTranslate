# Deployment Guide - Universal Multi-Format Multi-Language Ebook Translation System

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Deployment Options](#deployment-options)
4. [Single-Server Deployment](#single-server-deployment)
5. [Distributed Deployment](#distributed-deployment)
6. [Docker Deployment](#docker-deployment)
7. [Kubernetes Deployment](#kubernetes-deployment)
8. [Cloud Platform Deployment](#cloud-platform-deployment)
9. [Security Configuration](#security-configuration)
10. [Monitoring and Logging](#monitoring-and-logging)
11. [Backup and Recovery](#backup-and-recovery)
12. [Performance Optimization](#performance-optimization)
13. [Troubleshooting](#troubleshooting)

## Overview

This guide provides comprehensive instructions for deploying the Universal Multi-Format Multi-Language Ebook Translation System in production environments. The system supports multiple deployment architectures from single-server setups to large-scale distributed deployments.

### Supported Deployment Architectures

- **Single-Server**: All components on one machine
- **Distributed**: Multiple workers with coordination server
- **Containerized**: Docker or Kubernetes deployments
- **Cloud**: AWS, GCP, Azure deployments

## Prerequisites

### System Requirements

#### Minimum Requirements
- **CPU**: 4 cores
- **Memory**: 8GB RAM
- **Storage**: 50GB SSD
- **Network**: 100 Mbps
- **OS**: Linux (Ubuntu 20.04+, CentOS 8+), macOS 10.15+, Windows 10+

#### Recommended Requirements
- **CPU**: 8+ cores
- **Memory**: 16GB+ RAM
- **Storage**: 100GB+ SSD
- **Network**: 1 Gbps
- **OS**: Linux (Ubuntu 22.04 LTS)

### Software Dependencies

```bash
# Go 1.25.2+
go version

# Docker 20.10+
docker --version

# Docker Compose 2.0+
docker-compose --version

# Git 2.30+
git --version
```

### Network Requirements

- **Port 8080**: API server (HTTP)
- **Port 8081**: API server (HTTPS)
- **Port 8082**: WebSocket server
- **Port 9090**: Metrics endpoint
- **Port 6379**: Redis (if using distributed mode)
- **Port 5432**: PostgreSQL (if using external database)

## Deployment Options

### Option 1: Binary Deployment

**Pros**: Full control, minimal dependencies
**Cons**: Manual setup and maintenance

```bash
# Download binaries
wget https://releases.example.com/translator/v1.0.0/translator-linux-amd64
wget https://releases.example.com/translator/v1.0.0/translator-server-linux-amd64

# Make executable
chmod +x translator-linux-amd64 translator-server-linux-amd64

# Move to system path
sudo mv translator-linux-amd64 /usr/local/bin/translator
sudo mv translator-server-linux-amd64 /usr/local/bin/translator-server
```

### Option 2: Docker Deployment

**Pros**: Isolated environment, easy scaling
**Cons**: Docker overhead

```bash
# Pull images
docker pull translator/cli:latest
docker pull translator/server:latest

# Or build from source
docker build -t translator/cli .
docker build -t translator/server -f Dockerfile.server .
```

### Option 3: Kubernetes Deployment

**Pros**: Auto-scaling, high availability
**Cons**: Complex setup

```bash
# Apply manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## Single-Server Deployment

### Step 1: Configuration

Create configuration file:

```bash
# Create config directory
sudo mkdir -p /etc/translator
sudo mkdir -p /var/log/translator
sudo mkdir -p /var/lib/translator

# Create production config
sudo tee /etc/translator/config.json > /dev/null <<EOF
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "tls_port": 8081,
    "enable_tls": true,
    "cert_file": "/etc/translator/certs/server.crt",
    "key_file": "/etc/translator/certs/server.key"
  },
  "database": {
    "type": "sqlite",
    "connection_string": "/var/lib/translator/translator.db"
  },
  "cache": {
    "type": "memory",
    "ttl": "1h"
  },
  "llm": {
    "provider": "anthropic",
    "api_key": "\${ANTHROPIC_API_KEY}",
    "model": "claude-3-sonnet-20240229",
    "max_tokens": 4096,
    "temperature": 0.3
  },
  "logging": {
    "level": "info",
    "file": "/var/log/translator/translator.log",
    "max_size": "100MB",
    "max_backups": 10
  },
  "security": {
    "api_key_required": true,
    "rate_limit": {
      "requests_per_minute": 60,
      "burst": 10
    }
  }
}
EOF
```

### Step 2: SSL Certificates

Generate SSL certificates:

```bash
# Create certs directory
sudo mkdir -p /etc/translator/certs

# Generate self-signed certificate (for testing)
sudo openssl req -x509 -newkey rsa:4096 -keyout /etc/translator/certs/server.key \
  -out /etc/translator/certs/server.crt -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=translator.example.com"

# Or use Let's Encrypt (for production)
sudo apt install certbot
sudo certbot certonly --standalone -d translator.example.com
sudo cp /etc/letsencrypt/live/translator.example.com/fullchain.pem /etc/translator/certs/server.crt
sudo cp /etc/letsencrypt/live/translator.example.com/privkey.pem /etc/translator/certs/server.key
```

### Step 3: Systemd Service

Create systemd service file:

```bash
sudo tee /etc/systemd/system/translator-server.service > /dev/null <<EOF
[Unit]
Description=Translator Server
After=network.target

[Service]
Type=simple
User=translator
Group=translator
WorkingDirectory=/opt/translator
ExecStart=/usr/local/bin/translator-server --config /etc/translator/config.json
Restart=always
RestartSec=10
Environment=ANTHROPIC_API_KEY=\${ANTHROPIC_API_KEY}
EnvironmentFile=-/etc/translator/environment

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/translator /var/log/translator

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

# Create user
sudo useradd -r -s /bin/false translator
sudo chown -R translator:translator /etc/translator /var/lib/translator /var/log/translator

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable translator-server
sudo systemctl start translator-server
```

### Step 4: Nginx Reverse Proxy (Optional)

```bash
sudo apt install nginx

sudo tee /etc/nginx/sites-available/translator > /dev/null <<EOF
server {
    listen 80;
    server_name translator.example.com;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name translator.example.com;

    ssl_certificate /etc/translator/certs/server.crt;
    ssl_certificate_key /etc/translator/certs/server.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    client_max_body_size 100M;

    location / {
        proxy_pass https://localhost:8081;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /ws {
        proxy_pass http://localhost:8082;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/translator /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Distributed Deployment

### Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │  Coordination   │    │     Workers     │
│                 │    │     Server      │    │                 │
│  ┌─────────────┐│    │                 │    │  ┌─────────────┐│
│  │   Nginx     ││    │  ┌─────────────┐│    │  │  Worker 1   ││
│  │   HAProxy   ││────┤  │translator-  ││────┤  │  Worker 2   ││
│  │   AWS ALB   ││    │  │server       ││    │  │  Worker N   ││
│  └─────────────┘│    │  └─────────────┘│    │  └─────────────┘│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Step 1: Coordination Server Setup

```bash
# Create coordination server config
sudo tee /etc/translator/coordination-config.json > /dev/null <<EOF
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "coordination": {
      "enabled": true,
      "worker_timeout": "30s",
      "heartbeat_interval": "10s"
    }
  },
  "database": {
    "type": "postgresql",
    "connection_string": "postgres://translator:password@db.example.com:5432/translator"
  },
  "cache": {
    "type": "redis",
    "connection_string": "redis://redis.example.com:6379"
  },
  "queue": {
    "type": "redis",
    "connection_string": "redis://redis.example.com:6379"
  }
}
EOF
```

### Step 2: Worker Configuration

```bash
# Create worker config
sudo tee /etc/translator/worker-config.json > /dev/null <<EOF
{
  "worker": {
    "id": "worker-1",
    "coordination_server": "http://coordination.example.com:8080",
    "heartbeat_interval": "10s",
    "max_concurrent_jobs": 5
  },
  "llm": {
    "provider": "anthropic",
    "api_key": "\${ANTHROPIC_API_KEY}",
    "model": "claude-3-sonnet-20240229"
  },
  "storage": {
    "type": "s3",
    "bucket": "translator-files",
    "region": "us-west-2",
    "access_key": "\${AWS_ACCESS_KEY_ID}",
    "secret_key": "\${AWS_SECRET_ACCESS_KEY}"
  }
}
EOF
```

### Step 3: Worker Service

```bash
sudo tee /etc/systemd/system/translator-worker.service > /dev/null <<EOF
[Unit]
Description=Translator Worker
After=network.target

[Service]
Type=simple
User=translator
Group=translator
WorkingDirectory=/opt/translator
ExecStart=/usr/local/bin/translator --config /etc/translator/worker-config.json worker
Restart=always
RestartSec=10
Environment=ANTHROPIC_API_KEY=\${ANTHROPIC_API_KEY}
Environment=AWS_ACCESS_KEY_ID=\${AWS_ACCESS_KEY_ID}
Environment=AWS_SECRET_ACCESS_KEY=\${AWS_SECRET_ACCESS_KEY}

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable translator-worker
sudo systemctl start translator-worker
```

## Docker Deployment

### Docker Compose Configuration

```yaml
# docker-compose.production.yml
version: '3.8'

services:
  translator-server:
    image: translator/server:latest
    ports:
      - "8080:8080"
      - "8081:8081"
      - "8082:8082"
    volumes:
      - ./config:/etc/translator
      - ./certs:/etc/translator/certs
      - ./logs:/var/log/translator
      - ./data:/var/lib/translator
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
    command: redis-server --appendonly yes

  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=translator
      - POSTGRES_USER=translator
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    depends_on:
      - translator-server
    restart: unless-stopped

volumes:
  redis_data:
  postgres_data:
```

### Deployment Commands

```bash
# Create environment file
cat > .env <<EOF
ANTHROPIC_API_KEY=your_api_key_here
DATABASE_URL=postgres://translator:password@postgres:5432/translator
REDIS_URL=redis://redis:6379
POSTGRES_PASSWORD=secure_password_here
EOF

# Deploy
docker-compose -f docker-compose.production.yml up -d

# Scale workers
docker-compose -f docker-compose.production.yml up -d --scale translator-worker=3

# Check status
docker-compose -f docker-compose.production.yml ps
docker-compose -f docker-compose.production.yml logs -f translator-server
```

## Kubernetes Deployment

### Namespace and ConfigMap

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: translator

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: translator-config
  namespace: translator
data:
  config.json: |
    {
      "server": {
        "host": "0.0.0.0",
        "port": 8080
      },
      "database": {
        "type": "postgresql",
        "connection_string": "postgres://translator:password@postgres:5432/translator"
      },
      "cache": {
        "type": "redis",
        "connection_string": "redis://redis:6379"
      }
    }
```

### Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: translator-server
  namespace: translator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: translator-server
  template:
    metadata:
      labels:
        app: translator-server
    spec:
      containers:
      - name: translator-server
        image: translator/server:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /etc/translator
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: translator-secrets
              key: anthropic-api-key
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: translator-config
```

### Service and Ingress

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: translator-service
  namespace: translator
spec:
  selector:
    app: translator-server
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8081
  type: ClusterIP

---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: translator-ingress
  namespace: translator
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: 100m
spec:
  tls:
  - hosts:
    - translator.example.com
    secretName: translator-tls
  rules:
  - host: translator.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: translator-service
            port:
              number: 80
```

## Cloud Platform Deployment

### AWS Deployment

#### EC2 Single Instance

```bash
# Create EC2 instance
aws ec2 run-instances \
  --image-id ami-0c02fb55956c7d316 \
  --instance-type t3.medium \
  --key-name my-key-pair \
  --security-group-ids sg-903004f8 \
  --subnet-id subnet-6e7f829e \
  --user-data file://user-data.sh

# user-data.sh
#!/bin/bash
apt update && apt install -y docker.io
systemctl enable docker
systemctl start docker
docker run -d \
  --name translator \
  -p 8080:8080 \
  -e ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY} \
  translator/server:latest
```

#### ECS Fargate

```json
{
  "family": "translator",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "translator",
      "image": "translator/server:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        }
      ],
      "secrets": [
        {
          "name": "ANTHROPIC_API_KEY",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:translator/api-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/translator",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### Google Cloud Platform

#### Cloud Run

```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT_ID/translator

# Deploy to Cloud Run
gcloud run deploy translator \
  --image gcr.io/PROJECT_ID/translator \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --memory 1Gi \
  --cpu 1 \
  --max-instances 100 \
  --set-env-vars ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY
```

#### GKE Deployment

```bash
# Create cluster
gcloud container clusters create translator-cluster \
  --num-nodes 3 \
  --machine-type e2-standard-2 \
  --region us-central1

# Deploy
kubectl apply -f k8s/
```

### Azure Deployment

#### Container Instances

```bash
# Create resource group
az group create --name translator-rg --location eastus

# Deploy container
az container create \
  --resource-group translator-rg \
  --name translator \
  --image translator/server:latest \
  --cpu 1 \
  --memory 2 \
  --ports 8080 \
  --environment-variables ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  --dns-name-label translator-unique
```

## Security Configuration

### API Key Management

```bash
# Use environment variables
export ANTHROPIC_API_KEY="your_api_key_here"

# Or use secret management
kubectl create secret generic translator-secrets \
  --from-literal=anthropic-api-key="your_api_key_here"

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name translator/api-key \
  --secret-string "your_api_key_here"
```

### Network Security

```bash
# Firewall rules
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# Docker network isolation
docker network create --driver bridge translator-net
docker run --network translator-net translator/server
```

### SSL/TLS Configuration

```nginx
# Strong SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
```

## Monitoring and Logging

### Prometheus Metrics

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'translator'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: /metrics
    scrape_interval: 5s
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Translator Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

### Log Aggregation

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/translator/*.log
  fields:
    service: translator
  fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

## Backup and Recovery

### Database Backup

```bash
# PostgreSQL backup
pg_dump translator > backup_$(date +%Y%m%d_%H%M%S).sql

# Automated backup
0 2 * * * pg_dump translator | gzip > /backups/translator_$(date +\%Y\%m\%d).sql.gz
```

### File Backup

```bash
# Backup configuration and data
tar -czf translator_backup_$(date +%Y%m%d).tar.gz \
  /etc/translator \
  /var/lib/translator \
  /var/log/translator

# S3 backup
aws s3 sync /var/lib/translator s3://translator-backups/$(date +%Y%m%d)/
```

### Disaster Recovery

```bash
# Restore from backup
gunzip -c translator_20231101.sql.gz | psql translator

# Restore files
tar -xzf translator_backup_20231101.tar.gz -C /
```

## Performance Optimization

### Database Optimization

```sql
-- Create indexes
CREATE INDEX idx_translations_created_at ON translations(created_at);
CREATE INDEX idx_files_status ON files(status);

-- Optimize configuration
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
```

### Caching Strategy

```json
{
  "cache": {
    "type": "redis",
    "connection_string": "redis://redis:6379",
    "ttl": "1h",
    "max_memory": "512mb",
    "eviction_policy": "allkeys-lru"
  }
}
```

### Load Balancing

```nginx
upstream translator_backend {
    least_conn;
    server translator1:8080 max_fails=3 fail_timeout=30s;
    server translator2:8080 max_fails=3 fail_timeout=30s;
    server translator3:8080 max_fails=3 fail_timeout=30s;
}
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check logs
journalctl -u translator-server -f

# Check configuration
translator-server --config /etc/translator/config.json --validate

# Check ports
netstat -tlnp | grep :8080
```

#### High Memory Usage

```bash
# Monitor memory
top -p $(pgrep translator-server)

# Check for memory leaks
valgrind --tool=memcheck --leak-check=full translator-server

# Optimize configuration
{
  "server": {
    "max_connections": 1000,
    "connection_timeout": "30s"
  }
}
```

#### Database Connection Issues

```bash
# Test connection
psql -h localhost -U translator -d translator

# Check connection pool
SELECT * FROM pg_stat_activity WHERE datname = 'translator';
```

### Health Checks

```bash
# API health check
curl -f http://localhost:8080/health

# Deep health check
curl -f http://localhost:8080/health/deep

# Component status
curl -f http://localhost:8080/health/components
```

### Performance Monitoring

```bash
# CPU and memory usage
htop

# Network connections
ss -tuln

# Disk usage
df -h

# Process monitoring
ps aux | grep translator
```

## Conclusion

This deployment guide covers all aspects of deploying the Universal Multi-Format Multi-Language Ebook Translation System in production environments. Choose the deployment option that best fits your requirements:

- **Small deployments**: Single-server or Docker
- **Medium deployments**: Distributed with multiple workers
- **Large deployments**: Kubernetes or cloud platform services

For additional support, refer to the troubleshooting guide or contact the support team.