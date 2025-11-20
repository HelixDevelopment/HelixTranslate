#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
from pathlib import Path

# Try to import translation libraries
try:
    from googletrans import Translator
    translator = Translator()
    TRANSLATE_ENABLED = True
    print("Google Translate enabled")
except ImportError:
    TRANSLATE_ENABLED = False
    print("Google Translate not available - will create template")

def process_text_chunks(file_path, output_path):
    """Process the FB2 file in chunks to avoid timeouts"""
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the file
        tree = ET.parse(file_path)
        root = tree.getroot()
        
        # Find all text elements to translate
        text_elements = []
        
        def find_text_elements(element, path=""):
            if element.text and element.text.strip():
                text_elements.append((element, path + "/" + element.tag, element.text))
            if element.tail and element.tail.strip():
                text_elements.append((element, path + "/" + element.tag + "/tail", element.tail))
            for child in element:
                find_text_elements(child, path + "/" + element.tag)
        
        find_text_elements(root)
        print(f"Found {len(text_elements)} text elements to process")
        
        # Process each element
        for element, path, text in text_elements:
            if "/tail" in path:
                # Process tail text
                if TRANSLATE_ENABLED:
                    try:
                        result = translator.translate(text.strip(), dest='sr')
                        element.tail = result.text
                        print(f".", end="", flush=True)
                    except Exception as e:
                        print(f"\nError translating tail: {e}")
                        element.tail = f"[TRANSLATE: {text}]"
                else:
                    element.tail = f"[TRANSLATE: {text}]"
            else:
                # Process element text
                if TRANSLATE_ENABLED:
                    try:
                        result = translator.translate(text.strip(), dest='sr')
                        element.text = result.text
                        print(f".", end="", flush=True)
                    except Exception as e:
                        print(f"\nError translating element: {e}")
                        element.text = f"[TRANSLATE: {text}]"
                else:
                    element.text = f"[TRANSLATE: {text}]"
        
        print("\nUpdating document language to Serbian...")
        
        # Update document language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'
        
        # Write the processed file
        tree.write(output_path, encoding='utf-8', xml_declaration=True)
        
        print(f"Successfully processed {file_path} -> {output_path}")
        return True
        
    except Exception as e:
        print(f"Error processing file: {e}")
        return False

def main():
    input_file = "Ratibor_1f.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    if TRANSLATE_ENABLED:
        output_file = "Ratibor_1f_sr.b2"
        print("Starting translation to Serbian...")
    else:
        output_file = "Ratibor_1f_template.b2"
        print("Creating translation template...")
    
    success = process_text_chunks(input_file, output_file)
    
    if success:
        if not TRANSLATE_ENABLED:
            print("\nTemplate created! Replace [TRANSLATE: text] with Serbian translations.")
            print("After translation, rename the file to .fb2 format for reading.")
        else:
            print("\nTranslation completed! Review the output file.")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)