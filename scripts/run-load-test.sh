#!/bin/bash

# Script to run load tests against the reverse proxy
# Usage: ./run-load-test.sh [duration] [clients] [rps] [endpoint]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOADGEN_BIN="$SCRIPT_DIR/cmd/loadgen/main.go"

# Default parameters
DURATION=${1:-30s}
CLIENTS=${2:-10}
RPS=${3:-100}
ENDPOINT=${4:-/echo}
PROXY_URL=${PROXY_URL:-http://localhost:8080}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}   Reverse Proxy Load Test${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Proxy URL:        $PROXY_URL"
echo "  Duration:         $DURATION"
echo "  Clients:          $CLIENTS"
echo "  RPS per client:   $RPS"
echo "  Total RPS:        $((CLIENTS * RPS))"
echo "  Endpoint:         $ENDPOINT"
echo ""

# Check if proxy is reachable
echo -e "${YELLOW}Checking proxy connectivity...${NC}"
if ! curl -s -f "$PROXY_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot reach proxy at $PROXY_URL${NC}"
    echo "Make sure the proxy is running: go run ./cmd/main.go"
    exit 1
fi
echo -e "${GREEN}✓ Proxy is reachable${NC}"
echo ""

# Check if backends are healthy
echo -e "${YELLOW}Checking backend health...${NC}"
if ! curl -s -f "$PROXY_URL/readyz" > /dev/null 2>&1; then
    echo -e "${RED}Warning: Proxy readiness check failed${NC}"
    echo "Backends might not be healthy yet"
fi
echo -e "${GREEN}✓ Ready to start load test${NC}"
echo ""

# Run the load generator
echo -e "${YELLOW}Starting load generator...${NC}"
echo -e "${BLUE}=====================================${NC}"

go run "$LOADGEN_BIN" \
    -url "$PROXY_URL" \
    -clients "$CLIENTS" \
    -rps "$RPS" \
    -duration "$DURATION" \
    -verbose=false

echo ""
echo -e "${BLUE}=====================================${NC}"
echo -e "${GREEN}Load test completed!${NC}"
echo -e "${BLUE}=====================================${NC}"