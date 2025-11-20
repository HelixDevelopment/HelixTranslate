#!/bin/bash
# Batch translate multiple texts

curl -X POST https://localhost:8443/api/v1/translate/batch \
  -H "Content-Type: application/json" \
  -d '{
    "texts": [
      "герой",
      "мир",
      "человек",
      "любовь",
      "смерть"
    ],
    "provider": "dictionary"
  }' \
  --insecure

echo ""
