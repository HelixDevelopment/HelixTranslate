#!/bin/bash
# Universal Ebook Translator - Restart Script
# This script restarts Docker services

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}==================================================================${NC}"
echo -e "${GREEN}  Universal Ebook Translator - Docker Services Restart${NC}"
echo -e "${GREEN}==================================================================${NC}"
echo

# Parse command line arguments
SERVICE=""
BUILD=false
ADMIN=false

usage() {
    echo "Usage: $0 [OPTIONS] [SERVICE]"
    echo
    echo "Services:"
    echo "  api           - Restart API server only"
    echo "  postgres      - Restart PostgreSQL only"
    echo "  redis         - Restart Redis only"
    echo "  all (default) - Restart all services"
    echo
    echo "Options:"
    echo "  --build      Build images before restarting"
    echo "  --admin      Include admin tools"
    echo "  -h, --help   Show this help message"
    echo
    echo "Examples:"
    echo "  $0                # Restart all services"
    echo "  $0 api            # Restart only API server"
    echo "  $0 --build        # Rebuild and restart all"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD=true
            shift
            ;;
        --admin)
            ADMIN=true
            shift
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

# Map service names
case $SERVICE in
    api)
        SERVICE_NAME="translator-api"
        ;;
    postgres)
        SERVICE_NAME="postgres"
        ;;
    redis)
        SERVICE_NAME="redis"
        ;;
    all)
        SERVICE_NAME=""
        ;;
    *)
        echo -e "${RED}Unknown service: $SERVICE${NC}"
        usage
        ;;
esac

# Build if requested
if [ "$BUILD" = true ]; then
    echo -e "${GREEN}Building Docker images...${NC}"
    docker-compose build $SERVICE_NAME
    echo
fi

# Restart services
if [ -z "$SERVICE_NAME" ]; then
    echo -e "${GREEN}Restarting all services...${NC}"
    ./scripts/stop.sh
    if [ "$ADMIN" = true ]; then
        ./scripts/start.sh --admin
    else
        ./scripts/start.sh
    fi
else
    echo -e "${GREEN}Restarting $SERVICE...${NC}"
    docker-compose restart $SERVICE_NAME
    echo
    echo -e "${GREEN}Service restarted successfully!${NC}"
    docker-compose ps $SERVICE_NAME
fi
