#!/bin/bash
expect << EOF
spawn ssh -o StrictHostKeyChecking=no milosvasic@thinker.local "echo 'SSH test successful'"
expect "password:"
send "$env(SSH_WORKER_PASSWORD)\r"
expect eof
EOF