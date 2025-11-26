#!/bin/bash
expect << EOF
spawn scp -o StrictHostKeyChecking=no server milosvasic@thinker.local:~/translator
expect "password:"
send "WhiteSnake8587\r"
expect eof
EOF

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
send "chmod +x ~/translator\r"
expect "$ "
send "mkdir -p ~/certs\r"
expect "$ "
send "openssl req -x509 -newkey rsa:4096 -keyout ~/certs/server.key -out ~/certs/server.crt -days 365 -nodes -subj \"/C=US/ST=State/L=City/O=Organization/CN=thinker.local\"\r"
expect "$ "
send "ls -la ~/\r"
expect "$ "
send "exit\r"
expect eof
EOF