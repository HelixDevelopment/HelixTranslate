#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "cat ~/worker.log\r"
expect "$ "
send "ls -la ~/worker.pid\r"
expect "$ "
send "ps aux | grep translator\r"
expect "$ "
send "exit\r"
expect eof
EOF