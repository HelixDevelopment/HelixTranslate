#!/bin/bash
echo "Starting SSH translation demonstration..."
echo "This will translate the full book using remote SSH worker with llama.cpp"
echo ""
./translate-ssh -input="internal/materials/books/book1.fb2" \
                -output="internal/materials/books/book1_sr.epub" \
                -host="thinker.local" \
                -user="milosvasic" \
                -password="${SSH_WORKER_PASSWORD}" \
                -port=22 \
                -remote-dir="/tmp/translation-workspace"
