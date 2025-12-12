#!/bin/bash

# Script to start dummy backend servers for testing
# These servers will run continuously until the script is terminated with Ctrl+C

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DUMMY_SERVER_BIN="$SCRIPT_DIR/cmd/dummy-server/main.go"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Backend configuration: name -> port:latency:error_rate
# Error rates are already decimals (0.0-1.0), no conversion needed
declare -A BACKENDS=(
    [backend-1]="8081:10:0.0"
    [backend-2]="8082:15:0.0"
    [backend-3]="8083:20:0.0"
)

# Array to track child PIDs
PIDS=()

log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_header() {
    echo ""
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}$*${NC}"
    echo -e "${BLUE}================================================${NC}"
}

# Cleanup function: stop all background processes
cleanup() {
    log_warn "Terminating all backend servers..."
    
    # Kill all child processes
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill -TERM "$pid" 2>/dev/null || true
        fi
    done
    
    # Wait for graceful shutdown
    sleep 1
    
    # Force kill any remaining processes
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
    
    log_info "All servers stopped"
}

# Set up trap to handle signals
trap cleanup EXIT INT TERM

# Verify Go is installed
if ! command -v go &> /dev/null; then
    log_error "Go is not installed"
    exit 1
fi

log_header "Starting Dummy Backend Servers"

# Build the dummy server binary once
log_info "Building dummy-server binary..."
if ! go build -o "$SCRIPT_DIR/dummy-server" "$DUMMY_SERVER_BIN" 2>/dev/null; then
    log_error "Failed to build dummy-server"
    exit 1
fi
log_info "Build successful"

echo ""
log_info "Starting backend servers..."
echo ""

# Start each backend server
for backend_name in "${!BACKENDS[@]}"; do
    IFS=':' read -r port latency error_rate <<< "${BACKENDS[$backend_name]}"
    
    log_info "Starting $backend_name on port $port (latency: ${latency}ms, error_rate: ${error_rate})"
    
    # Use the pre-built binary instead of go run for better startup time
    "$SCRIPT_DIR/dummy-server" \
        -port "$port" \
        -name "$backend_name" \
        -latency "$latency" \
        -error-rate "$error_rate" \
        > /dev/null 2>&1 &
    
    PIDS+=($!)
    
    # Brief pause between starting servers
    sleep 0.3
done

echo ""
log_header "Backends Running"

log_info "All backend servers have started"
echo ""
log_info "Available endpoints:"
for backend_name in "${!BACKENDS[@]}"; do
    IFS=':' read -r port _ _ <<< "${BACKENDS[$backend_name]}"
    echo "  - $backend_name: http://localhost:$port/"
    echo "    - Health check: http://localhost:$port/health"
done

echo ""
log_warn "Press Ctrl+C to stop all servers"
echo ""

# Wait for all background jobs indefinitely
# The trap will handle cleanup when the script receives a signal
wait