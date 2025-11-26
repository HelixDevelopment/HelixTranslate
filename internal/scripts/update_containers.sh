#!/bin/bash
# update_containers.sh - Update and restart Docker containers for the distributed translation system

set -e

# Configuration
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
LOG_FILE="${LOG_FILE:-update.log}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1${NC}" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARN: $1${NC}" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}" | tee -a "$LOG_FILE"
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."

    # Check if docker and docker-compose are available
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    # Check if compose file exists
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        log_error "Compose file not found: $COMPOSE_FILE"
        exit 1
    fi

    log_success "Prerequisites validated"
}

# Create backup of current state
create_backup() {
    log_info "Creating backup of current state..."

    # Create backup directory
    mkdir -p "$BACKUP_DIR"
    backup_name="backup_$(date +%Y%m%d_%H%M%S)"
    backup_path="$BACKUP_DIR/$backup_name"

    mkdir -p "$backup_path"

    # Backup compose file
    cp "$COMPOSE_FILE" "$backup_path/"

    # Backup environment files
    if [[ -f ".env" ]]; then
        cp ".env" "$backup_path/"
    fi

    # Backup container logs
    log_info "Backing up container logs..."
    containers=$(docker-compose ps -q 2>/dev/null || docker compose ps -q 2>/dev/null || echo "")
    for container in $containers; do
        container_name=$(docker inspect --format='{{.Name}}' "$container" | sed 's/\///')
        docker logs "$container" > "$backup_path/${container_name}.log" 2>&1 || true
    done

    log_success "Backup created: $backup_path"
    echo "$backup_path"
}

# Update all services
update_all_services() {
    log_info "Updating all services..."

    # Pull latest images
    log_info "Pulling latest images..."
    if ! docker-compose pull; then
        log_error "Failed to pull images"
        return 1
    fi

    # Restart all services
    log_info "Restarting all services..."
    if ! docker-compose up -d; then
        log_error "Failed to restart services"
        return 1
    fi

    log_success "All services updated and restarted"
}

# Update specific service
update_service() {
    local service_name="$1"
    local new_image="$2"

    log_info "Updating service $service_name..."

    if [[ -n "$new_image" ]]; then
        log_info "Updating to image: $new_image"

        # Stop the service
        if ! docker-compose stop "$service_name"; then
            log_error "Failed to stop service $service_name"
            return 1
        fi

        # Remove the service
        if ! docker-compose rm -f "$service_name"; then
            log_error "Failed to remove service $service_name"
            return 1
        fi

        # Pull new image
        if ! docker pull "$new_image"; then
            log_error "Failed to pull image $new_image"
            return 1
        fi

        # Update the compose file with new image (if needed)
        # This would require parsing and updating the YAML file

        # Start the service
        if ! docker-compose up -d "$service_name"; then
            log_error "Failed to start updated service $service_name"
            return 1
        fi
    else
        # Just restart the service
        if ! docker-compose restart "$service_name"; then
            log_error "Failed to restart service $service_name"
            return 1
        fi
    fi

    log_success "Service $service_name updated"
}

# Restart all services
restart_all_services() {
    log_info "Restarting all services..."

    if ! docker-compose restart; then
        log_error "Failed to restart all services"
        return 1
    fi

    log_success "All services restarted"
}

# Restart specific service
restart_service() {
    local service_name="$1"

    log_info "Restarting service $service_name..."

    if ! docker-compose restart "$service_name"; then
        log_error "Failed to restart service $service_name"
        return 1
    fi

    log_success "Service $service_name restarted"
}

# Wait for services to be healthy
wait_for_healthy() {
    local timeout="${1:-300}"  # Default 5 minutes

    log_info "Waiting for services to become healthy (timeout: ${timeout}s)..."

    local start_time=$(date +%s)
    local end_time=$((start_time + timeout))

    while [[ $(date +%s) -lt $end_time ]]; do
        if docker-compose ps | grep -q "healthy\|running"; then
            local healthy_count=$(docker-compose ps | grep -c "healthy\|running" || echo "0")
            local total_count=$(docker-compose ps | grep -c "Up\|running\|healthy" || echo "0")

            if [[ $healthy_count -eq $total_count ]] && [[ $total_count -gt 0 ]]; then
                log_success "All services are healthy"
                return 0
            fi
        fi

        sleep 10
    done

    log_error "Timeout waiting for services to become healthy"
    return 1
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] ACTION

