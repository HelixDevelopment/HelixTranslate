#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "which llama.cpp\r"
expect "$ "
send "ls -la ~/llama.cpp*\r"
expect "$ "
send "ollama list 2>/dev/null || echo 'Ollama not available'\r"
expect "$ "
send "which llamacpp\r"
expect "$ "
send "find /usr -name "*llama*" 2>/dev/null | head -10\r"
expect "$ "
send "exit\r"
expect eof
EOF