#!/usr/bin/env python3
import sys
import re
import os
from pathlib import Path

def markdown_to_xhtml(markdown_text, title="Translated Book"):
    """Convert markdown to valid XHTML with proper Serbian content"""
    
    # Split content by lines
    lines = markdown_text.split('\n')
    xhtml_lines = []
    xhtml_lines.append('<?xml version="1.0" encoding="UTF-8"?>')
    xhtml_lines.append('<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">')
    xhtml_lines.append('<html xmlns="http://www.w3.org/1999/xhtml">')
    xhtml_lines.append('<head>')
    xhtml_lines.append(f'<title>{title}</title>')
    xhtml_lines.append('</head>')
    xhtml_lines.append('<body>')
    
    in_paragraph = False
    
    for line in lines:
        line = line.strip()
        
        if not line:
            # Empty line - end current paragraph if we're in one
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            continue
        
        # Headers
        if line.startswith('# '):
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            header_text = line[2:].strip()
            xhtml_lines.append(f'<h1>{header_text}</h1>')
        elif line.startswith('## '):
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            header_text = line[3:].strip()
            xhtml_lines.append(f'<h2>{header_text}</h2>')
        elif line.startswith('### '):
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            header_text = line[4:].strip()
            xhtml_lines.append(f'<h3>{header_text}</h3>')
        elif line.startswith('- '):
            # List item
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            item_text = line[2:].strip()
            xhtml_lines.append(f'<li>{item_text}</li>')
        elif line.startswith('> '):
            # Blockquote
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            quote_text = line[2:].strip()
            xhtml_lines.append(f'<blockquote>{quote_text}</blockquote>')
        elif line.startswith('    '):
            # Code block
            if in_paragraph:
                xhtml_lines.append('</p>')
                in_paragraph = False
            code_text = line[4:].strip()
            xhtml_lines.append(f'<code>{code_text}</code><br/>')
        else:
            # Regular paragraph
            if not in_paragraph:
                xhtml_lines.append('<p>')
                in_paragraph = True
            # Convert any line breaks within paragraphs
            xhtml_lines.append(line + ' ')
    
    # Close any open paragraph
    if in_paragraph:
        xhtml_lines.append('</p>')
    
    xhtml_lines.append('</body>')
    xhtml_lines.append('</html>')
    
    return '\n'.join(xhtml_lines)

def create_epub(input_markdown, output_epub, title="Translated Book"):
    """Create a valid EPUB from markdown"""
    import zipfile
    import tempfile
    
    # Create temporary directory for EPUB contents
    with tempfile.TemporaryDirectory() as temp_dir:
        os.makedirs(os.path.join(temp_dir, 'META-INF'), exist_ok=True)
        os.makedirs(os.path.join(temp_dir, 'OEBPS'), exist_ok=True)
        
        # Create mimetype
        with open(os.path.join(temp_dir, 'mimetype'), 'w') as f:
            f.write('application/epub+zip')
        
        # Create container.xml
        container_xml = '''<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>'''
        with open(os.path.join(temp_dir, 'META-INF', 'container.xml'), 'w') as f:
            f.write(container_xml)
        
        # Convert markdown to XHTML
        xhtml_content = markdown_to_xhtml(input_markdown, title)
        with open(os.path.join(temp_dir, 'OEBPS', 'chapter1.xhtml'), 'w', encoding='utf-8') as f:
            f.write(xhtml_content)
        
        # Create content.opf
        content_opf = f'''<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata>
    <dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">{title}</dc:title>
    <dc:language xmlns:dc="http://purl.org/dc/elements/1.1/">sr</dc:language>
    <dc:creator xmlns:dc="http://purl.org/dc/elements/1.1/">Translated</dc:creator>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>'''
        with open(os.path.join(temp_dir, 'OEBPS', 'content.opf'), 'w', encoding='utf-8') as f:
            f.write(content_opf)
        
        # Create EPUB zip file
        with zipfile.ZipFile(output_epub, 'w', zipfile.ZIP_DEFLATED) as epub:
            # Add mimetype first (uncompressed)
            epub.write(os.path.join(temp_dir, 'mimetype'), 'mimetype', compress_type=zipfile.ZIP_STORED)
            
            # Add other files
            epub.write(os.path.join(temp_dir, 'META-INF', 'container.xml'), 'META-INF/container.xml')
            epub.write(os.path.join(temp_dir, 'OEBPS', 'content.opf'), 'OEBPS/content.opf')
            epub.write(os.path.join(temp_dir, 'OEBPS', 'chapter1.xhtml'), 'OEBPS/chapter1.xhtml')
    
    return True

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 epub_generator.py <input.md> <output.epub>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        # Read markdown content
        with open(input_file, 'r', encoding='utf-8') as f:
            markdown_content = f.read()
        
        # Extract title from first header if available
        title = "Translated Book"
        for line in markdown_content.split('\n'):
            if line.strip().startswith('# '):
                title = line.strip()[2:]
                break
        
        # Create EPUB
        if create_epub(markdown_content, output_file, title):
            print(f"EPUB created successfully: {output_file}")
        else:
            print("Failed to create EPUB")
            sys.exit(1)
            
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()