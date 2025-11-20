#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
from pathlib import Path
import textwrap
import html

# Try to import translation libraries
try:
    from googletrans import Translator
    translator_available = True
except ImportError:
    translator_available = False

# Serbian-specific language helpers
SERBIAN_CYRILLIC_TO_LATIN = {
    'А': 'A', 'Б': 'B', 'В': 'V', 'Г': 'G', 'Д': 'D', 'Ђ': 'Đ', 'Е': 'E', 'Ж': 'Ž', 'З': 'Z',
    'И': 'I', 'Ј': 'J', 'К': 'K', 'Л': 'L', 'Љ': 'Lj', 'М': 'M', 'Н': 'N', 'Њ': 'Nj', 'О': 'O',
    'П': 'P', 'Р': 'R', 'С': 'S', 'Т': 'T', 'Ћ': 'Ć', 'У': 'U', 'Ф': 'F', 'Х': 'H', 'Ц': 'C',
    'Ч': 'Č', 'Џ': 'Dž', 'Ш': 'Š', 'а': 'a', 'б': 'b', 'в': 'v', 'г': 'g', 'д': 'd', 'ђ': 'đ',
    'е': 'e', 'ж': 'ž', 'з': 'z', 'и': 'i', 'ј': 'j', 'к': 'k', 'л': 'l', 'љ': 'lj', 'м': 'm',
    'н': 'n', 'њ': 'nj', 'о': 'o', 'п': 'p', 'р': 'r', 'с': 's', 'т': 't', 'ћ': 'ć', 'у': 'u',
    'ф': 'f', 'х': 'h', 'ц': 'c', 'ч': 'č', 'џ': 'dž', 'ш': 'š'
}

class FB2Translator:
    def __init__(self, target_script='cyrillic'):
        self.target_script = target_script  # 'cyrillic' or 'latin'
        self.translator = Translator() if translator_available else None
        self.translation_cache = {}
        
    def translate_text(self, text, context=""):
        """Translate text with context awareness and quality checks"""
        if not text or not text.strip():
            return text
            
        if not translator_available:
            return text
            
        # Check cache first
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            return self.translation_cache[cache_key]
            
        try:
            # Prepare context for better translation
            context_text = f"Context: {context}\nText: " if context else ""
            
            # Translate to Serbian
            result = self.translator.translate(text, dest='sr')
            translated = result.text
            
            # Post-process for better quality
            translated = self.post_process_translation(translated, text)
            
            # Cache the result
            self.translation_cache[cache_key] = translated
            return translated
            
        except Exception as e:
            print(f"Translation error: {e}")
            return text
    
    def post_process_translation(self, translated, original):
        """Improve translation quality with post-processing"""
        # Fix common translation issues
        # Ensure proper formatting is maintained
        if original.endswith('\n'):
            translated = translated.rstrip() + '\n'
        
        # Fix quote marks if needed
        if '"' in original and '"' not in translated:
            translated = translated.replace('"', '"')
            
        # Fix apostrophes
        if "'" in original and "'" not in translated:
            translated = translated.replace("'", "'")
            
        return translated
    
    def convert_script(self, text):
        """Convert between Cyrillic and Latin script"""
        if self.target_script == 'latin':
            # Convert Cyrillic to Latin
            result = ""
            for char in text:
                result += SERBIAN_CYRILLIC_TO_LATIN.get(char, char)
            return result
        else:
            return text  # Keep Cyrillic
    
    def process_element(self, element, parent_context=""):
        """Process element with context-aware translation"""
        # Create new element
        new_element = ET.Element(element.tag, element.attrib)
        
        # Determine element context
        element_type = element.tag.split('}')[-1]  # Remove namespace if present
        current_context = f"{parent_context}/{element_type}"
        
        # Process text content
        if element.text:
            if len(element.text.strip()) > 0:
                translated = self.translate_text(element.text, current_context)
                new_element.text = self.convert_script(translated)
            else:
                new_element.text = element.text
        
        # Process child elements
        for child in element:
            new_child = self.process_element(child, current_context)
            new_element.append(new_child)
        
        # Process tail content
        if element.tail:
            if len(element.tail.strip()) > 0:
                translated = self.translate_text(element.tail, current_context)
                new_element.tail = self.convert_script(translated)
            else:
                new_element.tail = element.tail
        
        return new_element
    
    def translate_fb2_file(self, input_file, output_file):
        """Main translation function"""
        try:
            # Register namespaces to preserve them
            ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
            ET.register_namespace('l', "http://www.w3.org/1999/xlink")
            
            # Parse the XML file
            tree = ET.parse(input_file)
            root = tree.getroot()
            
            # Process the entire document
            new_root = self.process_element(root)
            
            # Create new tree
            new_tree = ET.ElementTree(new_root)
            
            # Write with proper formatting
            self.write_pretty_xml(new_tree, output_file)
            
            print(f"Successfully translated {input_file} to {output_file}")
            print(f"Translation used {self.target_script} script")
            print(f"Cached {len(self.translation_cache)} translations")
            return True
            
        except Exception as e:
            print(f"Error translating file: {e}")
            return False
    
    def write_pretty_xml(self, tree, output_file):
        """Write XML with proper formatting"""
        from xml.dom import minidom
        
        # Get the XML string
        rough_string = ET.tostring(tree.getroot(), encoding='unicode')
        
        # Parse with minidom for pretty printing
        reparsed = minidom.parseString(rough_string)
        
        # Write to file with proper encoding
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(reparsed.toprettyxml(indent=" ", encoding=None))

