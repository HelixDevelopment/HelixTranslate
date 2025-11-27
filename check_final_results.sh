#!/bin/bash
# Check final translation status

echo "=== Checking Final Translation Results ==="

ssh milosvasic@thinker.local "cd /tmp/translate-ssh && \
echo '=== Translation Files ===' && \
ls -la *translated* 2>/dev/null && \
echo '' && \
echo '=== Production Translation Status ===' && \
if [ -f book1_production_translated.md ]; then \
  echo '✅ Production translation completed!'; \
  echo 'File size:'; \
  wc -c book1_production_translated.md; \
  echo 'Line count:'; \
  wc -l book1_production_translated.md; \
  echo '' ;\
  echo '=== Sample Output ==='; \
  head -30 book1_production_translated.md; \
else \
  echo '⏳ Production translation not yet completed'; \
  echo 'Recent logs:'; \
  tail -20 production_translation.log 2>/dev/null || echo 'No log file found'; \
fi"

echo ""
echo "=== System Performance Summary ===" 
echo "GPU Status:"
ssh milosvasic@thinker.local "nvidia-smi --query-gpu=name,utilization.gpu,memory.used --format=csv,noheader,nounits 2>/dev/null || echo 'GPU query failed'"