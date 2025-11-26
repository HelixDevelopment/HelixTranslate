#!/bin/bash

# WebSocket Monitoring System - Quick Demo Script
# This script demonstrates the complete WebSocket monitoring workflow

set -e

echo "🚀 WebSocket Monitoring System - Quick Demo"
echo "=========================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if dependencies are available
check_dependencies() {
    print_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v lsof &> /dev/null; then
        print_warning "lsof not available, some checks will be skipped"
    fi
    
    print_status "Dependencies check passed"
}

# Check if port is available
check_port() {
    local port=$1
    if lsof -i :$port &> /dev/null; then
        print_warning "Port $port is already in use"
        return 1
    fi
    return 0
}

# Start monitoring server
start_monitoring_server() {
    print_info "Starting WebSocket monitoring server..."
    
    if check_port 8090; then
        # Start server in background
        go run ./cmd/monitor-server > monitor-server.log 2>&1 &
        MONITOR_PID=$!
        
        # Wait for server to start
        sleep 2
        
        # Check if server is running
        if kill -0 $MONITOR_PID 2>/dev/null; then
            print_status "Monitoring server started (PID: $MONITOR_PID)"
            print_info "WebSocket: ws://localhost:8090/ws"
            print_info "Dashboard: http://localhost:8090/monitor"
        else
            print_error "Failed to start monitoring server"
            exit 1
        fi
    else
        print_error "Port 8090 is already in use"
        print_info "Please stop the existing server or choose a different port"
        exit 1
    fi
}

# Open monitoring dashboard
open_dashboard() {
    print_info "Opening monitoring dashboard..."
    
    # Try to open in default browser
    if command -v open &> /dev/null; then
        open http://localhost:8090/monitor
    elif command -v xdg-open &> /dev/null; then
        xdg-open http://localhost:8090/monitor
    else
        print_warning "Could not auto-open browser. Please visit: http://localhost:8090/monitor"
    fi
}

# Run translation demo
run_translation_demo() {
    print_info "Starting translation demo with WebSocket monitoring..."
    
    # Check if demo file exists
    if [ ! -f "test/fixtures/ebooks/russian_sample.txt" ]; then
        print_error "Test input file not found: test/fixtures/ebooks/russian_sample.txt"
        exit 1
    fi
    
    # Run demo in background
    echo -e "${BLUE}🔄 Running translation demo...${NC}"
    go run demo-translation-with-monitoring-fixed.go > translation-demo.log 2>&1 &
    DEMO_PID=$!
    
    print_status "Translation demo started (PID: $DEMO_PID)"
    print_info "Log file: translation-demo.log"
    
    # Wait a bit to see progress
    sleep 3
}

# Show live progress
show_live_progress() {
    print_info "Showing live progress from demo..."
    
    if [ -f "translation-demo.log" ]; then
        echo -e "${BLUE}📊 Recent progress:${NC}"
        tail -n 10 translation-demo.log
    fi
}

# Run SSH worker demo (optional)
run_ssh_worker_demo() {
    print_info "Starting SSH worker demo..."
    
    # Set environment variables for SSH worker (if not set)
    if [ -z "$SSH_WORKER_HOST" ]; then
        export SSH_WORKER_HOST=localhost
        print_info "Setting SSH_WORKER_HOST=localhost"
    fi
    
    if [ -z "$SSH_WORKER_USER" ]; then
        export SSH_WORKER_USER=milosvasic
        print_info "Setting SSH_WORKER_USER=milosvasic"
    fi
    
    # Run SSH worker demo
    go run demo-ssh-worker-with-monitoring.go > ssh-worker-demo.log 2>&1 &
    SSH_DEMO_PID=$!
    
    print_status "SSH worker demo started (PID: $SSH_DEMO_PID)"
    print_info "Log file: ssh-worker-demo.log"
}

# Cleanup function
cleanup() {
    print_info "Cleaning up..."
    
    # Kill background processes
    if [ ! -z "$MONITOR_PID" ] && kill -0 $MONITOR_PID 2>/dev/null; then
        kill $MONITOR_PID
        print_status "Monitoring server stopped"
    fi
    
    if [ ! -z "$DEMO_PID" ] && kill -0 $DEMO_PID 2>/dev/null; then
        kill $DEMO_PID
        print_status "Translation demo stopped"
    fi
    
    if [ ! -z "$SSH_DEMO_PID" ] && kill -0 $SSH_DEMO_PID 2>/dev/null; then
        kill $SSH_DEMO_PID
        print_status "SSH worker demo stopped"
    fi
    
    # Clean up log files (optional)
    # rm -f monitor-server.log translation-demo.log ssh-worker-demo.log
}

