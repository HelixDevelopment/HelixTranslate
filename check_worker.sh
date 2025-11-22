#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "ps aux | grep translator\r"
expect "$ "
send "netstat -tlnp | grep 8443\r"
expect "$ "
send "curl -k https://localhost:8443/health 2>/dev/null || echo 'No service running'\r"
expect "$ "
send "exit\r"
expect eof
EOF