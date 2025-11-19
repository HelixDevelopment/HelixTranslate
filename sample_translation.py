#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import re
from pathlib import Path

def create_sample_translation():
    """Create a sample translated version of the first few sections"""
    input_file = "Ratibor_1f.b2"
    output_file = "Ratibor_1f_sr_sample.b2"
    
    # Sample translations for demonstration
    translations = {
        "Ратибор": "Ратибор",
        "Отзвуки": "Одјеци",
        "1. А день был солнечный": "1. А дан је био сунчан",
        "Уходя во тьму, зажигайте с собою свет.": "Олазећи у мрак, палећите са собом светлост.",
        "Во тьме много тех, кто потерял свой свет.": "У мраку има много оних који су изгубили своју светлост.",
        "Освещая им тьму, помогайте зажечь потерянный свет.": "Осветљавајући им мрак, помозите да запале изгубљену светлост.",
        "Ведь пройти сквозь тьму, возможно": "Јер проћи кроз мрак је могуће",
        "лишь, пронеся через всю жизнь свет.": "само ако се носи светлост кроз цео живот.",
        "Зазвонил телефон оперативной связи.": "Зазвонио је телефон оперативне везе.",
        "— Ярцев, слушаю. Да. Нет. Да. Будет сделано.": "— Јарцев, слушам. Да. Не. Да. Биће урађено.",
        "Положив, трубку я откинулся на кресле и посмотрел в потолок.": "Оставив слушалицу, одбацио сам се у столицу и погледао у плафон.",
        "Снова эти бумажки печатать, чертова бюрократия.": "Опет те папире да штампам, проклета бирократија.",
        "Но работа есть работа, необходимо сделать побыстрее.": "Али посао је посао, потребно је да се уради брже.",
        "Зовут меня Ярцев Михаил Александрович. 29 лет. Жены нет, есть кошка.": "Зову ме Јарцев Михаило Александрович. 29 година. Немам жену, имају мачку.",
        "Родители не так давно погибли в автокатастрофе.": "Родитељи су погинули не тако давно у саобраћајној несрећи.",
        "Уже несколько лет по сути жил я, да Буся, которая поддерживала меня в непростое время, но жизнь есть жизнь и нужно стремиться идти дальше.": "Више година сам заправо живео, је Буся, која ме је подржавала у тешко време, али живот је живот и треба тежити да се иде даље.",
        "Служба в органах безопасности проходила ровно без сюрпризов, хотя наверное самый главный сюрприз был в том, что сама деятельность и работа в корне отличались от представлений, тогда еще, молодого пацана, отслужившего срочную службу в мотострелках и отучившегося на юриста по уголовно правовым дисциплинам.": "Служба у органима безбедности је текла глатко без изненађења, иако је вероватно највеће изненађење било што су сама делатност и рад били сасвим другачији од представа, тада још младића, који је одслужио обавезни војни рок у моторизованој пешадији и студирао право кривично-правне дисциплине.",
        "Ничего примечательного или интересного, обычная жизнь обычного человека.": "Ништа посебно или занимљиво, обичан живот обичне особе.",
        "Все как-то думают, что служба в ФСБ — это гонять преступников с пистолетом и гранатой по дворам, валить террористов, раскрывать преступления, быть своего рода разведчиком.": "Сви некако мисле да служба у ФСБ-у значи јурати злочинце са пиштољем и бомбом по двориштима, свргавати терористе, откривати злочине, бити врсте шпијуна.",
        "И мало кто знает, что на самом деле служба здесь, зачастую, иная. 80% ФСБ совсем не ФСБ. Да.": "И мало ко зна да је служба овде, најчешће, другачија. 80% ФСБ-а уопште није ФСБ. Да.",
        "И сейчас я просто занимался крючкотворством на вверенном направлении, а именно, координировал действия разных подразделений ведомства касательно научно-исследовательских направлений, вел переписку с Организациями вне ведомства.": "И сада сам се само бавио бирократијом на повереном подручју, а то јесте, координирао сам деловање различитих одељења ресора у вези са научно-истраживачким правцима, водио преписку са организацијама ван ресора.",
        "Ну как координировал, состоял в группе, которая за это отвечала.": "Па како координирао, био сам у групи која је за то била одговорна.",
        "Скучная монотонная работа.": "Досадан монотонан посао.",
        "Как же жарко сегодня, а кондиционер сдох.": "Како је врели данас, а клима уређај се покварио.",
        "Я открыл файлы на компьютере и стал составлять документ, что после будет отдан начальнику.": "Отворио сам датотеке на рачунару и почео састављати документ који ће након тога бити предат шефу.",
        "Чтож в научно-исследовательском управлении было неплохо.": "Па у научно-истраживачкој управи није било лоше.",
        "Получил уже звание старшего лейтенант.": "Већ сам добио чин старијег поручника.",
        "Тихая размеренная, немного скучная служба, разбавляемая разве что занятиями, да выездам на полигоны, где стреляли из всего что стреляет.": "Миран, уредан, мало досадан сервис, испуњен само вежбама и излетима на полигоне где смо пуцали из свега што пуца."
    }
    
    try:
        # Register namespaces
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        # Parse the file
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        translation_count = 0
        
        def translate_element_text(element):
            nonlocal translation_count
            
            # Translate element text
            if element.text and element.text.strip():
                text = element.text.strip()
                if text in translations:
                    element.text = translations[text]
                    translation_count += 1
            
            # Process child elements
            for child in element:
                translate_element_text(child)
            
            # Translate tail text
            if element.tail and element.tail.strip():
                text = element.tail.strip()
                if text in translations:
                    element.tail = translations[text]
                    translation_count += 1
        
        # Process only the first section to create a sample
        body = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}body')
        if body is not None:
            first_section = body.find('{http://www.gribuser.ru/xml/fictionbook/2.0}section')
            if first_section is not None:
                translate_element_text(first_section)
                
                # Also translate the title
                title = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if title is not None:
                    for p in title.findall('{http://www.gribuser.ru/xml/fictionbook/2.0}p'):
                        if p.text and p.text.strip() in translations:
                            p.text = translations[p.text.strip()]
                            translation_count += 1
                
                # Update document language
                description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
                if description is not None:
                    title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
                    if title_info is not None:
                        lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                        if lang is not None:
                            lang.text = 'sr'
                        
                        # Translate the title
                        book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                        if book_title is not None:
                            book_title.text = "Одјеци"
        
        # Write the sample file
        tree.write(output_file, encoding='utf-8', xml_declaration=True)
        
        print(f"Sample translation created: {output_file}")
        print(f"Translated {translation_count} elements")
        print("This is just a sample showing how the translation would look.")
        
        return True
        
    except Exception as e:
        print(f"Error creating sample: {e}")
        return False

def main():
    success = create_sample_translation()
    if success:
        print("\nTo continue with the full translation:")
        print("1. Use the translation list (translation_list.txt)")
        print("2. Fill in Serbian translations for all Russian text")
        print("3. Apply translations using translation_helper.py option 2")
        print("\nOr use a professional translator for the best results")
    
    return success

if __name__ == "__main__":
    main()