# Show results
show_results() {
    print_info "Checking results..."
    
    # Check output files
    if [ -f "demo_translation_output.md" ]; then
        print_status "Basic demo output: demo_translation_output.md"
        echo -e "${BLUE}📄 Content preview:${NC}"
        head -5 demo_translation_output.md
    fi
    
    if [ -f "demo_ssh_worker_output.md" ]; then
        print_status "SSH worker demo output: demo_ssh_worker_output.md"
        echo -e "${BLUE}📄 Content preview:${NC}"
        head -5 demo_ssh_worker_output.md
    fi
    
    if [ -f "demo_real_llm_output.md" ]; then
        print_status "Real LLM demo output: demo_real_llm_output.md"
        echo -e "${BLUE}📄 Content preview:${NC}"
        head -5 demo_real_llm_output.md
    fi
}

# Interactive menu
show_menu() {
    echo -e "\n${BLUE}🎯 Choose demo option:${NC}"
    echo "1) Basic WebSocket monitoring demo (Recommended)"
    echo "2) Real LLM translation with monitoring"
    echo "3) SSH worker translation demo"
    echo "4) All demos (sequentially)"
    echo "5) Show live progress"
    echo "6) View results"
    echo "7) Open enhanced dashboard"
    echo "8) Cleanup and exit"
    
    read -p "Enter your choice (1-8): " choice
    
    case $choice in
        1)
            run_translation_demo
            sleep 5
            show_live_progress
            show_menu
            ;;
        2)
            run_real_llm_demo
            sleep 10
            show_live_progress
            show_menu
            ;;
        3)
            run_ssh_worker_demo
            sleep 10
            show_live_progress
            show_menu
            ;;
        4)
            print_info "Running all demos sequentially..."
            run_translation_demo
            sleep 5
            run_real_llm_demo
            sleep 10
            run_ssh_worker_demo
            sleep 10
            show_results
            show_menu
            ;;
        5)
            show_live_progress
            show_menu
            ;;
        6)
            show_results
            show_menu
            ;;
        7)
            open_enhanced_dashboard
            show_menu
            ;;
        8)
            cleanup
            print_status "Demo completed. Goodbye!"
            exit 0
            ;;
        *)
            print_error "Invalid choice. Please try again."
            show_menu
            ;;
    esac
}

# Run real LLM demo
run_real_llm_demo() {
    print_info "Starting real LLM translation demo..."
    
    # Check for OpenAI API key
    if [ -z "$OPENAI_API_KEY" ]; then
        print_warning "OPENAI_API_KEY not set. Running in demo mode."
    fi
    
    # Run real LLM demo
    go run demo-real-llm-with-monitoring.go > real-llm-demo.log 2>&1 &
    LLM_DEMO_PID=$!
    
    print_status "Real LLM demo started (PID: $LLM_DEMO_PID)"
    print_info "Log file: real-llm-demo.log"
}

# Open enhanced dashboard
open_enhanced_dashboard() {
    print_info "Opening enhanced monitoring dashboard..."
    
    if [ -f "enhanced-monitor.html" ]; then
        if command -v open &> /dev/null; then
            open enhanced-monitor.html
        elif command -v xdg-open &> /dev/null; then
            xdg-open enhanced-monitor.html
        else
            print_warning "Could not auto-open browser. Please open: enhanced-monitor.html"
        fi
    else
        print_error "Enhanced dashboard not found: enhanced-monitor.html"
    fi
}

# Main execution
main() {
    # Setup cleanup on exit
    trap cleanup EXIT
    
    # Print welcome message
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║          WebSocket Translation Monitoring System Demo            ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    echo -e "${BLUE}This demo will show you:${NC}"
    echo "• Real-time WebSocket monitoring of translation workflows"
    echo "• Interactive web dashboard with progress tracking"
    echo "• SSH worker integration and monitoring"
    echo "• Event-driven architecture with comprehensive logging"
    echo ""
    
    # Check dependencies
    check_dependencies
    
    # Start monitoring server
    start_monitoring_server
    
    # Open dashboard
    open_dashboard
    
    # Show initial menu
    show_menu
}

# Run main function
main "$@"