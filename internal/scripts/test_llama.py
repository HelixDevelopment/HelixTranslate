#!/usr/bin/env python3

import sys
import os
import subprocess

def test_llama():
    """Test llama.cpp functionality"""
    print("Testing llama.cpp translation...")
    
    # Test simple Russian to Serbian translation
    test_text = "Привет мир"
    
    # Find llama.cpp binary
    llama_binary = './llama.cpp'
    if not os.path.exists(llama_binary):
        llama_binary = '/home/milosvasic/llama.cpp'
    
    if not os.path.exists(llama_binary):
        print(f"llama.cpp binary not found at {llama_binary}")
        return False
    
    # Find model
    model_path = '/home/milosvasic/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf'
    if not os.path.exists(model_path):
        print(f"Model not found at {model_path}")
        return False
    
    # Build prompt
    prompt = f"""Translate the following text from RU to SR. 
Provide ONLY the translation without any explanations, notes, or additional text.

Source text:
{test_text}

Translation:"""
    
    # Build command
    cmd = [
        llama_binary,
        '-m', model_path,
        '-p', prompt,
        '--ctx-size', '4096',
        '--temp', '0.3',
        '-n', '100'
    ]
    
    print(f"Running: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        
        if result.returncode == 0:
            print("llama.cpp executed successfully")
            print("STDOUT:")
            print(result.stdout)
            if result.stderr:
                print("STDERR:")
                print(result.stderr)
            return True
        else:
            print(f"llama.cpp failed with exit code {result.returncode}")
            print("STDERR:", result.stderr)
            return False
            
    except Exception as e:
        print(f"Error running llama.cpp: {e}")
        return False

if __name__ == "__main__":
    if test_llama():
        print("llama.cpp test passed")
        sys.exit(0)
    else:
        print("llama.cpp test failed")
        sys.exit(1)