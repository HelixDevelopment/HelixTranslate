#!/usr/bin/env python3
"""
PURE LLM Translation System - NO CHARACTER FALLBACKS
Uses ONLY LLMs (llama.cpp/OpenAI/Anthropic) for Russian to Serbian translation
"""

import sys
import os
import subprocess
import time
import json
import requests

def get_translation_provider():
    """Auto-detect and return best available translation provider"""
    
    # Priority 1: Local llama.cpp
    llama_binary = find_llama_binary()
    if llama_binary and has_llama_model():
        return "llamacpp", llama_binary
    
    # Priority 2: API providers (check for API keys/configs)
    if os.path.exists('config.json'):
        try:
            with open('config.json', 'r') as f:
                config = json.load(f)
                if 'openai' in config and config['openai'].get('api_key'):
                    return "openai", config['openai']
                if 'anthropic' in config and config['anthropic'].get('api_key'):
                    return "anthropic", config['anthropic']
        except:
            pass
    
    return None, None

def find_llama_binary():
    """Find llama.cpp binary in common locations"""
    paths = [
        '/home/milosvasic/llama.cpp/build/bin/llama-cli',
        '/home/milosvasic/llama.cpp/build/tools/main',
        '/tmp/translate-ssh/llama.cpp',
        '/usr/local/bin/llama',
        './llama.cpp',
        'llama'
    ]
    
    for path in paths:
        if os.path.exists(path) and os.access(path, os.X_OK):
            return path
    return None

def has_llama_model():
    """Check if we have GGUF models available"""
    model_paths = [
        '/tmp/translate-ssh/models',
        '/home/milosvasic/models',
        './models'
    ]
    
    for path in model_paths:
        if os.path.exists(path):
            for file in os.listdir(path):
                if file.endswith('.gguf'):
                    return True
    return False

def translate_with_llamacpp(text, from_lang="ru", to_lang="sr"):
    """Translate using local llama.cpp - PURE LLM translation only"""
    llama_binary = find_llama_binary()
    if not llama_binary:
        raise Exception("llama.cpp binary not found")
    
    model_path = "/home/milosvasic/models/tiny-llama-working.gguf"
    if not os.path.exists(model_path):
        raise Exception("No working GGUF model found")
    
    # Build strong translation prompt
    prompt = f"""You are a professional translator from Russian to Serbian. 
Translate the following text. Provide ONLY the Serbian translation, no explanations.
Preserve formatting, structure, and paragraph breaks.

Original text:
{text}

Serbian translation:"""
    
    cmd = [
        llama_binary,
        '-m', model_path,
        '--n-gpu-layers', '0',
        '-p', prompt,
        '--ctx-size', '2048',
        '--temp', '0.1',  # Lower temperature for consistency
        '-n', '2048'     # Allow longer output
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
        if result.returncode != 0:
            raise Exception(f"llama.cpp failed: {result.stderr}")
        
        # Extract text after "Serbian translation:" marker
        lines = result.stdout.split('\n')
        translation_started = False
        result_lines = []
        
        for line in lines:
            if "Serbian translation:" in line:
                translation_started = True
                # Get everything after the marker
                parts = line.split("Serbian translation:", 1)
                if len(parts) > 1 and parts[1].strip():
                    result_lines.append(parts[1].strip())
                continue
            elif translation_started and line.strip():
                # Stop if we hit the original text again
                if "Original text:" in line or line.startswith(text[:20]):
                    break
                result_lines.append(line.strip())
        
        result_text = '\n'.join(result_lines).strip()
        
        # If no valid translation found, it's LLM failure - NO fallbacks
        if not result_text or result_text.lower() == text.lower()[:len(result_text)]:
            raise Exception("LLM failed to provide valid Serbian translation")
        
        return result_text
        
    except subprocess.TimeoutExpired:
        raise Exception("Translation timed out")

def translate_with_openai(text, from_lang="ru", to_lang="sr", api_key=None):
    """Translate using OpenAI API - PURE LLM translation only"""
    if not api_key:
        raise Exception("OpenAI API key required")
    
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "gpt-4",
        "messages": [
            {
                "role": "system",
                "content": "You are a professional translator. Translate from Russian to Serbian. Provide ONLY the Serbian translation, no explanations, no commentary, no alternative suggestions."
            },
            {
                "role": "user", 
                "content": f"Translate this Russian text to Serbian:\n\n{text}"
            }
        ],
        "temperature": 0.1
    }
    
    try:
        response = requests.post(
            "https://api.openai.com/v1/chat/completions",
            headers=headers,
            json=data,
            timeout=120
        )
        response.raise_for_status()
        
        result = response.json()
        translation = result['choices'][0]['message']['content'].strip()
        
        # Verify it's actually different and not just copied
        if not translation or translation.lower() == text.lower():
            raise Exception("OpenAI failed to provide valid Serbian translation")
            
        return translation
        
    except Exception as e:
        raise Exception(f"OpenAI translation failed: {e}")