Update and restart Docker containers for the distributed translation system.

ACTIONS:
    update-all              Update all services to latest images and restart
    update <service>        Update specific service (optionally specify new image with -i)
    restart-all             Restart all services
    restart <service>       Restart specific service

OPTIONS:
    -c, --compose FILE      Docker compose file (default: docker-compose.yml)
    -i, --image IMAGE       New image for update action
    -b, --backup-dir DIR    Backup directory (default: ./backups)
    -l, --log FILE          Log file (default: update.log)
    --no-backup             Skip backup creation
    --no-wait               Don't wait for services to become healthy
    -h, --help              Show this help

EXAMPLES:
    # Update all services
    $0 update-all

    # Update specific service to new image
    $0 -i translator:latest update translator-main

    # Restart all services
    $0 restart-all

    # Restart specific service
    $0 restart translator-worker-1

ENVIRONMENT VARIABLES:
    COMPOSE_FILE            Docker compose file
    BACKUP_DIR              Backup directory
    LOG_FILE                Log file

EOF
}

# Main execution
main() {
    local skip_backup=false
    local skip_wait=false
    local new_image=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--compose)
                COMPOSE_FILE="$2"
                shift 2
                ;;
            -i|--image)
                new_image="$2"
                shift 2
                ;;
            -b|--backup-dir)
                BACKUP_DIR="$2"
                shift 2
                ;;
            -l|--log)
                LOG_FILE="$2"
                shift 2
                ;;
            --no-backup)
                skip_backup=true
                shift
                ;;
            --no-wait)
                skip_wait=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                break
                ;;
        esac
    done

    local action="$1"
    local service_name="$2"

    # Validate action
    case $action in
        update-all|restart-all)
            ;;
        update|restart)
            if [[ -z "$service_name" ]]; then
                log_error "Service name is required for $action action"
                usage
                exit 1
            fi
            ;;
        *)
            log_error "Unknown action: $action"
            usage
            exit 1
            ;;
    esac

    log_info "=== Container Update Script Started ==="
    log_info "Compose file: $COMPOSE_FILE"
    log_info "Action: $action"
    if [[ -n "$service_name" ]]; then
        log_info "Service: $service_name"
    fi
    if [[ -n "$new_image" ]]; then
        log_info "New image: $new_image"
    fi

    # Validate prerequisites
    validate_prerequisites

    # Create backup (unless skipped)
    local backup_path=""
    if [[ "$skip_backup" != true ]]; then
        backup_path=$(create_backup)
    fi

    # Execute action
    local success=false
    case $action in
        update-all)
            if update_all_services; then
                success=true
            fi
            ;;
        update)
            if update_service "$service_name" "$new_image"; then
                success=true
            fi
            ;;
        restart-all)
            if restart_all_services; then
                success=true
            fi
            ;;
        restart)
            if restart_service "$service_name"; then
                success=true
            fi
            ;;
    esac

    # Wait for healthy (unless skipped)
    if [[ "$skip_wait" != true ]] && [[ "$success" == true ]]; then
        if ! wait_for_healthy; then
            log_error "Services failed to become healthy after update"
            success=false
        fi
    fi

    # Final status
    if [[ "$success" == true ]]; then
        log_success "=== Update completed successfully ==="
        if [[ -n "$backup_path" ]]; then
            log_info "Backup available at: $backup_path"
        fi
        exit 0
    else
        log_error "=== Update failed ==="
        if [[ -n "$backup_path" ]]; then
            log_warn "You can restore from backup at: $backup_path"
        fi
        exit 1
    fi
}

# Run main function
main "$@"</content>
</xai:function_call">/dev/null