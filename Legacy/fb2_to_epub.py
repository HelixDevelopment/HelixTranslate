#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import sys
from pathlib import Path
from ebooklib import epub
import os

def fb2_to_epub(input_file, output_file):
    """Convert FB2 file to EPUB format"""
    try:
        # Parse FB2 file
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        # Create EPUB book
        book = epub.EpubBook()
        
        # Set metadata
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                # Title
                book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if book_title is not None and book_title.text:
                    book.set_title(book_title.text)
                else:
                    book.set_title("Translated Book")
                
                # Author
                author = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}author')
                if author is not None:
                    first_name = author.find('{http://www.gribuser.ru/xml/fictionbook/2.0}first-name')
                    last_name = author.find('{http://www.gribuser.ru/xml/fictionbook/2.0}last-name')
                    author_name = ""
                    if first_name is not None and first_name.text:
                        author_name += first_name.text + " "
                    if last_name is not None and last_name.text:
                        author_name += last_name.text
                    if author_name.strip():
                        book.add_author(author_name.strip())
                
                # Language
                lang = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}lang')
                if lang is not None and lang.text:
                    book.set_language(lang.text)
                else:
                    book.set_language('sr')
                
                # Annotation
                annotation = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}annotation')
                if annotation is not None:
                    annotation_text = ""
                    for p in annotation.findall('{http://www.gribuser.ru/xml/fictionbook/2.0}p'):
                        if p.text:
                            annotation_text += p.text + "\n"
                    if annotation_text.strip():
                        book.add_metadata('DC', 'description', annotation_text.strip())
        
        # Default metadata
        if not book.get_metadata('DC', 'title'):
            book.set_title("Translated Book")
        if not book.get_metadata('DC', 'language'):
            book.set_language('sr')
        
        # Process body content
        body = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}body')
        if body is None:
            print("No body content found in FB2 file")
            return False
        
        chapters = []
        spine_items = []
        
        def element_to_html(element):
            """Convert FB2 element to HTML"""
            tag = element.tag.replace('{http://www.gribuser.ru/xml/fictionbook/2.0}', '')
            
            if tag == 'section':
                html_content = ""
                for child in element:
                    html_content += element_to_html(child)
                return html_content
            elif tag == 'title':
                html_content = "<h2>"
                for child in element:
                    if child.text:
                        html_content += child.text
                html_content += "</h2>\n"
                return html_content
            elif tag == 'p':
                html_content = "<p>"
                if element.text:
                    html_content += element.text
                html_content += "</p>\n"
                return html_content
            elif tag == 'empty-line':
                return "<br/>\n"
            elif tag == 'emphasis':
                html_content = "<em>"
                if element.text:
                    html_content += element.text
                html_content += "</em>"
                return html_content
            elif tag == 'strong':
                html_content = "<strong>"
                if element.text:
                    html_content += element.text
                html_content += "</strong>"
                return html_content
            else:
                # Default handling
                html_content = ""
                if element.text:
                    html_content += element.text
                for child in element:
                    html_content += element_to_html(child)
                return html_content
        
        # Process sections as chapters
        section_num = 1
        for section in body.findall('{http://www.gribuser.ru/xml/fictionbook/2.0}section'):
            chapter_content = f"<h1>Chapter {section_num}</h1>\n"
            
            # Process section title
            title = section.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title')
            if title is not None:
                title_html = element_to_html(title)
                chapter_content = title_html + "\n"
            
            # Process section content
            for element in section:
                if element.tag != '{http://www.gribuser.ru/xml/fictionbook/2.0}title':
                    chapter_content += element_to_html(element)
            
            # Create chapter
            chapter_file = f'chapter_{section_num}.xhtml'
            chapter = epub.EpubHtml(
                title=f'Chapter {section_num}',
                file_name=chapter_file,
                content=chapter_content
            )
            book.add_item(chapter)
            chapters.append(chapter)
            spine_items.append(chapter)
            section_num += 1
        
        # Add default NCX and Nav files
        book.add_item(epub.EpubNcx())
        book.add_item(epub.EpubNav())
        
        # Define CSS style
        style = '''
        body { font-family: serif; line-height: 1.6; margin: 1em; }
        h1 { color: #333; border-bottom: 1px solid #ccc; }
        h2 { color: #666; }
        p { text-align: justify; }
        '''
        nav_css = epub.EpubItem(
            uid="nav_css",
            file_name="style/nav.css",
            media_type="text/css",
            content=style
        )
        book.add_item(nav_css)
        
        # Add spine
        book.spine = ['nav'] + spine_items
        
        # Write EPUB file
        epub.write_epub(output_file, book, {})
        
        print(f"EPUB conversion completed!")
        print(f"Output file: {output_file}")
        print(f"Chapters: {len(chapters)}")
        
        return True
        
    except Exception as e:
        print(f"Error converting FB2 to EPUB: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 fb2_to_epub.py <input_file.fb2> [output_file.epub]")
        print("Example: python3 fb2_to_epub.py book_sr.b2 book_sr.epub")
        return False
    
    input_file = sys.argv[1]
    
    # Generate output filename if not provided
    if len(sys.argv) >= 3:
        output_file = sys.argv[2]
    else:
        input_path = Path(input_file)
        stem = input_path.stem
        output_file = f"{stem}.epub"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    print("Converting FB2 to EPUB...")
    success = fb2_to_epub(input_file, output_file)
    
    if success:
        print("\nEPUB conversion completed successfully!")
        print(f"You can now read the book in any EPUB reader.")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)