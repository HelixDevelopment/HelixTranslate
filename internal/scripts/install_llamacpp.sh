#!/bin/bash

# Automatic llama.cpp installation and setup script
# Works on Ubuntu/Debian and CentOS/RHEL systems

set -e

INSTALL_DIR="/tmp/translate-ssh"
LLAMA_VERSION="b3732"

echo "Installing llama.cpp automatically..."

# Check if llama.cpp already exists
if [ -f "$INSTALL_DIR/llama.cpp" ] || [ -f "/home/milosvasic/llama.cpp" ]; then
    echo "llama.cpp binary already exists, skipping installation"
    exit 0
fi

# Install dependencies based on package manager
if command -v apt-get >/dev/null 2>&1; then
    echo "Using apt-get package manager (Ubuntu/Debian)"
    sudo apt-get update
    sudo apt-get install -y build-essential cmake git wget
elif command -v yum >/dev/null 2>&1; then
    echo "Using yum package manager (CentOS/RHEL)"
    sudo yum groupinstall -y "Development Tools"
    sudo yum install -y cmake git wget
else
    echo "Unsupported package manager. Please install build-essential, cmake, git, and wget manually."
    exit 1
fi

# Download and build llama.cpp
cd /tmp
rm -rf llama.cpp 2>/dev/null || true

echo "Downloading llama.cpp..."
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
git checkout $LLAMA_VERSION

echo "Building llama.cpp (this may take several minutes)..."
make LLAMA_CUBLAS=1 -j$(nproc)

# Copy binary to installation directory
cp llama "$INSTALL_DIR/llama.cpp"
chmod +x "$INSTALL_DIR/llama.cpp"

echo "llama.cpp installation completed successfully!"
echo "Binary installed at: $INSTALL_DIR/llama.cpp"