def create_quality_check_template(input_file, output_file):
    """Create a template for manual quality checking"""
    tree = ET.parse(input_file)
    root = tree.getroot()
    
    def add_review_markers(element):
        if element.text and len(element.text.strip()) > 5:  # Only mark substantial text
            element.text = f"[REVIEW: {element.text}]"
        
        for child in element:
            add_review_markers(child)
            
        if element.tail and len(element.tail.strip()) > 5:
            element.tail = f"[REVIEW: {element.tail}]"
    
    add_review_markers(root)
    
    # Register namespaces
    ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
    ET.register_namespace('l', "http://www.w3.org/1999/xlink")
    
    tree.write(output_file, encoding='utf-8', xml_declaration=True)
    print(f"Quality check template created: {output_file}")

def main():
    input_file = "Ratibor_1f.b2"
    output_file_cyr = "Ratibor_1f_sr_cyrillic.b2"
    output_file_lat = "Ratibor_1f_sr_latin.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    if not translator_available:
        print("Google Translate library not available. Please install it with:")
        print("pip3 install googletrans==4.0.0-rc1")
        
        # Create template for manual translation
        create_template = input("Create template for manual translation? (y/n): ").lower()
        if create_template == 'y':
            return create_quality_check_template(input_file, "Ratibor_1f_sr_template.b2")
        return False
    
    # Ask user which script they prefer
    script_choice = input("Choose script for Serbian translation:\n1. Cyrillic\n2. Latin\nEnter choice (1 or 2): ").strip()
    
    target_script = 'cyrillic' if script_choice == '1' else 'latin'
    output_file = output_file_cyr if target_script == 'cyrillic' else output_file_lat
    
    print(f"Starting high-quality translation to Serbian ({target_script})...")
    print("This may take several minutes depending on the text length...")
    
    translator = FB2Translator(target_script=target_script)
    success = translator.translate_fb2_file(input_file, output_file)
    
    if success:
        print("\nTranslation completed!")
        print("Recommendation: Please review the translation for:")
        print("1. Cultural nuances and idioms")
        print("2. Character names and place names")
        print("3. Technical terms specific to the context")
        print("4. Overall flow and readability")
        
        create_review = input("\nCreate a quality check template for manual review? (y/n): ").lower()
        if create_review == 'y':
            review_file = output_file.replace('.b2', '_review.b2')
            create_quality_check_template(output_file, review_file)
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)