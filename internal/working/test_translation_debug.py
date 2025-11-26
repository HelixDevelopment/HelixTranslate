#!/usr/bin/env python3
"""
Test script to debug llama.cpp translation
"""
import os
import sys
import subprocess

# Add the current directory to Python path
sys.path.insert(0, '/tmp/translate-ssh')

# Import the translation functions
from translate_llm_only import translate_text, get_translation_provider

def test_simple_translation():
    """Test translation of a simple Russian phrase"""
    test_text = "Переведите меня на сербский"
    
    print("Testing translation:")
    print(f"Input: {test_text}")
    
    # Check what provider is available
    provider, config = get_translation_provider()
    print(f"Provider: {provider}")
    print(f"Config: {config}")
    
    try:
        result = translate_text(test_text, "ru", "sr")
        print(f"Output: {result}")
        
        if result == test_text:
            print("ERROR: Translation returned original text!")
        else:
            print("SUCCESS: Text was translated!")
    except Exception as e:
        print(f"Translation failed: {e}")

if __name__ == "__main__":
    test_simple_translation()