#!/bin/bash
# Universal Ebook Translator - Stop Script
# This script stops all Docker services

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}==================================================================${NC}"
echo -e "${GREEN}  Universal Ebook Translator - Docker Services Shutdown${NC}"
echo -e "${GREEN}==================================================================${NC}"
echo

# Parse command line arguments
REMOVE_VOLUMES=false
REMOVE_ORPHANS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --volumes|-v)
            REMOVE_VOLUMES=true
            echo -e "${YELLOW}Warning: This will remove all volumes and data!${NC}"
            ;;
        --all|-a)
            REMOVE_ORPHANS=true
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --volumes, -v    Remove volumes (WARNING: deletes all data)"
            echo "  --all, -a        Remove orphaned containers"
            echo "  --help, -h       Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
    shift
done

# Confirm volume removal
if [ "$REMOVE_VOLUMES" = true ]; then
    echo -e "${RED}WARNING: You are about to delete all volumes and data!${NC}"
    read -p "Are you sure you want to continue? (yes/no): " confirmation
    if [ "$confirmation" != "yes" ]; then
        echo -e "${YELLOW}Operation cancelled.${NC}"
        exit 0
    fi
fi

# Stop services
echo -e "${GREEN}Stopping Docker services...${NC}"

STOP_CMD="docker-compose down"

if [ "$REMOVE_VOLUMES" = true ]; then
    STOP_CMD="$STOP_CMD -v"
fi

if [ "$REMOVE_ORPHANS" = true ]; then
    STOP_CMD="$STOP_CMD --remove-orphans"
fi

eval $STOP_CMD

echo
echo -e "${GREEN}Services stopped successfully!${NC}"
echo

if [ "$REMOVE_VOLUMES" = false ]; then
    echo -e "${YELLOW}Note: Data volumes were preserved. Use --volumes to remove them.${NC}"
fi
