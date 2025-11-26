#!/usr/bin/env python3
import sys
import json
import os
import subprocess
import time

def load_config():
    """Load llama.cpp configuration from JSON file"""
    try:
        with open('config.json', 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading config: {e}")
        return None

def get_available_models():
    """Check what llama.cpp models are available on the system"""
    models = []
    common_paths = [
        '/tmp/translate-ssh/models',
        '/home/milosvasic/models',
        '/usr/local/models',
        './models'
    ]
    
    for path in common_paths:
        if os.path.exists(path):
            for file in os.listdir(path):
                if file.endswith('.gguf'):
                    models.append({
                        'path': os.path.join(path, file),
                        'name': file.replace('.gguf', ''),
                        'id': file.replace('.gguf', '')
                    })
    
    return models

def build_llama_command(model_path, text, from_lang="ru", to_lang="sr"):
    """Build llama.cpp command for translation"""
    # Use correct llama.cpp binary path
    llama_binary = './llama.cpp'
    if not os.path.exists(llama_binary):
        llama_binary = '/home/milosvasic/llama.cpp/build/tools/main'
    
    if not os.path.exists(llama_binary):
        raise Exception(f"llama.cpp binary not found at {llama_binary}")
    
    # Construct proper translation prompt
    prompt = f"""Translate the following text from {from_lang.upper()} to {to_lang.upper()}. 
Provide ONLY the translation without any explanations, notes, or additional text.
Maintain the original formatting, line breaks, and structure perfectly.

Source text:
{text}

Translation:"""
    
    # Build llama.cpp command with proper parameters
    cmd = [
        llama_binary,
        '-m', model_path,
        '-p', prompt,
        '--ctx-size', '4096',
        '--temp', '0.3',
        '--top-p', '0.9',
        '--top-k', '40',
        '--repeat-penalty', '1.1',
        '--color',
        '-n', '2048'  # Max tokens to generate
    ]
    
    return cmd, prompt

def translate_with_llama(text, from_lang="ru", to_lang="sr"):
    """Translate text using llama.cpp"""
    models = get_available_models()
    
    if not models:
        print("No llama.cpp models found")
        return None
    
    # Use first available model (could be enhanced to select best model)
    model = models[0]
    print(f"Using model: {model['name']}")
    
    cmd, prompt = build_llama_command(model['path'], text, from_lang, to_lang)
    
    try:
        print("Executing llama.cpp...")
        start_time = time.time()
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=60,  # 60 second timeout
            cwd='/tmp/translate-ssh'
        )
        
        duration = time.time() - start_time
        print(f"Translation completed in {duration:.2f} seconds")
        
        if result.returncode != 0:
            print(f"llama.cpp failed: {result.stderr}")
            return None
        
        # Parse output to extract translation
        translation = parse_llama_output(result.stdout, prompt)
        return translation
        
    except subprocess.TimeoutExpired:
        print("Translation timed out")
        return None
    except Exception as e:
        print(f"Translation error: {e}")
        return None

def parse_llama_output(output, original_prompt):
    """Parse llama.cpp output to extract translation"""
    # Split output by lines and look for the translation part
    lines = output.split('\n')
    
    # Find where the actual translation starts (after "Translation:" in prompt)
    translation_lines = []
    found_translation = False
    
    for line in lines:
        line = line.strip()
        
        # Skip ANSI color codes and llama.cpp status messages
        if line.startswith('\x1b[') or line == '>' or not line:
            continue
            
        if found_translation:
            # We're in the translation part
            translation_lines.append(line)
        elif "Translation:" in line:
            # Found the translation marker
            found_translation = True
            # Extract text after "Translation:"
            parts = line.split("Translation:", 1)
            if len(parts) > 1:
                remaining = parts[1].strip()
                if remaining:
                    translation_lines.append(remaining)
    
    translation = '\n'.join(translation_lines)
    return translation.strip()

def translate_markdown_paragraphs(input_file, output_file):
    """Translate markdown file paragraph by paragraph using llama.cpp"""
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading input file: {e}")
        return False
    
    # Split content into paragraphs (preserve empty lines)
    paragraphs = content.split('\n\n')
    translated_paragraphs = []
    
    print(f"Translating {len(paragraphs)} paragraphs...")
    
    for i, paragraph in enumerate(paragraphs):
        if not paragraph.strip():
            # Preserve empty paragraphs
            translated_paragraphs.append(paragraph)
            continue
            
        print(f"Translating paragraph {i+1}/{len(paragraphs)}...")
        
        # Handle line-by-line translation within paragraphs
        lines = paragraph.split('\n')
        translated_lines = []
        
        for line in lines:
            if line.strip():
                # Translate non-empty lines
                # Skip markdown headers, code blocks, etc.
                if (line.startswith('#') or 
                    line.startswith('>') or 
                    line.startswith('```') or 
                    line.startswith('    ')):
                    translated_lines.append(line)
                else:
                    translated = translate_with_llama(line)
                    if translated:
                        translated_lines.append(translated)
                    else:
                        translated_lines.append(line)  # Fallback to original
            else:
                # Preserve empty lines
                translated_lines.append(line)
        
        translated_paragraph = '\n'.join(translated_lines)
        translated_paragraphs.append(translated_paragraph)
        
        # Small delay between paragraphs to avoid overwhelming the system
        time.sleep(0.1)
    
    # Write translated content
    try:
        translated_content = '\n\n'.join(translated_paragraphs)
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(translated_content)
        
        print(f"Translation completed: {len(content)} -> {len(translated_content)} characters")
        return True
        
    except Exception as e:
        print(f"Error writing output file: {e}")
        return False

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 translate_with_llama.py <input.md> <output.md>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    print(f"Starting llama.cpp translation from {input_file} to {output_file}")
    
    # Check if llama.cpp binary exists, try to setup if not
    llama_binary = './llama.cpp'
    if not os.path.exists(llama_binary):
        llama_binary = '/home/milosvasic/llama.cpp/build/tools/main'
    
    if not os.path.exists(llama_binary):
        print(f"llama.cpp binary not found at {llama_binary}, attempting automatic setup...")
        # Try to setup llama.cpp automatically
        try:
            import subprocess
            result = subprocess.run([sys.executable, 'setup_llama.py', 'setup'], 
                                  capture_output=True, text=True, timeout=300)
            if result.returncode == 0 and os.path.exists('./llama.cpp'):
                llama_binary = './llama.cpp'
                print("llama.cpp setup completed successfully")
            else:
                print(f"Automatic setup failed: {result.stderr}")
                sys.exit(1)
        except Exception as e:
            print(f"Failed to setup llama.cpp automatically: {e}")
            sys.exit(1)
    
    # Load configuration
    config = load_config()
    if config:
        print(f"Loaded config: {config}")
    
    # Perform translation
    if translate_markdown_paragraphs(input_file, output_file):
        print("Translation completed successfully")
    else:
        print("Translation failed")
        sys.exit(1)

if __name__ == "__main__":
    main()