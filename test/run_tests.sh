#!/bin/bash

# Traefik Proxy Test Runner
# This script runs all tests and provides a summary

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  Traefik Proxy Test Suite${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Clean build cache
echo -e "${YELLOW}📦 Cleaning build cache...${NC}"
go clean -cache
echo ""

# Run unit tests
echo -e "${BLUE}🧪 Running Unit Tests...${NC}"
echo "-----------------------------------"
cd test
if go test -v; then
    echo ""
    echo -e "${GREEN}✅ Unit tests completed${NC}"
    UNIT_TESTS_PASSED=true
else
    echo ""
    echo -e "${RED}❌ Some unit tests failed${NC}"
    UNIT_TESTS_PASSED=false
fi
cd ..
echo ""

# Check if proxy is running
echo -e "${YELLOW}🔍 Checking if proxy is running on port 8080...${NC}"
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Proxy is running${NC}"
    PROXY_RUNNING=true
else
    echo -e "${YELLOW}⚠️  Proxy is not running on port 8080${NC}"
    echo -e "   Start it with: ${BLUE}go run cmd/main.go${NC}"
    PROXY_RUNNING=false
fi
echo ""

# Run integration tests if proxy is running
if [ "$PROXY_RUNNING" = true ]; then
    echo -e "${BLUE}🌐 Running Integration Tests...${NC}"
    echo "-----------------------------------"
    if go run test/integration/main.go; then
        echo ""
        echo -e "${GREEN}✅ Integration tests completed${NC}"
        INTEGRATION_TESTS_PASSED=true
    else
        echo ""
        echo -e "${RED}❌ Integration tests failed${NC}"
        INTEGRATION_TESTS_PASSED=false
    fi
else
    echo -e "${YELLOW}⏭️  Skipping integration tests (proxy not running)${NC}"
    INTEGRATION_TESTS_PASSED="skipped"
fi
echo ""

# Summary
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  Test Summary${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

if [ "$UNIT_TESTS_PASSED" = true ]; then
    echo -e "Unit Tests:        ${GREEN}✅ PASSED${NC}"
else
    echo -e "Unit Tests:        ${RED}❌ FAILED${NC}"
fi

if [ "$INTEGRATION_TESTS_PASSED" = true ]; then
    echo -e "Integration Tests: ${GREEN}✅ PASSED${NC}"
elif [ "$INTEGRATION_TESTS_PASSED" = "skipped" ]; then
    echo -e "Integration Tests: ${YELLOW}⏭️  SKIPPED${NC}"
else
    echo -e "Integration Tests: ${RED}❌ FAILED${NC}"
fi

echo ""

# Exit with appropriate code
if [ "$UNIT_TESTS_PASSED" = true ] && ([ "$INTEGRATION_TESTS_PASSED" = true ] || [ "$INTEGRATION_TESTS_PASSED" = "skipped" ]); then
    echo -e "${GREEN}🎉 All tests completed successfully!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed. Please review the output above.${NC}"
    exit 1
fi
