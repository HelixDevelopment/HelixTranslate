#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import sys
import time
from pathlib import Path

TRANSLATE_ENABLED = False
Translator = None
try:
    from googletrans import Translator
    TRANSLATE_ENABLED = True
    print("Google Translate enabled")
except ImportError:
    print("Google Translate not available")

def translate_text_batch(texts, translator=None):
    """Translate a batch of texts from Russian to Serbian"""
    if not TRANSLATE_ENABLED or not translator:
        return [text for text in texts]  # Return originals if no translator
    
    translations = []
    for i, text in enumerate(texts):
        try:
            if text.strip() and len(text.strip()) > 2:
                result = translator.translate(text, dest='sr', src='ru')
                translated = result.text if hasattr(result, 'text') else str(result)
                translations.append(translated)
                print(".", end="", flush=True)
            else:
                translations.append(text)
        except Exception as e:
            print(f"E", end="", flush=True)
            translations.append(text)  # Keep original on error
        time.sleep(0.1)  # Rate limiting
    
    return translations

def process_translation_list(input_file, output_file):
    """Process translation list and auto-translate empty entries"""
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            lines = f.readlines()
        
        translator = None
        if TRANSLATE_ENABLED and Translator:
            try:
                translator = Translator()
            except Exception as e:
                print(f"Warning: Could not initialize translator: {e}")
        
        updated_lines = []
        i = 0
        
        while i < len(lines):
            line = lines[i]
            updated_lines.append(line)
            
            if line.startswith('RU:'):
                ru_text = line[3:].strip()
                if i + 1 < len(lines) and lines[i + 1].startswith('SR:'):
                    sr_line = lines[i + 1]
                    sr_text = sr_line[3:].strip()
                    
                    if not sr_text:  # Only translate if empty
                        if TRANSLATE_ENABLED and translator:
                            try:
                                result = translator.translate(ru_text, dest='sr', src='ru')
                                translated = result.text if hasattr(result, 'text') else str(result)
                                updated_lines.append(f"SR: {translated}\n")
                                print(".", end="", flush=True)
                            except Exception as e:
                                print(f"E", end="", flush=True)
                                updated_lines.append(sr_line)  # Keep original on error
                        else:
                            updated_lines.append(sr_line)
                    else:
                        updated_lines.append(sr_line)  # Keep existing translation
                    i += 2  # Skip SR line as we processed it
                else:
                    i += 1
            else:
                i += 1
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.writelines(updated_lines)
        
        print(f"\nAuto-translation completed: {output_file}")
        return True
        
    except Exception as e:
        print(f"Error processing translation list: {e}")
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 auto_translate_list.py <translation_list.txt> [output_file.txt]")
        return False
    
    input_file = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) >= 3 else input_file.replace('.txt', '_auto.txt')
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    print("Auto-translating empty Serbian entries...")
    success = process_translation_list(input_file, output_file)
    
    if success:
        print("Auto-translation completed!")
        print(f"Output: {output_file}")
        print("Review the translations and apply with translation_helper.py option 2")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)