#!/bin/bash
# Comprehensive test orchestration script for reverse proxy
# This script manages the full testing workflow: building, starting backends, running load tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
PROXY_PORT=8080
PROBE_PORT=8085
DUMMY_SERVER_BIN="$SCRIPT_DIR/dummy-server"
LOADGEN_BIN="$SCRIPT_DIR/loadgen"
PID_DIR="$SCRIPT_DIR/.pids"

# Test parameters (can be overridden by environment variables)
TEST_DURATION=${TEST_DURATION:-30s}
TEST_CLIENTS=${TEST_CLIENTS:-5}
TEST_RPS=${TEST_RPS:-50}
NUM_BACKENDS=${NUM_BACKENDS:-3}

# Helper functions
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

cleanup() {
    log_warn "Cleaning up..."
    
    # Stop backends
    if [ -d "$PID_DIR" ]; then
        for pid_file in "$PID_DIR"/*.pid; do
            if [ -f "$pid_file" ]; then
                pid=$(cat "$pid_file")
                if kill -0 "$pid" 2>/dev/null; then
                    log_info "Stopping process $pid..."
                    kill -TERM "$pid" 2>/dev/null || true
                fi
                rm -f "$pid_file"
            fi
        done
    fi
    
    # Stop proxy if running
    if [ -f "$PID_DIR/proxy.pid" ]; then
        pid=$(cat "$PID_DIR/proxy.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping proxy (PID: $pid)..."
            kill -TERM "$pid" 2>/dev/null || true
        fi
    fi
    
    sleep 1
    log_info "Cleanup complete"
}

trap cleanup EXIT INT TERM

build_binaries() {
    log_header "Building Binaries"
    
    if [ ! -f "$DUMMY_SERVER_BIN" ]; then
        log_info "Building dummy-server..."
        go build -o "$DUMMY_SERVER_BIN" "$SCRIPT_DIR/cmd/dummy-server/main.go"
    fi
    
    if [ ! -f "$LOADGEN_BIN" ]; then
        log_info "Building loadgen..."
        go build -o "$LOADGEN_BIN" "$SCRIPT_DIR/cmd/loadgen/main.go"
    fi
    
    log_info "Binaries ready"
}

start_backends() {
    log_header "Starting Backend Servers"
    
    mkdir -p "$PID_DIR"
    
    for i in $(seq 1 $NUM_BACKENDS); do
        PORT=$((8080 + i))
        NAME="backend-$i"
        
        # Vary parameters by backend
        LATENCY=$((5 + i * 5))
        ERROR_RATE=0
        
        log_info "Starting $NAME on port $PORT (latency: ${LATENCY}ms)..."
        
        "$DUMMY_SERVER_BIN" \
            -port "$PORT" \
            -name "$NAME" \
            -latency "$LATENCY" \
            -error-rate "$ERROR_RATE" \
            > "$PID_DIR/${NAME}.log" 2>&1 &
        
        PID=$!
        echo "$PID" > "$PID_DIR/${NAME}.pid"
        
        # Wait for server to be ready
        sleep 0.5
        
        if ! kill -0 "$PID" 2>/dev/null; then
            log_error "Failed to start $NAME"
            cat "$PID_DIR/${NAME}.log"
            exit 1
        fi
    done
    
    log_info "All backends started successfully"
}

wait_for_backends() {
    log_header "Waiting for Backends to be Healthy"
    
    local max_retries=30
    local retry_count=0
    
    for i in $(seq 1 $NUM_BACKENDS); do
        PORT=$((8080 + i))
        retry_count=0
        
        while [ $retry_count -lt $max_retries ]; do
            if curl -s -f "http://localhost:$PORT/health" > /dev/null 2>&1; then
                log_info "Backend $i is healthy"
                break
            fi
            
            retry_count=$((retry_count + 1))
            if [ $retry_count -eq 1 ]; then
                echo -n "Waiting for backend $i..."
            else
                echo -n "."
            fi
            
            sleep 0.5
        done
        
        if [ $retry_count -eq $max_retries ]; then
            log_error "Backend $i failed to become healthy"
            exit 1
        fi
        echo ""
    done
}

start_proxy() {
    log_header "Starting Reverse Proxy"
    
    mkdir -p "$PID_DIR"
    
    log_info "Starting proxy on port $PROXY_PORT..."
    
    cd "$PROJECT_ROOT"
    go run ./cmd/main.go \
        > "$PID_DIR/proxy.log" 2>&1 &
    
    PROXY_PID=$!
    echo "$PROXY_PID" > "$PID_DIR/proxy.pid"
    
    # Wait for proxy to be ready
    sleep 2
    
    if ! kill -0 "$PROXY_PID" 2>/dev/null; then
        log_error "Failed to start proxy"
        cat "$PID_DIR/proxy.log"
        exit 1
    fi
    
    log_info "Proxy started (PID: $PROXY_PID)"
}

wait_for_proxy() {
    log_header "Waiting for Proxy to be Ready"
    
    local max_retries=30
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        if curl -s -f "http://localhost:$PROBE_PORT/healthz" > /dev/null 2>&1; then
            log_info "Proxy is responding"
            break
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -eq 1 ]; then
            echo -n "Waiting for proxy..."
        else
            echo -n "."
        fi
        
        sleep 0.5
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_error "Proxy failed to become ready"
        cat "$PID_DIR/proxy.log"
        exit 1
    fi
    echo ""
    
    # Wait for readiness probe
    retry_count=0
    while [ $retry_count -lt $max_retries ]; do
        if curl -s -f "http://localhost:$PROBE_PORT/readyz" > /dev/null 2>&1; then
            log_info "Proxy is ready (backends healthy)"
            break
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -eq 1 ]; then
            echo -n "Waiting for backends to be ready..."
        else
            echo -n "."
        fi
        
        sleep 0.5
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_warn "Backends not healthy yet, but proxy is responsive"
    fi
    echo ""
}

run_load_test() {
    log_header "Running Load Test"
    
    log_info "Configuration:"
    log_info "  Duration:       $TEST_DURATION"
    log_info "  Clients:        $TEST_CLIENTS"
    log_info "  RPS per client: $TEST_RPS"
    log_info "  Total RPS:      $((TEST_CLIENTS * TEST_RPS))"
    log_info "  Target:         http://localhost:$PROXY_PORT"
    echo ""
    
    cd "$SCRIPT_DIR"
    "$LOADGEN_BIN" \
        -url "http://localhost:$PROXY_PORT" \
        -clients "$TEST_CLIENTS" \
        -rps "$TEST_RPS" \
        -duration "$TEST_DURATION" \
        -verbose=false
}

show_logs() {
    log_header "System Logs"
    
    echo -e "${YELLOW}=== Proxy Log (last 20 lines) ===${NC}"
    if [ -f "$PID_DIR/proxy.log" ]; then
        tail -20 "$PID_DIR/proxy.log"
    fi
    
    echo ""
    echo -e "${YELLOW}=== Backend Logs ===${NC}"
    for log_file in "$PID_DIR"/backend-*.log; do
        if [ -f "$log_file" ]; then
            echo -e "${YELLOW}$(basename $log_file):${NC}"
            tail -5 "$log_file"
            echo ""
        fi
    done
}

# Main execution
main() {
    log_header "Reverse Proxy Test Suite"
    
    # Check prerequisites
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi
    
    # Execute test workflow
    build_binaries
    start_backends
    wait_for_backends
    start_proxy
    wait_for_proxy
    run_load_test
    
    log_header "Test Summary"
    log_info "Test completed successfully!"
    log_info "Proxy log: $PID_DIR/proxy.log"
    log_info "Backend logs: $PID_DIR/backend-*.log"
    
    echo ""
    log_info "Press Ctrl+C to stop all servers and exit"
    
    # Keep running until interrupted
    sleep infinity
}

# Handle command line arguments
case "${1:-full}" in
    full)
        main
        ;;
    backends-only)
        log_header "Starting Backends Only"
        build_binaries
        start_backends
        wait_for_backends
        log_info "Backends are running. Press Ctrl+C to stop."
        sleep infinity
        ;;
    proxy-only)
        log_header "Starting Proxy Only"
        build_binaries
        start_proxy
        wait_for_proxy
        log_info "Proxy is running. Press Ctrl+C to stop."
        sleep infinity
        ;;
    load-test)
        run_load_test
        ;;
    logs)
        show_logs
        ;;
    clean)
        cleanup
        ;;
    *)
        echo "Usage: $0 {full|backends-only|proxy-only|load-test|logs|clean}"
        echo ""
        echo "Commands:"
        echo "  full            - Run complete test suite (default)"
        echo "  backends-only   - Start only backend servers"
        echo "  proxy-only      - Start only the proxy"
        echo "  load-test       - Run load test (proxy must be running)"
        echo "  logs            - Show system logs"
        echo "  clean           - Stop all processes and cleanup"
        echo ""
        echo "Environment variables:"
        echo "  TEST_DURATION   - Load test duration (default: 30s)"
        echo "  TEST_CLIENTS    - Number of concurrent clients (default: 5)"
        echo "  TEST_RPS        - Requests per second per client (default: 50)"
        echo "  NUM_BACKENDS    - Number of backend servers (default: 3)"
        echo ""
        exit 1
        ;;
esac