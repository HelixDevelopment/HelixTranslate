#!/bin/bash
# Universal Ebook Translator - Exec Script
# This script executes commands inside Docker containers

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Show usage
usage() {
    echo "Usage: $0 <service> <command> [args...]"
    echo
    echo "Services:"
    echo "  translator    - Translator CLI tool"
    echo "  api           - API server"
    echo "  postgres      - PostgreSQL database"
    echo "  redis         - Redis cache"
    echo
    echo "Examples:"
    echo "  $0 translator -input Books/book.epub -locale de"
    echo "  $0 api /bin/sh"
    echo "  $0 postgres psql -U translator"
    echo "  $0 redis redis-cli"
    exit 1
}

if [ $# -lt 2 ]; then
    usage
fi

SERVICE=$1
shift

case $SERVICE in
    translator)
        CONTAINER="translator-api"
        CMD="/app/translator $@"
        ;;
    api)
        CONTAINER="translator-api"
        CMD="$@"
        ;;
    postgres)
        CONTAINER="translator-postgres"
        CMD="$@"
        ;;
    redis)
        CONTAINER="translator-redis"
        CMD="$@"
        ;;
    *)
        echo -e "${RED}Unknown service: $SERVICE${NC}"
        usage
        ;;
esac

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    echo -e "${RED}Error: Container '$CONTAINER' is not running.${NC}"
    echo -e "${YELLOW}Start services with: ./scripts/start.sh${NC}"
    exit 1
fi

# Execute command
echo -e "${GREEN}Executing command in $CONTAINER...${NC}"
docker exec -it $CONTAINER $CMD
