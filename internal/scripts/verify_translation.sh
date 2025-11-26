#!/bin/bash

# Verification script for translated ebook
set -euo pipefail

EBOOK_DIR="materials/books"
ORIGINAL_FB2="book1.fb2"
ORIGINAL_MD="book1_original.md"
TRANSLATED_MD="book1_translated.md"
TRANSLATED_EPUB="book1_sr.epub"

echo "=========================================="
echo "EBOOK TRANSLATION VERIFICATION REPORT"
echo "=========================================="
echo ""

# Check all required files exist
echo "Checking file existence:"
files=("$ORIGINAL_FB2" "$ORIGINAL_MD" "$TRANSLATED_MD" "$TRANSLATED_EPUB")
all_exist=true

for file in "${files[@]}"; do
    path="$EBOOK_DIR/$file"
    if [[ -f "$path" ]]; then
        size=$(wc -c < "$path")
        echo "✓ $path exists ($size bytes)"
    else
        echo "✗ $path NOT FOUND"
        all_exist=false
    fi
done

if [[ $all_exist == false ]]; then
    echo ""
    echo "ERROR: Not all files were created!"
    exit 1
fi

echo ""
echo "----------------------------------------"
echo "CONTENT VERIFICATION"
echo "----------------------------------------"

# Verify original FB2 has Russian text
echo "1. Checking original FB2 file for Russian content:"
russian_count=$(grep -c "[а-яА-Я]" "$EBOOK_DIR/$ORIGINAL_FB2" || echo "0")
echo "   - Russian characters found: $russian_count"

# Verify original markdown was created from FB2
echo ""
echo "2. Checking original markdown file:"
md_size=$(wc -c < "$EBOOK_DIR/$ORIGINAL_MD")
echo "   - File size: $md_size bytes"
has_title=$(grep -c "^# " "$EBOOK_DIR/$ORIGINAL_MD" || echo "0")
echo "   - Markdown headers found: $has_title"

# Verify translated markdown has Serbian Cyrillic
echo ""
echo "3. Checking translated markdown for Serbian Cyrillic:"
serbian_count=$(grep -c "[ђјљњћчћшџЂЈЉЊЋЧЋШЂ]" "$EBOOK_DIR/$TRANSLATED_MD" || echo "0")
echo "   - Serbian Cyrillic characters found: $serbian_count"

# Check for some specific Serbian translations
echo ""
echo "4. Checking for specific Serbian translations:"
translations=(
    "Крв на снегу"
    "Ја – убица"
    "мами"
    "наруџби"
)

for trans in "${translations[@]}"; do
    if grep -F "$trans" "$EBOOK_DIR/$TRANSLATED_MD" > /dev/null; then
        echo "   ✓ Found: $trans"
    else
        echo "   ✗ Missing: $trans"
    fi
done

# Verify EPUB structure
echo ""
echo "5. Checking translated EPUB file:"
epub_size=$(wc -c < "$EBOOK_DIR/$TRANSLATED_EPUB")
echo "   - EPUB size: $epub_size bytes"

# Check if EPUB has valid structure
if command -v unzip > /dev/null; then
    echo "   - EPUB structure verification:"
    if unzip -t "$EBOOK_DIR/$TRANSLATED_EPUB" > /dev/null 2>&1; then
        echo "     ✓ EPUB structure is valid"
        
        # Extract and check content
        temp_dir=$(mktemp -d)
        trap "rm -rf $temp_dir" EXIT
        
        unzip -q "$EBOOK_DIR/$TRANSLATED_EPUB" -d "$temp_dir"
        
        if [[ -f "$temp_dir/mimetype" ]]; then
            mimetype=$(cat "$temp_dir/mimetype")
            if [[ "$mimetype" == "application/epub+zip" ]]; then
                echo "     ✓ Correct mimetype"
            else
                echo "     ✗ Wrong mimetype: $mimetype"
            fi
        fi
        
        # Check for Serbian content in EPUB
        if find "$temp_dir" -name "*.xhtml" -o -name "*.html" | xargs grep -l "[ђјљњћчћшџ]" > /dev/null 2>&1; then
            echo "     ✓ Serbian Cyrillic found in EPUB content"
        else
            echo "     ✗ Serbian Cyrillic NOT found in EPUB content"
        fi
    else
        echo "     ✗ EPUB structure is INVALID"
    fi
else
    echo "   - unzip command not available, skipping structure check"
fi

echo ""
echo "----------------------------------------"
echo "SUMMARY"
echo "----------------------------------------"

if [[ $all_exist == true ]] && [[ $serbian_count -gt 100 ]]; then
    echo "✓ VERIFICATION PASSED"
    echo "  All required files created with proper Serbian Cyrillic content"
    echo ""
    echo "Generated files:"
    echo "  1. Original ebook: $EBOOK_DIR/$ORIGINAL_FB2"
    echo "  2. Original markdown: $EBOOK_DIR/$ORIGINAL_MD"
    echo "  3. Translated markdown: $EBOOK_DIR/$TRANSLATED_MD"
    echo "  4. Translated EPUB: $EBOOK_DIR/$TRANSLATED_EPUB"
    exit 0
else
    echo "✗ VERIFICATION FAILED"
    echo "  Some issues detected with the translation output"
    exit 1
fi