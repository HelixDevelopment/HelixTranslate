#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "mkdir -p ~/translator-project\r"
expect "$ "
send "cd ~/translator-project\r"
expect "$ "
send "exit\r"
expect eof
EOF

# Copy essential files
scp -o StrictHostKeyChecking=no go.mod go.sum milosvasic@thinker.local:~/translator-project/
scp -o StrictHostKeyChecking=no -r cmd milosvasic@thinker.local:~/translator-project/
scp -o StrictHostKeyChecking=no -r internal milosvasic@thinker.local:~/translator-project/
scp -o StrictHostKeyChecking=no -r pkg milosvasic@thinker.local:~/translator-project/

expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local
expect "password:"
send "WhiteSnake8587\r"
expect "$ "
send "cd ~/translator-project && go mod tidy\r"
expect "$ "
send "cd ~/translator-project && go build -o translator ./cmd/server\r"
expect "$ "
send "cd ~/translator-project && cp translator ~/\r"
expect "$ "
send "cd ~ && chmod +x translator\r"
expect "$ "
send "cp ~/config.worker.json ~/config.worker.json.backup\r"
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