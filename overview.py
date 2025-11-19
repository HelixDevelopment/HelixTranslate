#!/usr/bin/env python3
"""
FB2 Translation Tools Overview
============================

This script provides an overview of all translation tools available
for converting Russian FB2 books to Serbian.
"""

import os
from pathlib import Path

def show_file_info(filename, description):
    """Display information about a file"""
    if Path(filename).exists():
        size = os.path.getsize(filename) / 1024  # KB
        print(f"✓ {filename} ({size:.1f} KB) - {description}")
    else:
        print(f"✗ {filename} - NOT FOUND")

def main():
    print("FB2 Translation Tools - Russian to Serbian")
    print("=" * 50)
    print()
    
    print("Source Files:")
    show_file_info("Ratibor_1f.b2", "Original Russian book")
    print()
    
    print("Translation Scripts:")
    show_file_info("fb2_translator.py", "Main translation tool")
    show_file_info("translation_helper.py", "Translation management helper")
    show_file_info("sample_translation.py", "Sample translator")
    print()
    
    print("Generated Translation Files:")
    show_file_info("Ratibor_1f_sr_template.b2", "Template for manual translation")
    show_file_info("translation_list.txt", "Text list for translation")
    show_file_info("Ratibor_1f_sr_sample.b2", "Sample with partial translation")
    print()
    
    print("Documentation:")
    show_file_info("TRANSLATION_GUIDE.md", "Comprehensive translation guide")
    show_file_info("AGENTS.md", "Repository information for agents")
    print()
    
    print("Quick Start Guide:")
    print("1. For automatic translation: python3 fb2_translator.py (option 1)")
    print("2. For manual translation: python3 translation_helper.py (option 1)")
    print("3. To apply translations: python3 translation_helper.py (option 2)")
    print("4. For detailed instructions: Read TRANSLATION_GUIDE.md")
    print()
    
    print("Dependencies:")
    try:
        import googletrans
        print("✓ googletrans library installed")
    except ImportError:
        print("✗ googletrans library not installed")
        print("  Install with: pip3 install googletrans==4.0.0-rc1")
    
    print("\nNeed help? Check TRANSLATION_GUIDE.md for detailed instructions")

if __name__ == "__main__":
    main()