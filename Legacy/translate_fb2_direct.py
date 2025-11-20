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

def translate_fb2_direct(input_file, output_file):
    """Direct FB2 translation without intermediate templates"""
    global TRANSLATE_ENABLED, Translator
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse file
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        translator = None
        if TRANSLATE_ENABLED and Translator:
            try:
                translator = Translator()
            except Exception as e:
                print(f"Warning: Could not initialize translator: {e}")
                TRANSLATE_ENABLED = False
        
        translation_count = 0
        error_count = 0
        
        def translate_text(text):
            nonlocal translation_count, error_count
            if not text or not text.strip() or len(text.strip()) <= 2:
                return text
            
            if not TRANSLATE_ENABLED or not translator:
                return text
            
            try:
                result = translator.translate(text, dest='sr', src='ru')
                translated = result.text if hasattr(result, 'text') else str(result)
                translation_count += 1
                print(".", end="", flush=True)
                return translated
            except Exception as e:
                error_count += 1
                print("E", end="", flush=True)
                return text
        
        def process_element(element):
            # Process element text
            if element.text:
                element.text = translate_text(element.text)
            
            # Process children
            for child in element:
                process_element(child)
            
            # Process tail text
            if element.tail:
                element.tail = translate_text(element.tail)
        
        print("Translating FB2 content...")
        process_element(root)
        
        # Update document language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'
        
        # Write the translated file
        print(f"\nWriting output to: {output_file}")
        tree.write(output_file, encoding='utf-8', xml_declaration=True)
        
        print(f"\nTranslation completed!")
        print(f"Successfully translated: {translation_count} elements")
        print(f"Failed translations: {error_count} elements")
        print(f"Output file: {output_file}")
        
        return True
        
    except Exception as e:
        print(f"Error translating FB2: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 translate_fb2_direct.py <input_file.fb2> [output_file.b2]")
        print("Example: python3 translate_fb2_direct.py book_ru.fb2 book_sr.b2")
        return False
    
    input_file = sys.argv[1]
    
    # Generate output filename if not provided
    if len(sys.argv) >= 3:
        output_file = sys.argv[2]
    else:
        input_path = Path(input_file)
        stem = input_path.stem
        output_file = f"{stem}_sr_cyrillic.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    print("Starting direct FB2 translation...")
    success = translate_fb2_direct(input_file, output_file)
    
    if success:
        print("\nTranslation completed successfully!")
        print("Next steps:")
        print("1. Review the translation in an FB2 reader")
        print("2. Make manual adjustments if needed")
        print("3. Convert to EPUB/PDF if desired")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)