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

def extract_and_translate_text(element, path=""):
    """Recursively extract and translate text from FB2 elements"""
    new_element = ET.Element(element.tag, element.attrib)
    
    # Process text content
    if element.text and element.text.strip():
        text = element.text.strip()
        if TRANSLATE_ENABLED:
            try:
                # Translate to Serbian
                result = translator.translate(text, dest='sr')
                translated_text = result.text
                new_element.text = translated_text
                print(f"Translated: {text[:30]}... -> {translated_text[:30]}...")
            except Exception as e:
                print(f"Translation error: {e}")
                new_element.text = element.text
        else:
            # Mark for manual translation
            new_element.text = f"[TRANSLATE: {text}]"
    else:
        new_element.text = element.text
    
    # Process child elements
    for child in element:
        new_child = extract_and_translate_text(child, path + "/" + element.tag)
        new_element.append(new_child)
    
    # Process tail content
    if element.tail and element.tail.strip():
        tail_text = element.tail.strip()
        if TRANSLATE_ENABLED:
            try:
                result = translator.translate(tail_text, dest='sr')
                translated_tail = result.text
                new_element.tail = translated_tail
            except Exception as e:
                new_element.tail = element.tail
        else:
            new_element.tail = f"[TRANSLATE: {tail_text}]"
    else:
        new_element.tail = element.tail
    
    return new_element

def translate_fb2_file(input_path, output_path):
    """Translate an FB2 file from Russian to Serbian"""
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the file
        tree = ET.parse(input_path)
        root = tree.getroot()
        
        # Update document language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'  # Set language to Serbian
        
        # Process all elements
        translated_root = extract_and_translate_text(root)
        
        # Create new tree and write to file
        new_tree = ET.ElementTree(translated_root)
        new_tree.write(output_path, encoding='utf-8', xml_declaration=True)
        
        print(f"Successfully processed {input_path} -> {output_path}")
        return True
        
    except Exception as e:
        print(f"Error processing file: {e}")
        return False

def main():
    input_file = "Ratibor_1f.b2"
    output_file = "Ratibor_1f_sr.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    # Create output filename based on whether we're translating or creating template
    if not TRANSLATE_ENABLED:
        output_file = "Ratibor_1f_template.b2"
        print("Creating translation template...")
    else:
        print("Starting translation to Serbian...")
    
    success = translate_fb2_file(input_file, output_file)
    
    if success and not TRANSLATE_ENABLED:
        print("\nTemplate created! Replace [TRANSLATE: text] with Serbian translations.")
        print("After translation, rename the file to .fb2 format for reading.")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)