def translate_text(text, from_lang="ru", to_lang="sr"):
    """Translate text using LLMs only - NO fallbacks"""
    provider, config = get_translation_provider()
    
    if not provider:
        raise Exception("No LLM translation provider available. Please install llama.cpp or configure API keys.")
    
    print(f"Using LLM provider: {provider}")
    
    if provider == "llamacpp":
        return translate_with_llamacpp(text, from_lang, to_lang)
    elif provider == "openai":
        return translate_with_openai(text, from_lang, to_lang, config.get('api_key'))
    else:
        raise Exception(f"Unsupported provider: {provider}")

def translate_markdown_file(input_file, output_file, from_lang="ru", to_lang="sr"):
    """Translate markdown file paragraph by paragraph using LLMs ONLY"""
    
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading input file: {e}")
        return False
    
    # Split into paragraphs
    paragraphs = content.split('\n\n')
    translated_paragraphs = []
    failed_paragraphs = 0
    
    print(f"Translating {len(paragraphs)} paragraphs with LLMs only...")
    
    for i, paragraph in enumerate(paragraphs):
        if not paragraph.strip():
            translated_paragraphs.append(paragraph)
            continue
        
        # Skip markdown headers and code blocks (preserve as-is)
        if (paragraph.startswith('#') or 
            paragraph.startswith('```') or 
            paragraph.startswith('>')):
            translated_paragraphs.append(paragraph)
            continue
        
        print(f"Translating paragraph {i+1}/{len(paragraphs)}...")
        
        try:
            translated = translate_text(paragraph.strip(), from_lang, to_lang)
            # Verify translation is different and reasonable
            if translated and translated != paragraph.strip():
                translated_paragraphs.append(translated)
                print(f"✓ Paragraph {i+1} translated successfully")
            else:
                raise Exception("Translation failed - no change detected")
                
        except Exception as e:
            failed_paragraphs += 1
            print(f"✗ Paragraph {i+1} LLM translation failed: {e}")
            print(f"✗ FAILED PARAGRAPH {i+1}: {paragraph[:100]}...")
            # NO FALLBACKS - keep original to indicate failure
            translated_paragraphs.append(f"[TRANSLATION FAILED] {paragraph.strip()}")
    
    # Write translated content
    try:
        translated_content = '\n\n'.join(translated_paragraphs)
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(translated_content)
        
        print(f"Translation completed: {len(content)} -> {len(translated_content)} characters")
        print(f"Failed paragraphs: {failed_paragraphs}/{len(paragraphs)}")
        
        # Consider success only if >90% of paragraphs translated
        success_rate = (len(paragraphs) - failed_paragraphs) / len(paragraphs)
        if success_rate < 0.9:
            print(f"WARNING: Low success rate ({success_rate:.1%}). Consider better model or API.")
            return False
            
        return True
        
    except Exception as e:
        print(f"Error writing output file: {e}")
        return False

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 translate_llm_only.py <input.md> <output.md>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    if not os.path.exists(input_file):
        print(f"Input file not found: {input_file}")
        sys.exit(1)
    
    print(f"Starting PURE LLM translation from {input_file} to {output_file}")
    print("IMPORTANT: Using LLM translation ONLY - no character or dictionary fallbacks")
    
    # Show available providers
    provider, config = get_translation_provider()
    if provider:
        print(f"Using LLM provider: {provider}")
    else:
        print("ERROR: No LLM translation provider found")
        print("Please install llama.cpp or configure API keys in config.json")
        sys.exit(1)
    
    # Translate with LLMs only
    if translate_markdown_file(input_file, output_file):
        print("✓ LLM translation completed successfully")
    else:
        print("✗ LLM translation failed")
        sys.exit(1)

if __name__ == "__main__":
    main()