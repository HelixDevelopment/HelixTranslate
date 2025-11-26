#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local "echo 'SSH test successful'"
expect "password:"
send "WhiteSnake8587\r"
expect eof
EOF