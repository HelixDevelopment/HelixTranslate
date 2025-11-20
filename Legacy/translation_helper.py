#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
from pathlib import Path

class TranslationHelper:
    def __init__(self, template_file):
        self.template_file = template_file
        self.translated_file = template_file.replace('_template.b2', '_translated.b2')
        self.tree = ET.parse(template_file)
        self.root = self.tree.getroot()
        
    def extract_translations(self):
        """Extract Russian text and placeholder Serbian translations"""
        translations = []
        
        def process_element(element):
            if element.text and element.text.strip():
                text = element.text.strip()
                if '[RU:' in text and '[SR:' in text:
                    # Extract Russian and Serbian parts
                    ru_match = re.search(r'\[RU: (.*?)\]', text)
                    sr_match = re.search(r'\[SR: (.*?)\]', text)
                    
                    if ru_match and sr_match:
                        ru_text = ru_match.group(1)
                        sr_text = sr_match.group(1) if sr_match.group(1) else None
                        translations.append((ru_text, sr_text, element))
            
            for child in element:
                process_element(child)
            
            if element.tail and element.tail.strip():
                tail = element.tail.strip()
                if '[RU:' in tail and '[SR:' in tail:
                    ru_match = re.search(r'\[RU: (.*?)\]', tail)
                    sr_match = re.search(r'\[SR: (.*?)\]', tail)
                    
                    if ru_match and sr_match:
                        ru_text = ru_match.group(1)
                        sr_text = sr_match.group(1) if sr_match.group(1) else None
                        translations.append((ru_text, sr_text, element, True))  # True indicates tail
        
        process_element(self.root)
        return translations
    
    def create_translation_list(self, output_file):
        """Create a simple list of translations for easier editing"""
        translations = self.extract_translations()
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write("FB2 Translation Helper - Serbian\n")
            f.write("=" * 50 + "\n\n")
            f.write("Format: Russian text -> Serbian translation\n")
            f.write("Fill in the Serbian translations and save the file.\n\n")
            
            for i, (ru_text, sr_text, element, *is_tail) in enumerate(translations, 1):
                location = "tail" if is_tail and is_tail[0] else "element"
                f.write(f"# {i}. [{location}]\n")
                f.write(f"RU: {ru_text}\n")
                f.write(f"SR: {sr_text or ''}\n")
                f.write("-" * 40 + "\n")
        
        print(f"Translation list created: {output_file}")
        return output_file
    
    def apply_translations(self, translation_file):
        """Apply translations from the list back to the XML"""
        # Parse the translation file
        translations = {}
        current_ru = None
        
        with open(translation_file, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if line.startswith('RU:'):
                    current_ru = line[3:].strip()
                elif line.startswith('SR:') and current_ru:
                    sr_text = line[3:].strip()
                    translations[current_ru] = sr_text
                    current_ru = None  # Reset after finding SR
        
        # Apply translations to the XML
        def apply_to_element(element):
            if element.text and element.text.strip():
                text = element.text.strip()
                if '[RU:' in text and '[SR:' in text:
                    ru_match = re.search(r'\[RU: (.*?)\]', text)
                    if ru_match:
                        ru_text = ru_match.group(1)
                        if ru_text in translations:
                            # Replace with the Serbian translation
                            sr_text = translations[ru_text]
                            element.text = sr_text
                        else:
                            # Keep original if no translation found
                            element.text = ru_text
            
            for child in element:
                apply_to_element(child)
            
            if element.tail and element.tail.strip():
                tail = element.tail.strip()
                if '[RU:' in tail and '[SR:' in tail:
                    ru_match = re.search(r'\[RU: (.*?)\]', tail)
                    if ru_match:
                        ru_text = ru_match.group(1)
                        if ru_text in translations:
                            sr_text = translations[ru_text]
                            element.tail = sr_text
                        else:
                            element.tail = ru_text
        
        apply_to_element(self.root)
        
        # Write the translated file
        self.tree.write(self.translated_file, encoding='utf-8', xml_declaration=True)
        print(f"Translated file created: {self.translated_file}")
        return True
    
    def show_translation_stats(self):
        """Show statistics about the translation progress"""
        translations = self.extract_translations()
        total = len(translations)
        completed = sum(1 for _, sr, *_ in translations if sr and sr.strip())
        pending = total - completed
        
        print(f"\nTranslation Statistics:")
        print(f"Total text elements: {total}")
        print(f"Completed translations: {completed}")
        print(f"Pending translations: {pending}")
        if total > 0:
            print(f"Progress: {completed/total*100:.1f}%")
        else:
            print("Progress: 0.0%")
        
        return total, completed, pending

def main():
    if len(sys.argv) >= 2:
        template_file = sys.argv[1]
    else:
        template_file = "Ratibor_1f_sr_template.b2"
    
    if not Path(template_file).exists():
        print(f"Template file {template_file} not found")
        return False
    
    helper = TranslationHelper(template_file)
    
    print("FB2 Serbian Translation Helper")
    print("=" * 40)
    
    # Show current statistics
    helper.show_translation_stats()
    
    print("\nOptions:")
    print("1. Create translation list for editing")
    print("2. Apply translations from edited list")
    print("3. Show translation statistics")
    
    choice = input("Select option (1-3): ").strip()
    
    if choice == "1":
        output_file = "translation_list.txt"
        helper.create_translation_list(output_file)
        print(f"\nNext steps:")
        print(f"1. Edit {output_file} with Serbian translations")
        print(f"2. Run this script again with option 2 to apply translations")
    elif choice == "2":
        translation_file = "translation_list.txt"
        if Path(translation_file).exists():
            helper.apply_translations(translation_file)
            print("\nTranslation applied successfully!")
            helper.show_translation_stats()
        else:
            print(f"Translation file {translation_file} not found")
            print("Please create it first with option 1")
    elif choice == "3":
        helper.show_translation_stats()
    else:
        print("Invalid choice")
        return False
    
    return True

if __name__ == "__main__":
    main()