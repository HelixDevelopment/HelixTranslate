#!/bin/bash
expect << EOF
spawn scp -o StrictHostKeyChecking=no config.worker.json milosvasic@thinker.local:~/config.worker.json
expect "password:"
send "WhiteSnake8587\r"
expect eof
EOF

expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "cd ~ && ./translator --config config.worker.json > worker.log 2>&1 & echo \$! > worker.pid\r"
expect "$ "
send "sleep 2\r"
expect "$ "
send "curl -k https://localhost:8443/health\r"
expect "$ "
send "curl -k https://localhost:8443/api/v1/providers\r"
expect "$ "
send "exit\r"
expect eof
EOF