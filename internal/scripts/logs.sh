#!/bin/bash
# Universal Ebook Translator - Logs Script
# This script shows logs from Docker containers

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Parse command line arguments
SERVICE=""
FOLLOW=false
TAIL=100

usage() {
    echo "Usage: $0 [OPTIONS] [SERVICE]"
    echo
    echo "Services:"
    echo "  api           - API server logs"
    echo "  postgres      - PostgreSQL logs"
    echo "  redis         - Redis logs"
    echo "  all (default) - All services logs"
    echo
    echo "Options:"
    echo "  -f, --follow       Follow log output"
    echo "  -t, --tail N       Number of lines to show (default: 100)"
    echo "  -h, --help         Show this help message"
    echo
    echo "Examples:"
    echo "  $0                  # Show last 100 lines of all services"
    echo "  $0 -f api           # Follow API server logs"
    echo "  $0 -t 500 postgres  # Show last 500 lines of PostgreSQL"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -t|--tail)
            TAIL=$2
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        api|postgres|redis|all)
            SERVICE=$1
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

# Default to all services
if [ -z "$SERVICE" ]; then
    SERVICE="all"
fi

# Map service names to container names
case $SERVICE in
    api)
        CONTAINER="translator-api"
        ;;
    postgres)
        CONTAINER="translator-postgres"
        ;;
    redis)
        CONTAINER="translator-redis"
        ;;
    all)
        CONTAINER=""
        ;;
    *)
        echo -e "${RED}Unknown service: $SERVICE${NC}"
        usage
        ;;
esac

# Build docker-compose logs command
CMD="docker-compose logs --tail=$TAIL"

if [ "$FOLLOW" = true ]; then
    CMD="$CMD -f"
fi

if [ -n "$CONTAINER" ]; then
    # Get service name from container name
    SERVICE_NAME=$(echo $CONTAINER | sed 's/translator-//')
    CMD="$CMD $SERVICE_NAME"
fi

# Execute command
echo -e "${GREEN}Showing logs for: $SERVICE${NC}"
echo
eval $CMD
