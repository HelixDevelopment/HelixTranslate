#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
from pathlib import Path

# Try to import googletrans, provide instructions if not available
try:
    from googletrans import Translator
    translator_available = True
except ImportError:
    translator_available = False
    print("Warning: googletrans library not found. Install with: pip3 install googletrans==4.0.0-rc1")

def extract_text_for_translation(element):
    """Extract text content from an element while preserving structure markers"""
    text_parts = []
    
    if element.text:
        text_parts.append(element.text.strip())
    
    for child in element:
        if child.tag.endswith('p') or child.tag.endswith('title'):
            # Process paragraph or title elements
            if child.text:
                text_parts.append(child.text.strip())
            if child.tail:
                text_parts.append(child.tail.strip())
        elif child.tag.endswith('emphasis') or child.tag.endswith('strong'):
            # Process emphasized text
            if child.text:
                text_parts.append(child.text.strip())
            if child.tail:
                text_parts.append(child.tail.strip())
        else:
            # Recursively process other elements
            text_parts.extend(extract_text_for_translation(child))
            
            if child.tail:
                text_parts.append(child.tail.strip())
    
    # Filter out empty strings and return
    return [part for part in text_parts if part]

def translate_text(text, target_lang='sr'):
    """Translate text to Serbian if translator is available"""
    if not translator_available or not text.strip():
        return text
    
    try:
        translator = Translator()
        result = translator.translate(text, dest=target_lang)
        return result.text
    except Exception as e:
        print(f"Translation error: {e}")
        return text  # Return original text if translation fails

def process_element(element, target_lang='sr'):
    """Process an element and translate its text content"""
    # Create a copy of the element to modify
    new_element = ET.Element(element.tag, element.attrib)
    
    # Process text content
    if element.text:
        # Try to translate the text if it's substantial enough
        if len(element.text.strip()) > 0:
            new_element.text = translate_text(element.text, target_lang)
        else:
            new_element.text = element.text
    
    # Process child elements
    for child in element:
        new_child = process_element(child, target_lang)
        new_element.append(new_child)
    
    # Process tail content (text after the element)
    if element.tail:
        if len(element.tail.strip()) > 0:
            new_element.tail = translate_text(element.tail, target_lang)
        else:
            new_element.tail = element.tail
    
    return new_element

def translate_fb2_file(input_file, output_file, target_lang='sr'):
    """Translate an FB2 file to the target language"""
    try:
        # Parse the XML file
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        # Create a new root element for the translated content
        new_root = process_element(root, target_lang)
        
        # Create a new tree with the translated content
        new_tree = ET.ElementTree(new_root)
        
        # Write the translated content to the output file
        new_tree.write(output_file, encoding='utf-8', xml_declaration=True)
        
        print(f"Successfully translated {input_file} to {output_file}")
        return True
    except Exception as e:
        print(f"Error translating file: {e}")
        return False

def main():
    input_file = "Ratibor_1f.b2"
    output_file = "Ratibor_1f_sr.b2"  # Serbian translation
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    if not translator_available:
        print("Google Translate library not available. Please install it with:")
        print("pip3 install googletrans==4.0.0-rc1")
        print("\nAlternatively, the script can create a template for manual translation.")
        create_template = input("Create a template for manual translation? (y/n): ").lower()
        
        if create_template == 'y':
            return create_translation_template(input_file, output_file.replace('.b2', '_template.b2'))
        else:
            return False
    
    return translate_fb2_file(input_file, output_file)

def create_translation_template(input_file, output_file):
    """Create a template file with placeholders for manual translation"""
    try:
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        def add_translation_markers(element):
            # Process text content
            if element.text and len(element.text.strip()) > 0:
                original_text = element.text.strip()
                if original_text:  # Only mark non-empty text
                    element.text = f"[TRANSLATE: {original_text}]"
            
            # Process child elements
            for child in element:
                add_translation_markers(child)
            
            # Process tail content
            if element.tail and len(element.tail.strip()) > 0:
                original_tail = element.tail.strip()
                if original_tail:
                    element.tail = f"[TRANSLATE: {original_tail}]"
        
        # Add translation markers to the entire document
        add_translation_markers(root)
        
        # Write the template file
        tree.write(output_file, encoding='utf-8', xml_declaration=True)
        
        print(f"Translation template created: {output_file}")
        print("Edit this file to replace [TRANSLATE: text] with Serbian translations")
        return True
    except Exception as e:
        print(f"Error creating template: {e}")
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)