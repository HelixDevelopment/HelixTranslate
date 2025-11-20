#!/bin/bash
# Translate using OpenAI GPT-4

export OPENAI_API_KEY="your-api-key-here"

curl -X POST https://localhost:8443/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{
    "text": "В темном лесу жил отважный герой, который мечтал о великих приключениях.",
    "provider": "openai",
    "model": "gpt-4",
    "context": "Fantasy literature",
    "script": "cyrillic"
  }' \
  --insecure

echo ""
