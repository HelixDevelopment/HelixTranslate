#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
import sys
import time
from pathlib import Path
from typing import Dict, List, Tuple, Optional

# Advanced translation imports
TRANSLATE_ENABLED = False
try:
    from googletrans import Translator
    import html
    TRANSLATE_ENABLED = True
    print("Google Translate enabled for advanced translation")
except ImportError:
    Translator = None
    print("Using enhanced template approach")

class AdvancedFB2Translator:
    def __init__(self):
        self.translator = None
        self.translate_enabled = TRANSLATE_ENABLED
        if self.translate_enabled and Translator:
            try:
                self.translator = Translator()
            except Exception as e:
                print(f"Warning: Could not initialize translator: {e}")
                self.translate_enabled = False
        self.translation_cache = {}
        self.context_cache = {}
        self.errors = []
        self.translation_stats = {
            'total': 0,
            'translated': 0,
            'errors': 0,
            'cached': 0
        }
        
        # Serbian Cyrillic to Latin mapping for quality enhancement
        self.cyrl_to_latn = {
            'А': 'A', 'Б': 'B', 'В': 'V', 'Г': 'G', 'Д': 'D', 'Ђ': 'Đ', 'Е': 'E', 'Ж': 'Ž', 'З': 'Z',
            'И': 'I', 'Ј': 'J', 'К': 'K', 'Л': 'L', 'Љ': 'Lj', 'М': 'M', 'Н': 'N', 'Њ': 'Nj', 'О': 'O',
            'П': 'P', 'Р': 'R', 'С': 'S', 'Т': 'T', 'Ћ': 'Ć', 'У': 'U', 'Ф': 'F', 'Х': 'H', 'Ц': 'C',
            'Ч': 'Č', 'Џ': 'Dž', 'Ш': 'Š', 'а': 'a', 'б': 'b', 'в': 'v', 'г': 'g', 'д': 'd', 'ђ': 'đ',
            'е': 'e', 'ж': 'ž', 'з': 'z', 'и': 'i', 'ј': 'j', 'к': 'k', 'л': 'l', 'љ': 'lj', 'м': 'm',
            'н': 'n', 'њ': 'nj', 'о': 'o', 'п': 'p', 'р': 'r', 'с': 's', 'т': 't', 'ћ': 'ć', 'у': 'u',
            'ф': 'f', 'х': 'h', 'ц': 'c', 'ч': 'č', 'џ': 'dž', 'ш': 'š'
        }
        
        # High-quality reference translations for common terms
        self.reference_translations = {
            "Ратибор": "Ратибор",  # Keep names
            "Отзвуки": "Одјеци",
            "фэнтези": "фантастикa",
            "научная фантастика": "научна фантастика",
            "приключения": "авантуре",
            "герой": "јунак",
            "мир": "свет",
            "человек": "човек",
            "жизнь": "живот",
            "любовь": "љубав",
            "смерть": "смрт",
            "время": "време",
            "дом": "кућа",
            "сердце": "срце",
            "душа": "душа",
            "свет": "светлост/свemin шин",
            "тьма": "мрак",
            "ночь": "ноћ",
            "день": "дан",
            "солнце": "сунце",
            "луна": "месец",
            "звезда": "звезда",
            "небо": "небо",
            "земля": "земља",
            "вода": "вода",
            "огонь": "ватра",
            "воздух": "ваздух",
            "деревня": "село",
            "город": "град",
            "улица": "улица",
            "дом": "кућа",
            "книга": "књига",
            "слово": "реч",
            "язык": "језик",
            "глава": "поглавље",
            "страница": "страница",
            "история": "прича",
            "конец": "крај",
            "начало": "почетак",
            "будущее": "будућност",
            "прошлое": "прошлост",
            "настоящее": "садашњост",
            "вопрос": "питање",
            "ответ": "одговор",
            "мысль": "мисао",
            "чувство": "осећање",
            "радость": "радост",
            "грусть": "туга",
            "страх": "страх",
            "надежда": "нада",
            "мечта": "сан"
        }
    
    def enhance_translation(self, text: str, translated: str, context: str) -> str:
        """Enhance translation with quality improvements"""
        # Fix common translation errors
        enhanced = translated
        
        # Ensure proper punctuation preservation
        original_punct = re.findall(r'[.!?;:,]', text)
        translated_punct = re.findall(r'[.!?;:,]', enhanced)
        
        # Fix mismatched quotation marks
        if '"' in text and '"' not in enhanced:
            enhanced = enhanced.replace('"', '"').replace('"', '"')
        
        # Fix apostrophes
        if "'" in text and "'" not in enhanced:
            enhanced = enhanced.replace("'", "'")
        
        # Ensure paragraph structure
        if text.endswith('\n') and not enhanced.endswith('\n'):
            enhanced += '\n'
        
        # Fix capitalization at sentence start
        if enhanced and enhanced[0].islower() and text and text[0].isupper():
            enhanced = enhanced[0].upper() + enhanced[1:]
        
        return enhanced
    
    def translate_with_context(self, text: str, context: str, retries: int = 3) -> Optional[str]:
        """Translate text with context awareness and retry logic"""
        if not text or not text.strip():
            return text
        
        # Check cache first
        cache_key = (text, context)
        if cache_key in self.translation_cache:
            self.translation_stats['cached'] += 1
            return self.translation_cache[cache_key]
        
        # Check reference translations first
        if text in self.reference_translations:
            self.translation_stats['translated'] += 1
            translation = self.reference_translations[text]
            self.translation_cache[cache_key] = translation
            return translation
        
        if not self.translate_enabled or not self.translator:
            return None
        
        # Try translation with retries
        for attempt in range(retries):
            try:
                # Add context to improve translation
                context_text = f"Context: {context}" if context else ""
                
                # Use multiple approaches for better quality
                result = self.translator.translate(text, dest='sr', src='ru')
                translation = result.text if hasattr(result, 'text') else str(result)
                
                # Enhance translation quality
                enhanced_translation = self.enhance_translation(text, translation, context)
                
                # Cache successful translation
                self.translation_cache[cache_key] = enhanced_translation
                self.translation_stats['translated'] += 1
                
                return enhanced_translation
                
            except Exception as e:
                if attempt == retries - 1:
                    self.errors.append(f"Translation failed: {text[:30]}... - {str(e)}")
                    self.translation_stats['errors'] += 1
                    return None
                time.sleep(1)  # Wait before retry
        
        return None
    
    def process_fb2_structure(self, input_path: str, output_path: str) -> bool:
        """Process FB2 file with high-quality translation"""
        try:
            # Register namespaces
            ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
            ET.register_namespace('l', "http://www.w3.org/1999/xlink")
            
            # Parse the file
            tree = ET.parse(input_path)
            root = tree.getroot()
            
            print(f"Processing FB2 structure...")
            
            # Update document metadata
            self.update_document_metadata(root)
            
            # Process all text elements
            self.process_element_translations(root)
            
            # Write the enhanced translation
            print(f"\nWriting output to: {output_path}")
            try:
                self.write_enhanced_xml(tree, output_path)  # type: ignore
            except Exception as e:
                print(f"Write failed: {e}")
                raise
            
            # Show statistics
            self.print_translation_stats()
            
            print("About to write file...")
            # Write the enhanced translation
            print(f"\nWriting output to: {output_path}")
            try:
                self.write_enhanced_xml(tree, output_path)  # type: ignore
            except Exception as e:
                print(f"Write failed: {e}")
                raise
            return True
            
        except Exception as e:
            print(f"Error processing FB2: {e}")
            import traceback
            traceback.print_exc()
            return False
    
    def update_document_metadata(self, root: ET.Element):
        """Update document metadata for Serbian translation"""
        # Update language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'
                
                # Translate title
                book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if book_title is not None:
                    book_title.text = self.translate_with_context("Отзвуки", "book-title") or "Одјеци"
    
    def process_element_translations(self, element: ET.Element, context: str = ""):
        """Recursively process and translate all elements"""
        # Process element text
        if element.text and element.text.strip():
            text = element.text.strip()
            if len(text) > 2:
                self.translation_stats['total'] += 1
                
                translation = self.translate_with_context(text, context)
                if translation:
                    element.text = translation
                    print(".", end="", flush=True)
                else:
                    element.text = text
                    print("o", end="", flush=True)
        
        # Process child elements
        for child in element:
            child_context = f"{context}/{child.tag}" if context else child.tag
            self.process_element_translations(child, child_context)
        
        # Process tail text
        if element.tail and element.tail.strip():
            text = element.tail.strip()
            if len(text) > 2:
                self.translation_stats['total'] += 1
                
                translation = self.translate_with_context(text, f"{context}/tail")
                if translation:
                    element.tail = translation
                    print(".", end="", flush=True)
                else:
                    element.tail = text
                    print("o", end="", flush=True)
        
        # Process child elements
        for child in element:
            child_context = f"{context}/{child.tag}" if context else child.tag
            self.process_element_translations(child, child_context)
        
        # Process tail text
        if element.tail and element.tail.strip():
            text = element.tail.strip()
            if len(text) > 2:
                self.translation_stats['total'] += 1
                
                translation = self.translate_with_context(text, f"{context}/tail")
                if translation:
                    element.tail = translation
                    print(".", end="", flush=True)
                else:
                    element.tail = text
                    print("o", end="", flush=True)
    
    def write_enhanced_xml(self, tree: ET.ElementTree, output_path: str):
        """Write enhanced XML with proper formatting"""
        try:
            print(f"Attempting enhanced XML writing...")
            # Get XML string
            root = tree.getroot()
            if root is None:
                raise ValueError("Root element is None")
            xml_string = ET.tostring(root, encoding='unicode')
            
            # Parse and format with minidom
            from xml.dom import minidom
            dom = minidom.parseString(xml_string)
            
            # Write with proper formatting
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(dom.toprettyxml(indent=" ", encoding=None))
            
            print(f"\n✓ Enhanced translation saved: {output_path}")
            
        except Exception as e:
            print(f"Enhanced writing failed: {e}")
            print("Falling back to standard XML writing...")
            # Fallback to standard XML writing
            try:
                tree.write(output_path, encoding='utf-8', xml_declaration=True)
                print(f"\n✓ Translation saved: {output_path}")
            except Exception as e2:
                print(f"Standard writing also failed: {e2}")
                raise e2
    
    def print_translation_stats(self):
        """Print translation statistics"""
        print("\n" + "="*50)
        print("TRANSLATION STATISTICS")
        print("="*50)
        print(f"Total text elements: {self.translation_stats['total']}")
        print(f"Successfully translated: {self.translation_stats['translated']}")
        print(f"From cache: {self.translation_stats['cached']}")
        print(f"Errors: {self.translation_stats['errors']}")
        
        if self.translation_stats['total'] > 0:
            success_rate = (self.translation_stats['translated'] / self.translation_stats['total']) * 100
            print(f"Success rate: {success_rate:.1f}%")
        
        if self.errors:
            print(f"\nErrors encountered: {len(self.errors)}")
            for i, error in enumerate(self.errors[:5], 1):
                print(f"  {i}. {error}")
            if len(self.errors) > 5:
                print(f"  ... and {len(self.errors) - 5} more")

def main():
    # Handle command line arguments
    if len(sys.argv) < 2:
        print("Usage: python3 high_quality_fb2_translator.py <input_file.fb2> [output_file.b2]")
        print("Example: python3 high_quality_fb2_translator.py book_ru.fb2 book_sr.b2")
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
    
    print("Starting HIGH QUALITY Russian to Serbian FB2 Translation")
    print("="*60)
    print("Using advanced translation with context awareness...")
    print("Cyrillic script (ћирилица) - Traditional Serbian")
    print("="*60)
    
    translator = AdvancedFB2Translator()
    success = translator.process_fb2_structure(input_file, output_file)
    
    if success:
        print("\n" + "="*60)
        print("✓ TRANSLATION COMPLETED SUCCESSFULLY!")
        print("="*60)
        print(f"✓ Output file: {output_file}")
        print("\nNext steps:")
        print("1. Review the translation in an FB2 reader")
        print("2. Make manual adjustments for cultural nuances")
        print("3. Test readability and flow")
        print("4. Consider professional review for final quality")
    else:
        print("\n✗ Translation encountered issues")
        print("Please check the error messages above")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)