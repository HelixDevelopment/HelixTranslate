#!/usr/bin/env python3

import sys
import os
import json
import subprocess
import requests
import tempfile
import shutil

def check_llama_binary():
    """Check if llama.cpp binary exists in common locations"""
    locations = [
        '/tmp/translate-ssh/llama.cpp',
        '/home/milosvasic/llama.cpp',
        './llama.cpp',
        '/usr/local/bin/llama',
        '/usr/bin/llama'
    ]
    
    for loc in locations:
        if os.path.exists(loc):
            print(f"Found llama.cpp binary at: {loc}")
            return loc
    return None

def download_llama_binary():
    """Download pre-compiled llama.cpp binary"""
    print("Downloading pre-compiled llama.cpp binary...")
    
    # Common download URLs for different architectures
    urls = [
        "https://github.com/ggerganov/llama.cpp/releases/latest/download/llama-ubuntu-x64",  # Ubuntu x64
        "https://huggingface.co/TheBloke/Llama-2-7B-Chat-GGUF/resolve/main/llama-cpp-bin/llama-ubuntu",  # Backup
    ]
    
    for url in urls:
        try:
            print(f"Trying to download from: {url}")
            response = requests.get(url, stream=True, timeout=300)
            if response.status_code == 200:
                # Save to /tmp/translate-ssh/llama.cpp
                os.makedirs('/tmp/translate-ssh', exist_ok=True)
                binary_path = '/tmp/translate-ssh/llama.cpp'
                
                with open(binary_path, 'wb') as f:
                    for chunk in response.iter_content(chunk_size=8192):
                        f.write(chunk)
                
                os.chmod(binary_path, 0o755)
                print(f"Successfully downloaded llama.cpp to {binary_path}")
                return binary_path
        except Exception as e:
            print(f"Failed to download from {url}: {e}")
            continue
    
    return None

def compile_llama_cpp():
    """Compile llama.cpp from source"""
    print("Compiling llama.cpp from source...")
    
    try:
        # Create temporary directory
        with tempfile.TemporaryDirectory() as temp_dir:
            # Clone repository
            subprocess.run([
                'git', 'clone', 
                'https://github.com/ggerganov/llama.cpp.git',
                temp_dir + '/llama.cpp'
            ], check=True, capture_output=True)
            
            # Build
            build_dir = temp_dir + '/llama.cpp'
            subprocess.run([
                'make', '-j', str(os.cpu_count())
            ], cwd=build_dir, check=True, capture_output=True)
            
            # Copy binary
            os.makedirs('/tmp/translate-ssh', exist_ok=True)
            shutil.copy(build_dir + '/llama', '/tmp/translate-ssh/llama.cpp')
            os.chmod('/tmp/translate-ssh/llama.cpp', 0o755)
            
            print("Successfully compiled llama.cpp")
            return '/tmp/translate-ssh/llama.cpp'
            
    except Exception as e:
        print(f"Failed to compile llama.cpp: {e}")
        return None

def setup_llama():
    """Main setup function"""
    print("Setting up llama.cpp...")
    
    # Check if binary already exists
    binary = check_llama_binary()
    if binary:
        return binary
    
    # Try to download pre-compiled binary
    binary = download_llama_binary()
    if binary:
        return binary
    
    # Try to compile from source
    binary = compile_llama_cpp()
    if binary:
        return binary
    
    print("Failed to setup llama.cpp")
    return None

def find_models():
    """Find available GGUF models"""
    model_paths = [
        '/tmp/translate-ssh/models',
        '/home/milosvasic/models',
        '/usr/local/models',
        './models'
    ]
    
    models = []
    for path in model_paths:
        if os.path.exists(path):
            for file in os.listdir(path):
                if file.endswith('.gguf'):
                    models.append(os.path.join(path, file))
    
    return models

if __name__ == "__main__":
    if len(sys.argv) != 2 or sys.argv[1] != "setup":
        print("Usage: python3 setup_llama.py setup")
        sys.exit(1)
    
    print("Automatic llama.cpp setup starting...")
    
    # Setup llama.cpp binary
    binary = setup_llama()
    if not binary:
        print("Failed to setup llama.cpp binary")
        sys.exit(1)
    
    # Check for models
    models = find_models()
    if models:
        print(f"Found {len(models)} models:")
        for model in models:
            print(f"  - {model}")
    else:
        print("No GGUF models found. You may need to download a model.")
        print("Example: wget https://huggingface.co/TheBloke/Llama-2-7B-Chat-GGUF/resolve/main/llama-2-7b-chat.Q4_K_M.gguf")
    
    print("Setup completed successfully!")