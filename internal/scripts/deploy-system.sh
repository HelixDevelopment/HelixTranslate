#!/bin/bash
# deploy-system.sh - Complete automated deployment

set -e

echo "🚀 Starting Universal Ebook Translator Deployment"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONFIG_FILE="${CONFIG_FILE:-config.distributed.json}"
PLAN_FILE="${PLAN_FILE:-deployment-plan.json}"
CLI_BINARY="${CLI_BINARY:-./build/deployment-cli}"

# Validate prerequisites
validate_prerequisites() {
    echo -e "${YELLOW}Validating prerequisites...${NC}"

    # Check if CLI binary exists
    if [[ ! -f "$CLI_BINARY" ]]; then
        echo -e "${RED}Error: CLI binary not found at $CLI_BINARY${NC}"
        echo "Run 'make build-deployment' first"
        exit 1
    fi

    # Check if config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo -e "${RED}Error: Configuration file not found at $CONFIG_FILE${NC}"
        exit 1
    fi

    # Check SSH keys
    if [[ -z "$SSH_KEY_PATH" ]]; then
        echo -e "${YELLOW}Warning: SSH_KEY_PATH not set, using default key locations${NC}"
    fi

    echo -e "${GREEN}Prerequisites validated${NC}"
}

# Generate deployment plan
generate_plan() {
    echo -e "${YELLOW}Generating deployment plan...${NC}"

    if [[ -f "$PLAN_FILE" ]]; then
        echo -e "${YELLOW}Plan file already exists, backing up...${NC}"
        mv "$PLAN_FILE" "${PLAN_FILE}.backup.$(date +%s)"
    fi

    "$CLI_BINARY" -action generate-plan -config "$CONFIG_FILE" -verbose

    if [[ ! -f "$PLAN_FILE" ]]; then
        echo -e "${RED}Error: Failed to generate deployment plan${NC}"
        exit 1
    fi

    echo -e "${GREEN}Deployment plan generated: $PLAN_FILE${NC}"
}

# Deploy the system
deploy_system() {
    echo -e "${YELLOW}Deploying distributed system...${NC}"

    "$CLI_BINARY" -action deploy -plan "$PLAN_FILE" -verbose

    echo -e "${GREEN}System deployment initiated${NC}"
}

# Wait for deployment completion
wait_for_deployment() {
    echo -e "${YELLOW}Waiting for deployment to complete...${NC}"

    local max_attempts=60  # 10 minutes with 10s intervals
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        echo -e "${YELLOW}Checking deployment status (attempt $attempt/$max_attempts)...${NC}"

        # Check if deployment is complete
        if "$CLI_BINARY" -action status >/dev/null 2>&1; then
            local status_output
            status_output=$("$CLI_BINARY" -action status 2>/dev/null)

            if echo "$status_output" | grep -q "healthy"; then
                echo -e "${GREEN}Deployment completed successfully!${NC}"
                return 0
            fi
        fi

        sleep 10
        ((attempt++))
    done

    echo -e "${RED}Deployment did not complete within timeout${NC}"
    return 1
}

# Validate deployment
validate_deployment() {
    echo -e "${YELLOW}Validating deployment...${NC}"

    local status_output
    status_output=$("$CLI_BINARY" -action status)

    echo "$status_output"

    # Check for healthy instances
    local healthy_count
    healthy_count=$(echo "$status_output" | grep -c "healthy" || true)

    if [[ $healthy_count -eq 0 ]]; then
        echo -e "${RED}Error: No healthy instances found${NC}"
        return 1
    fi

    echo -e "${GREEN}Validation successful: $healthy_count healthy instances${NC}"
}

# Main deployment flow
main() {
    echo -e "${GREEN}=== Universal Ebook Translator Deployment ===${NC}"

    validate_prerequisites
    generate_plan
    deploy_system

    if wait_for_deployment; then
        validate_deployment
        echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
        echo ""
        echo "Next steps:"
        echo "1. Check API logs: tail -f workers_api_communication.log"
        echo "2. Monitor health: $CLI_BINARY -action status"
        echo "3. Scale workers: docker-compose up -d --scale translator-worker-1=3"
    else
        echo -e "${RED}❌ Deployment failed${NC}"
        echo ""
        echo "Troubleshooting:"
        echo "1. Check logs: $CLI_BINARY -action status"
        echo "2. View container logs: docker-compose logs"
        echo "3. Cleanup and retry: $CLI_BINARY -action cleanup"
        exit 1
    fi
}

# Run main function
main "$@"</content>
</xai:function_call">  
<xai:function_call name="write">
<parameter name="filePath">scripts/monitor-deployment.sh