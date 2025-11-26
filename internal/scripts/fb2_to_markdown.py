#!/usr/bin/env python3
import sys
import re
import xml.etree.ElementTree as ET

def extract_text_from_element(element):
    """Extract text from XML element recursively"""
    if element is None:
        return ""
    
    text_parts = []
    # Add element's text content
    if element.text:
        text_parts.append(element.text)
    
    # Process children recursively
    for child in element:
        child_text = extract_text_from_element(child)
        if child_text:
            text_parts.append(child_text)
        # Add tail text after the child element
        if child.tail:
            text_parts.append(child.tail)
    
    return ''.join(text_parts)

def convert_fb2_to_markdown(input_file, output_file):
    """Convert FB2 file to Markdown format"""
    try:
        # Parse the FB2 XML file
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        # Handle namespace
        namespace = {'fb2': 'http://www.gribuser.ru/xml/fictionbook/2.0'}
        
        markdown_content = []
        
        # Extract title
        title_info = root.find('.//fb2:title-info', namespace)
        if title_info is not None:
            book_title = title_info.find('.//fb2:book-title', namespace)
            if book_title is not None:
                title_text = extract_text_from_element(book_title)
                markdown_content.append(f"# {title_text}")
                markdown_content.append("")
        
        # Extract authors
        authors = title_info.findall('.//fb2:author', namespace) if title_info else []
        if authors:
            markdown_content.append("## Authors")
            for author in authors:
                first_name = extract_text_from_element(author.find('.//fb2:first-name', namespace))
                last_name = extract_text_from_element(author.find('.//fb2:last-name', namespace))
                author_name = f"{first_name} {last_name}".strip()
                if author_name:
                    markdown_content.append(f"- {author_name}")
            markdown_content.append("")
        
        # Extract annotation
        annotation = title_info.find('.//fb2:annotation', namespace) if title_info else None
        if annotation is not None:
            markdown_content.append("## Annotation")
            for p in annotation.findall('.//fb2:p', namespace):
                para_text = extract_text_from_element(p)
                if para_text:
                    markdown_content.append(para_text)
                    markdown_content.append("")
        
        # Extract body content
        body = root.find('.//fb2:body', namespace)
        if body is not None:
            sections = body.findall('.//fb2:section', namespace)
            for section in sections:
                # Section title
                title = section.find('.//fb2:title', namespace)
                if title is not None:
                    for p in title.findall('.//fb2:p', namespace):
                        title_text = extract_text_from_element(p)
                        if title_text:
                            markdown_content.append(f"## {title_text}")
                            markdown_content.append("")
                
                # Section paragraphs
                paragraphs = section.findall('.//fb2:p', namespace)
                for p in paragraphs:
                    # Skip paragraphs that are part of titles (we already processed them)
                    if p.find('../fb2:title', namespace) is None:
                        para_text = extract_text_from_element(p)
                        if para_text:
                            markdown_content.append(para_text)
                            markdown_content.append("")
                
                # Epigraphs
                epigraphs = section.findall('.//fb2:epigraph', namespace)
                for epigraph in epigraphs:
                    epigraph_paragraphs = epigraph.findall('.//fb2:p', namespace)
                    if epigraph_paragraphs:
                        for p in epigraph_paragraphs:
                            epigraph_text = extract_text_from_element(p)
                            if epigraph_text:
                                markdown_content.append(f"> {epigraph_text}")
                        
                        # Text author
                        text_authors = epigraph.findall('.//fb2:text-author', namespace)
                        for text_author in text_authors:
                            author_text = extract_text_from_element(text_author)
                            if author_text:
                                markdown_content.append(f"> — {author_text}")
                        
                        markdown_content.append("")
                
                # Poems
                poems = section.findall('.//fb2:poem', namespace)
                for poem in poems:
                    # Poem title
                    poem_title = poem.find('.//fb2:title', namespace)
                    if poem_title is not None:
                        for p in poem_title.findall('.//fb2:p', namespace):
                            title_text = extract_text_from_element(p)
                            if title_text:
                                markdown_content.append(f"### {title_text}")
                                markdown_content.append("")
                    
                    # Stanzas
                    stanzas = poem.findall('.//fb2:stanza', namespace)
                    for stanza in stanzas:
                        verses = stanza.findall('.//fb2:v', namespace)
                        for v in verses:
                            verse_text = extract_text_from_element(v)
                            if verse_text:
                                markdown_content.append(f"    {verse_text}")
                        markdown_content.append("")
        
        # Write markdown to file
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write('\n'.join(markdown_content))
        
        print(f"FB2 converted to markdown: {len(markdown_content)} lines")
        return True
        
    except Exception as e:
        print(f"Error converting FB2: {e}")
        return False

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 fb2_to_markdown.py <input.fb2> <output.md>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    if convert_fb2_to_markdown(input_file, output_file):
        print(f"Successfully converted {input_file} to {output_file}")
    else:
        print("Conversion failed")
        sys.exit(1)