#!/usr/bin/env python3
# Simple test to debug translation issues

# Test dictionary with basic mappings
RU_TO_SR = {
    "я": "ја", "Я": "Ја",
    "убийца": "убица", "мама": "мајка",
    "кровь": "крв", "снегу": "снегу",
    "может": "може", "сказать": "рећи",
    "другое": "друго", "гожусь": "вештам"
}

CYRILLIC_CHARS = {
    'я': 'ја', 'Я': 'Ја',
    'ё': 'јо', 'Ё': 'Јо',
    'ы': 'и', 'Ы': 'И',
    'э': 'е', 'Э': 'Е',
    'ъ': '', 'Ъ': '',
}

def translate_text(text):
    # Apply character mapping first
    for char, replacement in CYRILLIC_CHARS.items():
        text = text.replace(char, replacement)
    
    # Simple word replacement
    words = text.split()
    translated_words = []
    
    for word in words:
        # Clean punctuation for lookup
        clean_word = word.strip('.,!?;:()[]{}\"\'')
        if clean_word in RU_TO_SR:
            translated_word = word.replace(clean_word, RU_TO_SR[clean_word])
        else:
            translated_word = word
        translated_words.append(translated_word)
    
    return ' '.join(translated_words)

# Test with actual content from the file
test_text = "Ја – убица. Убијам людей по наруџби. Можно сказать, ни на шта другое ја и не гожусь. Однако у менја јести једна проблема: ја не могу причинить вред женщине. Изгледа, ето због мами. И још ја слишком легко влюблјаюсь."

print("Original:")
print(test_text)
print("\nTranslated:")
print(translate_text(test_text))