#!/bin/bash
# Convert Serbian text from Cyrillic to Latin

curl -X POST https://localhost:8443/api/v1/convert/script \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Ратибор је јунак из фантастичне приче.",
    "target": "latin"
  }' \
  --insecure

echo ""
