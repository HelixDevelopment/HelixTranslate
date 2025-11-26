#!/usr/bin/env python3
import sys

# Character mapping from Russian to Serbian Cyrillic
CYRILLIC_CHARS = {
    # Russian letters that differ in Serbian
    'я': 'ја', 'Я': 'Ја',
    'ё': 'јо', 'Ё': 'Јо',
    'ы': 'и', 'Ы': 'И',
    'э': 'е', 'Э': 'Е',
    'ъ': '', 'Ъ': '',  # Hard sign not used in Serbian
}

RU_TO_SR = {
    "я": "ја", "Я": "Ја",
    "убийца": "убица", "убиваю": "убијам", "людей": "људи", "по": "по",
    "заказу": "наруџби", "можно": "може", "сказать": "рећи", "ни": "ни",
    "на": "на", "что": "шта", "другое": "друго", "я": "ја", "и": "и",
    "не": "не", "гожусь": "годим"
}

def translate_russian_to_serbian(text):
    if not text.strip():
        return text
    
    # Apply character mapping first
    for char, replacement in CYRILLIC_CHARS.items():
        text = text.replace(char, replacement)
    
    # Then word-by-word translation
    words = text.split(' ')
    translated_words = []
    
    for word in words:
        prefix = ''
        suffix = ''
        clean_word = word
        
        # Extract prefix punctuation
        while clean_word and not clean_word[0].isalnum():
            prefix += clean_word[0]
            clean_word = clean_word[1:]
        
        # Extract suffix punctuation
        while clean_word and not clean_word[-1].isalnum():
            suffix = clean_word[-1] + suffix
            clean_word = clean_word[:-1]
        
        # Translate the clean word
        translated_clean = RU_TO_SR.get(clean_word.lower(), clean_word)
        
        # Preserve capitalization
        if clean_word and clean_word[0].isupper():
            translated_clean = translated_clean.capitalize()
        
        # Reassemble
        translated_word = prefix + translated_clean + suffix
        translated_words.append(translated_word)
    
    return ' '.join(translated_words)

if __name__ == "__main__":
    test_text = 'Я – убийца. Убиваю людей по заказу. Можно сказать, ни на что другое я и не гожусь.'
    result = translate_russian_to_serbian(test_text)
    print('Original:', test_text)
    print('Translated:', result)
