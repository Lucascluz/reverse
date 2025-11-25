#!/bin/bash

# Quick Test Runner - Run specific test suites without full setup
# Usage: ./run_quick_test.sh [proxy|cache|all]

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="${PROJECT_ROOT}/tests"

print_usage() {
    echo "Usage: $0 [proxy|cache|all]"
    echo ""
    echo "Options:"
    echo "  proxy  - Run only proxy tests"
    echo "  cache  - Run only cache tests"
    echo "  all    - Run all tests (default)"
    echo ""
    echo "Note: This script assumes proxy and backends are already running."
    echo "      Use run_all_tests.sh for full automated testing."
}

check_services() {
    echo -e "${BLUE}🔍 Checking services...${NC}"

    # Check proxy
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Proxy is running on port 8080${NC}"
    else
        echo -e "${RED}  ❌ Proxy is NOT running on port 8080${NC}"
        echo -e "${YELLOW}     Start with: go run cmd/main.go${NC}"
        return 1
    fi

    # Check backends
    local backend_count=0
    for port in 8081 8082 8083 8084; do
        if lsof -Pi :${port} -sTCP:LISTEN -t >/dev/null 2>&1; then
            backend_count=$((backend_count + 1))
        fi
    done

    if [ $backend_count -gt 0 ]; then
        echo -e "${GREEN}  ✅ $backend_count backend server(s) running${NC}"
    else
        echo -e "${RED}  ❌ No backend servers running${NC}"
        echo -e "${YELLOW}     Start with: go run tests/backend/server.go -port=8081${NC}"
        return 1
    fi

    echo ""
    return 0
}

run_proxy_tests() {
    echo -e "${BLUE}════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running Proxy Tests${NC}"
    echo -e "${BLUE}════════════════════════════════════════${NC}"
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

run_cache_tests() {
    echo -e "${BLUE}════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running Cache Tests${NC}"
    echo -e "${BLUE}════════════════════════════════════════${NC}"
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

main() {
    local test_type="${1:-all}"

    if [ "$test_type" == "-h" ] || [ "$test_type" == "--help" ]; then
        print_usage
        exit 0
    fi

    echo -e "${BLUE}════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Quick Test Runner${NC}"
    echo -e "${BLUE}════════════════════════════════════════${NC}"
    echo ""

    # Check services
    if ! check_services; then
        echo -e "${RED}Services not ready. Exiting.${NC}"
        exit 1
    fi

    # Run tests based on argument
    case "$test_type" in
        proxy)
            run_proxy_tests
            exit $?
            ;;
        cache)
            run_cache_tests
            exit $?
            ;;
        all)
            PROXY_RESULT=0
            CACHE_RESULT=0

            run_proxy_tests || PROXY_RESULT=$?
            echo ""
            run_cache_tests || CACHE_RESULT=$?

            echo ""
            echo -e "${BLUE}════════════════════════════════════════${NC}"
            echo -e "${BLUE}  Summary${NC}"
            echo -e "${BLUE}════════════════════════════════════════${NC}"

            if [ $PROXY_RESULT -eq 0 ]; then
                echo -e "  Proxy Tests: ${GREEN}✅ PASSED${NC}"
            else
                echo -e "  Proxy Tests: ${RED}❌ FAILED${NC}"
            fi

            if [ $CACHE_RESULT -eq 0 ]; then
                echo -e "  Cache Tests: ${GREEN}✅ PASSED${NC}"
            else
                echo -e "  Cache Tests: ${RED}❌ FAILED${NC}"
            fi

            echo ""

            if [ $PROXY_RESULT -eq 0 ] && [ $CACHE_RESULT -eq 0 ]; then
                echo -e "${GREEN}🎉 All tests passed!${NC}"
                exit 0
            else
                echo -e "${RED}💥 Some tests failed${NC}"
                exit 1
            fi
            ;;
        *)
            echo -e "${RED}Invalid option: $test_type${NC}"
            echo ""
            print_usage
            exit 1
            ;;
    esac
}

main "$@"
