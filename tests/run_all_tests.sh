#!/bin/bash

# Traefik Proxy Comprehensive Test Runner
# This script starts the proxy, backend servers, runs all tests, and cleans up

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROXY_PORT=8080
BACKEND_PORTS=(8081 8082 8083 8084)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="${PROJECT_ROOT}/tests"

# PID tracking
PROXY_PID=""
BACKEND_PIDS=()

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}🧹 Cleaning up processes...${NC}"

    # Kill backend servers
    for pid in "${BACKEND_PIDS[@]}"; do
        if [ ! -z "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            echo -e "${YELLOW}  Stopping backend server (PID: $pid)${NC}"
            kill "$pid" 2>/dev/null || true
        fi
    done

    # Kill proxy server
    if [ ! -z "$PROXY_PID" ] && kill -0 "$PROXY_PID" 2>/dev/null; then
        echo -e "${YELLOW}  Stopping proxy server (PID: $PROXY_PID)${NC}"
        kill "$PROXY_PID" 2>/dev/null || true
    fi

    # Give processes time to terminate
    sleep 1

    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Print header
print_header() {
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}  Traefik Proxy - Comprehensive Test Suite${NC}"
    echo -e "${BLUE}================================================================${NC}"
    echo ""
}

# Check if port is in use
check_port() {
    local port=$1
    if lsof -Pi :${port} -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Kill process on port if exists
kill_port() {
    local port=$1
    if check_port $port; then
        echo -e "${YELLOW}  Port $port is in use, killing existing process...${NC}"
        local pid=$(lsof -Pi :${port} -sTCP:LISTEN -t)
        kill $pid 2>/dev/null || true
        sleep 1
    fi
}

# Start backend servers
start_backends() {
    echo -e "${CYAN}🚀 Starting backend servers...${NC}"

    cd "${TESTS_DIR}/backend"

    for port in "${BACKEND_PORTS[@]}"; do
        kill_port $port

        echo -e "${CYAN}  Starting backend on port $port...${NC}"
        go run server.go -port=$port > "/tmp/backend-${port}.log" 2>&1 &
        local pid=$!
        BACKEND_PIDS+=($pid)

        # Wait for backend to start
        sleep 1

        if kill -0 "$pid" 2>/dev/null && check_port $port; then
            echo -e "${GREEN}  ✅ Backend started on port $port (PID: $pid)${NC}"
        else
            echo -e "${RED}  ❌ Failed to start backend on port $port${NC}"
            return 1
        fi
    done

    echo ""
    return 0
}

# Start proxy server
start_proxy() {
    echo -e "${CYAN}🚀 Starting proxy server...${NC}"

    kill_port $PROXY_PORT

    cd "${PROJECT_ROOT}"

    echo -e "${CYAN}  Starting proxy on port $PROXY_PORT...${NC}"
    go run cmd/main.go > /tmp/proxy.log 2>&1 &
    PROXY_PID=$!

    # Wait for proxy to start
    for i in {1..10}; do
        sleep 1
        if check_port $PROXY_PORT; then
            echo -e "${GREEN}  ✅ Proxy started on port $PROXY_PORT (PID: $PROXY_PID)${NC}"
            echo ""
            return 0
        fi
    done

    echo -e "${RED}  ❌ Failed to start proxy on port $PROXY_PORT${NC}"
    echo -e "${RED}  Check /tmp/proxy.log for details${NC}"
    return 1
}

# Run proxy tests
run_proxy_tests() {
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running Proxy Tests${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo ""

    cd "${TESTS_DIR}/proxy"

    if go test -v -timeout=60s; then
        echo ""
        echo -e "${GREEN}✅ Proxy tests PASSED${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}❌ Proxy tests FAILED${NC}"
        return 1
    fi
}

# Run cache tests
run_cache_tests() {
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running Cache Tests${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo ""

    cd "${TESTS_DIR}/cache"

    if go test -v -timeout=60s; then
        echo ""
        echo -e "${GREEN}✅ Cache tests PASSED${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}❌ Cache tests FAILED${NC}"
        return 1
    fi
}

# Print summary
print_summary() {
    local proxy_result=$1
    local cache_result=$2

    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Test Summary${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo ""

    if [ $proxy_result -eq 0 ]; then
        echo -e "  Proxy Tests: ${GREEN}✅ PASSED${NC}"
    else
        echo -e "  Proxy Tests: ${RED}❌ FAILED${NC}"
    fi

    if [ $cache_result -eq 0 ]; then
        echo -e "  Cache Tests: ${GREEN}✅ PASSED${NC}"
    else
        echo -e "  Cache Tests: ${RED}❌ FAILED${NC}"
    fi

    echo ""
    echo -e "${CYAN}📊 Backend Servers: ${#BACKEND_PORTS[@]} running${NC}"
    echo -e "${CYAN}📊 Total Requests: Hundreds per test suite${NC}"
    echo ""

    if [ $proxy_result -eq 0 ] && [ $cache_result -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed successfully!${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}💥 Some tests failed. Check output above.${NC}"
        echo ""
        return 1
    fi
}

# Main execution
main() {
    print_header

    # Start backend servers
    if ! start_backends; then
        echo -e "${RED}Failed to start backend servers${NC}"
        exit 1
    fi

    # Start proxy server
    if ! start_proxy; then
        echo -e "${RED}Failed to start proxy server${NC}"
        exit 1
    fi

    # Give everything a moment to stabilize
    echo -e "${CYAN}⏳ Waiting for services to stabilize...${NC}"
    sleep 2
    echo ""

    # Run tests
    PROXY_RESULT=0
    CACHE_RESULT=0

    run_proxy_tests || PROXY_RESULT=$?
    run_cache_tests || CACHE_RESULT=$?

    # Print summary
    print_summary $PROXY_RESULT $CACHE_RESULT

    # Return overall result
    if [ $PROXY_RESULT -eq 0 ] && [ $CACHE_RESULT -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main
