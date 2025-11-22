#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "which go\r"
expect "$ "
send "go version\r"
expect "$ "
send "exit\r"
expect eof
EOF