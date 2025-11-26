#!/bin/bash
# Deploy translator to remote SSH worker

set -e

# Configuration
HOST="thinker.local"
USER="milosvasic"
PASS="WhiteSnake8587"
REMOTE_DIR="/tmp/translator-deploy"

echo "Deploying translator to SSH worker..."

# Create temporary directory
mkdir -p $REMOTE_DIR

# Build all binaries for Linux target
echo "Building binaries..."
GOOS=linux GOARCH=amd64 go build -o $REMOTE_DIR/translator ./cmd/cli
GOOS=linux GOARCH=amd64 go build -o $REMOTE_DIR/translator-ssh ./cmd/translate-ssh
GOOS=linux GOARCH=amd64 go build -o $REMOTE_DIR/translator-server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o $REMOTE_DIR/markdown-translator ./cmd/markdown-translator

# Copy necessary files
echo "Copying configuration files..."
cp go.mod go.sum $REMOTE_DIR/
cp -r pkg $REMOTE_DIR/
cp -r cmd $REMOTE_DIR/
cp -r internal $REMOTE_DIR/
cp -r scripts $REMOTE_DIR/
cp -r docs $REMOTE_DIR/
cp Dockerfile Makefile $REMOTE_DIR/

# Test SSH connection and create remote directory
echo "Testing SSH connection..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no $USER@$HOST "mkdir -p /tmp/translate-ssh"

# Upload using sshpass
echo "Uploading files to remote..."
sshpass -p "$PASS" scp -r -o StrictHostKeyChecking=no $REMOTE_DIR/* $USER@$HOST:/tmp/translate-ssh/

# Set executable permissions
echo "Setting permissions..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no $USER@$HOST "chmod +x /tmp/translate-ssh/translator"

# Test remote hash calculation
echo "Testing remote hash calculation..."
REMOTE_HASH=$(sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no $USER@$HOST "cd /tmp/translate-ssh && ./translator hash-codebase" 2>&1)
echo "Remote hash: $REMOTE_HASH"

# Clean up local temp directory
rm -rf $REMOTE_DIR

echo "Deployment completed successfully!"