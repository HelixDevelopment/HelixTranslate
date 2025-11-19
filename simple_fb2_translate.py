#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import sys
from pathlib import Path

def simple_fb2_translate(input_file, output_file):
    """Simple FB2 translation with basic text replacement"""
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse file
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        # Basic Russian to Serbian dictionary for common words
        basic_dict = {
            "Ратибор": "Ратибор",
            "Отзвуки": "Одјеци",
            "фэнтези": "фантастика",
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
        
        translation_count = 0
        
        def replace_text(text):
            nonlocal translation_count
            if not text or not text.strip():
                return text
            
            original = text
            for ru_word, sr_word in basic_dict.items():
                if ru_word in text:
                    text = text.replace(ru_word, sr_word)
                    translation_count += 1
            
            return text
        
        def process_element(element):
            # Process element text
            if element.text:
                element.text = replace_text(element.text)
            
            # Process children
            for child in element:
                process_element(child)
            
            # Process tail text
            if element.tail:
                element.tail = replace_text(element.tail)
        
        print("Applying basic translations...")
        process_element(root)
        
        # Update document language
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None:
                    lang.text = 'sr'
                
                # Translate title
                book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if book_title is not None and book_title.text:
                    book_title.text = replace_text(book_title.text)
        
        # Write translated file
        tree.write(output_file, encoding='utf-8', xml_declaration=True)
        
        print(f"Basic translation completed!")
        print(f"Basic replacements made: {translation_count}")
        print(f"Output file: {output_file}")
        
        return True
        
    except Exception as e:
        print(f"Error in basic translation: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 simple_fb2_translate.py <input_file.fb2> [output_file.b2]")
        print("Example: python3 simple_fb2_translate.py book_ru.fb2 book_sr.b2")
        return False
    
    input_file = sys.argv[1]
    
    # Generate output filename if not provided
    if len(sys.argv) >= 3:
        output_file = sys.argv[2]
    else:
        input_path = Path(input_file)
        stem = input_path.stem
        output_file = f"{stem}_sr_basic.b2"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    print("Starting basic FB2 translation...")
    success = simple_fb2_translate(input_file, output_file)
    
    if success:
        print("\nBasic translation completed!")
        print("Note: This is a basic translation using a dictionary.")
        print("For higher quality, use professional translation services.")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)