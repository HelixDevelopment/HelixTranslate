#!/bin/bash
# Translate FB2 file

curl -X POST https://localhost:8443/api/v1/translate/fb2 \
  -F "file=@book.fb2" \
  -F "provider=dictionary" \
  -F "script=cyrillic" \
  --output book_translated.fb2 \
  --insecure

echo "Translation complete: book_translated.fb2"
