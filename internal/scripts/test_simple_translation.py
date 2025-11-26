#!/usr/bin/env python3

import os
import subprocess

def test_translation():
    """Test simple Russian to Serbian translation"""
    
    # Test with simple Russian text
    text = "Привет мир"
    
    # Find llama.cpp binary
    llama_binary = None
    for path in ['./llama.cpp', '/tmp/translate-ssh/llama.cpp', '/home/milosvasic/llama.cpp/build/tools/main']:
        if os.path.exists(path):
            llama_binary = path
            break
    
    if not llama_binary:
        print("llama.cpp binary not found")
        return False
    
    # Find model
    model_path = None
    for path in ['/home/milosvasic/models', '/tmp/translate-ssh/models', './models']:
        if os.path.exists(path):
            for file in os.listdir(path):
                if file.endswith('.gguf'):
                    model_path = os.path.join(path, file)
                    break
            if model_path:
                break
    
    if not model_path:
        print("No GGUF model found")
        return False
    
    print(f"Using llama binary: {llama_binary}")
    print(f"Using model: {model_path}")
    
    # Build very simple prompt
    prompt = f"""Translate Russian to Serbian: {text}
Translation:"""
    
    # Build command
    cmd = [
        llama_binary,
        '-m', model_path,
        '-p', prompt,
        '-n', '50'
    ]
    
    print(f"Running command: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        
        if result.returncode != 0:
            print(f"llama.cpp failed with code {result.returncode}")
            print(f"STDERR: {result.stderr}")
            return False
        
        print(f"Raw output: {result.stdout}")
        
        # Extract translation
        if "Translation:" in result.stdout:
            parts = result.stdout.split("Translation:", 1)
            if len(parts) > 1:
                translation = parts[1].strip()
                print(f"Extracted translation: {translation}")
                return translation
        
        print("No translation found in output")
        return False
        
    except Exception as e:
        print(f"Error: {e}")
        return False

if __name__ == "__main__":
    result = test_translation()
    if result:
        print("SUCCESS")
    else:
        print("FAILED")