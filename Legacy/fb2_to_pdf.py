#!/usr/bin/env python3
import xml.etree.ElementTree as ET
import sys
from pathlib import Path
from weasyprint import HTML, CSS

def fb2_to_pdf(input_file, output_file):
    """Convert FB2 file to PDF format"""
    try:
        # Parse FB2 file
        ET.register_namespace('', "http://www.gribuser.ru/xml/fictionbook/2.0")
        ET.register_namespace('l', "http://www.w3.org/1999/xlink")
        
        tree = ET.parse(input_file)
        root = tree.getroot()
        
        # Extract metadata
        title = "Translated Book"
        author = "Unknown Author"
        
        description = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}description')
        if description is not None:
            title_info = description.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title-info')
            if title_info is not None:
                # Title
                book_title = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}book-title')
                if book_title is not None and book_title.text:
                    title = book_title.text
                
                # Author
                author_elem = title_info.find('{http://www.gribuser.ru/xml/fictionbook/2.0}author')
                if author_elem is not None:
                    first_name = author_elem.find('{http://www.gribuser.ru/xml/fictionbook/2.0}first-name')
                    last_name = author_elem.find('{http://www.gribuser.ru/xml/fictionbook/2.0}last-name')
                    author_name = ""
                    if first_name is not None and first_name.text:
                        author_name += first_name.text + " "
                    if last_name is not None and last_name.text:
                        author_name += last_name.text
                    if author_name.strip():
                        author = author_name.strip()
        
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
                html_content += "</h2>\\n"
                return html_content
            elif tag == 'p':
                html_content = "<p>"
                if element.text:
                    html_content += element.text
                html_content += "</p>\\n"
                return html_content
            elif tag == 'empty-line':
                return "<br/>\\n"
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
        
        # Process body content
        body = root.find('.//{http://www.gribuser.ru/xml/fictionbook/2.0}body')
        if body is None:
            print("No body content found in FB2 file")
            return False
        
        # Generate HTML content
        html_content = f"""
<!DOCTYPE html>
<html lang="sr">
<head>
    <meta charset="UTF-8">
    <title>{title}</title>
    <style>
        @page {{
            margin: 2cm;
            @bottom-center {{
                content: counter(page);
                font-size: 10pt;
            }}
        }}
        body {{
            font-family: 'Times New Roman', serif;
            font-size: 12pt;
            line-height: 1.6;
            text-align: justify;
        }}
        h1 {{
            text-align: center;
            font-size: 18pt;
            margin-bottom: 20pt;
            color: #333;
            border-bottom: 1px solid #ccc;
            padding-bottom: 10pt;
        }}
        h2 {{
            font-size: 14pt;
            margin-top: 20pt;
            margin-bottom: 10pt;
            color: #666;
            page-break-after: avoid;
        }}
        p {{
            text-indent: 1.5em;
            margin: 0;
            margin-bottom: 6pt;
        }}
        .title-page {{
            text-align: center;
            page-break-after: always;
        }}
        .title-page h1 {{
            font-size: 24pt;
            margin: 100pt 0;
            border: none;
        }}
        .title-page .author {{
            font-size: 16pt;
            margin: 50pt 0;
            font-style: italic;
        }}
        em {{
            font-style: italic;
        }}
        strong {{
            font-weight: bold;
        }}
    </style>
</head>
<body>
    <div class="title-page">
        <h1>{title}</h1>
        <div class="author">{author}</div>
    </div>
"""
        
        # Process sections
        section_num = 1
        for section in body.findall('{http://www.gribuser.ru/xml/fictionbook/2.0}section'):
            html_content += f"<h1>Поглавље {section_num}</h1>\\n"
            
            # Process section title
            title = section.find('{http://www.gribuser.ru/xml/fictionbook/2.0}title')
            if title is not None:
                title_html = element_to_html(title)
                html_content += title_html + "\\n"
            
            # Process section content
            for element in section:
                if element.tag != '{http://www.gribuser.ru/xml/fictionbook/2.0}title':
                    html_content += element_to_html(element)
            
            section_num += 1
        
        html_content += "</body>\\n</html>"
        
        # Create PDF
        html_doc = HTML(string=html_content)
        css_doc = CSS(string='''
            @page {
                margin: 2cm;
                @bottom-center {
                    content: counter(page);
                    font-size: 10pt;
                }
            }
        ''')
        
        html_doc.write_pdf(output_file, stylesheets=[css_doc])
        
        print(f"PDF conversion completed!")
        print(f"Output file: {output_file}")
        print(f"Sections: {section_num - 1}")
        
        return True
        
    except Exception as e:
        print(f"Error converting FB2 to PDF: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 fb2_to_pdf.py <input_file.fb2> [output_file.pdf]")
        print("Example: python3 fb2_to_pdf.py book_sr.b2 book_sr.pdf")
        return False
    
    input_file = sys.argv[1]
    
    # Generate output filename if not provided
    if len(sys.argv) >= 3:
        output_file = sys.argv[2]
    else:
        input_path = Path(input_file)
        stem = input_path.stem
        output_file = f"{stem}.pdf"
    
    if not Path(input_file).exists():
        print(f"Input file {input_file} not found")
        return False
    
    print("Converting FB2 to PDF...")
    success = fb2_to_pdf(input_file, output_file)
    
    if success:
        print("\\nPDF conversion completed successfully!")
        print(f"You can now read the book in any PDF reader.")
    
    return success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)