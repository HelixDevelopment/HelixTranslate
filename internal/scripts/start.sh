#!/bin/bash
# Universal Ebook Translator - Start Script
# This script starts all Docker services

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}==================================================================${NC}"
echo -e "${GREEN}  Universal Ebook Translator - Docker Services Startup${NC}"
echo -e "${GREEN}==================================================================${NC}"
echo

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found. Creating from .env.example...${NC}"
    cp .env.example .env
    echo -e "${YELLOW}Please edit .env file with your actual configuration!${NC}"
    echo
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if TLS certificates exist
if [ ! -f certs/server.crt ] || [ ! -f certs/server.key ]; then
    echo -e "${YELLOW}TLS certificates not found. Generating self-signed certificates...${NC}"
    make generate-certs || {
        echo -e "${RED}Failed to generate certificates. Please run 'make generate-certs' manually.${NC}"
        exit 1
    }
fi

# Parse command line arguments
PROFILE=""
BUILD=false
DETACH=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --admin)
            PROFILE="--profile admin"
            echo -e "${GREEN}Starting with admin tools (Adminer & Redis Commander)...${NC}"
            ;;
        --build)
            BUILD=true
            echo -e "${GREEN}Building images before starting...${NC}"
            ;;
        --foreground|-f)
            DETACH=false
            echo -e "${GREEN}Starting in foreground mode...${NC}"
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --admin          Start with admin tools (Adminer, Redis Commander)"
            echo "  --build          Build images before starting"
            echo "  --foreground, -f Run in foreground (don't detach)"
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

# Build images if requested
if [ "$BUILD" = true ]; then
    echo -e "${GREEN}Building Docker images...${NC}"
    docker-compose build
    echo
fi

# Start services
echo -e "${GREEN}Starting Docker services...${NC}"
if [ "$DETACH" = true ]; then
    docker-compose up -d $PROFILE
else
    docker-compose up $PROFILE
fi

if [ "$DETACH" = true ]; then
    echo
    echo -e "${GREEN}Services started successfully!${NC}"
    echo
    echo -e "${GREEN}Service Status:${NC}"
    docker-compose ps
    echo
    echo -e "${GREEN}Available Services:${NC}"
    echo -e "  • API Server:      https://localhost:$(grep API_PORT .env | cut -d '=' -f2 || echo 8443)"
    echo -e "  • PostgreSQL:      localhost:$(grep POSTGRES_PORT .env | cut -d '=' -f2 || echo 5432)"
    echo -e "  • Redis:           localhost:$(grep REDIS_PORT .env | cut -d '=' -f2 || echo 6379)"

    if [ -n "$PROFILE" ]; then
        echo
        echo -e "${GREEN}Admin Tools:${NC}"
        echo -e "  • Adminer:         http://localhost:$(grep ADMINER_PORT .env | cut -d '=' -f2 || echo 8081)"
        echo -e "  • Redis Commander: http://localhost:$(grep REDIS_COMMANDER_PORT .env | cut -d '=' -f2 || echo 8082)"
    fi

    echo
    echo -e "${GREEN}Useful Commands:${NC}"
    echo -e "  • View logs:   ${YELLOW}./scripts/logs.sh${NC}"
    echo -e "  • Stop all:    ${YELLOW}./scripts/stop.sh${NC}"
    echo -e "  • Restart:     ${YELLOW}./scripts/restart.sh${NC}"
    echo -e "  • CLI exec:    ${YELLOW}./scripts/exec.sh translator <command>${NC}"
    echo
fi
