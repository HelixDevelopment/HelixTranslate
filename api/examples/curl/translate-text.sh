#!/bin/bash
# Translate Russian text to Serbian using dictionary provider

curl -X POST https://localhost:8443/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Привет, мир! Это тестовое сообщение.",
    "provider": "dictionary",
    "script": "cyrillic"
  }' \
  --insecure

echo ""
