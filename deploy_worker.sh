#!/bin/bash
expect << EOF
spawn scp -o StrictHostKeyChecking=no -r . milosvasic@thinker.local:~/translator-src
expect "password:"
send "WhiteSnake8587\r"
expect eof
EOF

expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "cd ~/translator-src && go mod tidy\r"
expect "$ "
send "cd ~/translator-src && go build -o translator-linux ./cmd/server\r"
expect "$ "
send "cd ~/translator-src && cp translator-linux ~/translator\r"
expect "$ "
send "cd ~ && chmod +x translator\r"
expect "$ "
send "cd ~ && ./translator --config config.worker.json > worker.log 2>&1 & echo \$! > worker.pid\r"
expect "$ "
send "sleep 3\r"
expect "$ "
send "curl -k https://localhost:8443/health\r"
expect "$ "
send "curl -k https://localhost:8443/api/v1/providers\r"
expect "$ "
send "exit\r"
expect eof
EOF