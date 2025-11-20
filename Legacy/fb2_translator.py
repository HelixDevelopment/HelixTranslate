#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
from pathlib import Path

def create_translation_template(input_path, output_path):
    """Create a template with original Russian text and placeholders for Serbian translation"""
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the file
        tree = ET.parse(input_path)
        root = tree.getroot()
        
        # Create a copy for the template
        template_tree = ET.ElementTree(root)
        
        def mark_text_for_translation(element):
            """Mark text elements for manual translation"""
            # Process element text
            if element.text and element.text.strip():
                original = element.text.strip()
                if len(original) > 3:  # Only mark substantial text
                    element.text = f"[RU: {original}]\n[SR: ]"
                else:
                    element.text = original
            
            # Process child elements
            for child in element:
                mark_text_for_translation(child)
            
            # Process tail text
            if element.tail and element.tail.strip():
                original = element.tail.strip()
                if len(original) > 3:
                    element.tail = f"[RU: {original}]\n[SR: ]"
                else:
                    element.tail = original
        
        # Mark all text elements
        mark_text_for_translation(root)
        
        # Update document language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'  # Set to Serbian
        
        # Write the template file
        template_tree.write(output_path, encoding='utf-8', xml_declaration=True)
        
        print(f"Translation template created: {output_path}")
        print("Format: [RU: original Russian text]\\n[SR: Serbian translation]")
        print("Replace the [SR: ] sections with Serbian translations")
        return True
        
    except Exception as e:
        print(f"Error creating template: {e}")
        return False

def create_bilingual_template(input_path, output_path_ru, output_path_sr):
    """Create two files - one with Russian, one empty for Serbian translation"""
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the Russian file
        tree = ET.parse(input_path)
        root = ET.Element(tree.getroot().tag, tree.getroot().attrib)
        
        # Create empty Serbian version
        sr_tree = ET.ElementTree(root)
        
        def create_empty_structure(element):
            """Create empty structure matching the original"""
            new_element = ET.Element(element.tag, element.attrib)
            
            # Keep structural elements empty
            if element.tag.endswith('title') or element.tag.endswith('subtitle'):
                # For titles, preserve with placeholder
                if element.text and element.text.strip():
                    new_element.text = " [Превести на српски] "
            else:
                new_element.text = ""
            
            # Process children
            for child in element:
                new_child = create_empty_structure(child)
                new_element.append(new_child)
            
            new_element.tail = ""
            return new_element
        
        # Create the empty Serbian version
        sr_root = create_empty_structure(tree.getroot())
        sr_tree = ET.ElementTree(sr_root)
        
        # Write both files
        # Russian version (original)
        ET.ElementTree(tree.getroot()).write(output_path_ru, encoding='utf-8', xml_declaration=True)
        
        # Serbian version (empty template)
        sr_tree.write(output_path_sr, encoding='utf-8', xml_declaration=True)
        
        print(f"Created Russian reference: {output_path_ru}")
        print(f"Created Serbian template: {output_path_sr}")
        return True
        
    except Exception as e:
        print(f"Error creating bilingual template: {e}")
        return False

def try_automatic_translation(input_path, output_path):
    """Try automatic translation with error handling"""
    try:
        from googletrans import Translator
        translator = Translator()
        print("Google Translate available, attempting automatic translation...")
    except ImportError:
        print("Google Translate not available, creating manual template...")
        return create_translation_template(input_path, output_path)
    
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the file
        tree = ET.parse(input_path)
        root = tree.getroot()
        
        translation_count = 0
        error_count = 0
        
        def translate_with_retry(text, max_retries=3):
            """Translate text with retry logic"""
            for attempt in range(max_retries):
                try:
                    result = translator.translate(text, dest='sr')
                    return result.text
                except Exception as e:
                    if attempt == max_retries - 1:
                        print(f"Translation failed after {max_retries} attempts: {e}")
                        return text
                    continue
            return text
        
        def process_element(element):
            nonlocal translation_count, error_count
            
            # Process element text
            if element.text and element.text.strip():
                text = element.text.strip()
                if len(text) > 3:
                    try:
                        translated = translate_with_retry(text)
                        element.text = translated
                        translation_count += 1
                        print(f".", end="", flush=True)
                    except Exception as e:
                        element.text = f"[TRANSLATE: {text}]"
                        error_count += 1
                        print(f"E", end="", flush=True)
            
            # Process children
            for child in element:
                process_element(child)
            
            # Process tail text
            if element.tail and element.tail.strip():
                text = element.tail.strip()
                if len(text) > 3:
                    try:
                        translated = translate_with_retry(text)
                        element.tail = translated
                        translation_count += 1
                        print(f".", end="", flush=True)
                    except Exception as e:
                        element.tail = f"[TRANSLATE: {text}]"
                        error_count += 1
                        print(f"E", end="", flush=True)
        
        # Process all elements
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
        tree.write(output_path, encoding='utf-8', xml_declaration=True)
        
        print(f"\nTranslation completed!")
        print(f"Successfully translated: {translation_count} elements")
        print(f"Failed translations: {error_count} elements")
        
        return True
        
    except Exception as e:
        print(f"\nAutomatic translation failed: {e}")
        print("Falling back to manual template creation...")
        return create_translation_template(input_path, output_path.replace('.b2', '_template.b2'))

def main():
    input_file = "Ratibor_1f.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    # Provide options for the user
    print("FB2 Translation to Serbian")
    print("1. Try automatic translation (may have errors)")
    print("2. Create manual translation template")
    print("3. Create bilingual template (Russian + empty Serbian)")
    
    choice = input("Select option (1-3): ").strip()
    
    if choice == "1":
        output_file = "Ratibor_1f_sr.b2"
        success = try_automatic_translation(input_file, output_file)
    elif choice == "2":
        output_file = "Ratibor_1f_sr_template.b2"
        success = create_translation_template(input_file, output_file)
    elif choice == "3":
        ru_file = "Ratibor_1f_ru.b2"
        sr_file = "Ratibor_1f_sr_empty.b2"
        success = create_bilingual_template(input_path, ru_file, sr_file)
    else:
        print("Invalid choice")
        return False
    
    if success:
        print("\nOperation completed successfully!")
        print("Next steps:")
        print("1. If automatic translation: Review and correct errors")
        print("2. If manual template: Fill in Serbian translations")
        print("3. Test the final file in an FB2 reader")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)