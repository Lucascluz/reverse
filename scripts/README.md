# Reverse Proxy Testing Guide

This directory contains scripts and tools for load testing and validating the reverse proxy.

## Quick Start

### Run Complete Test Suite (Recommended)

```bash
cd scripts
chmod +x *.sh
./test.sh full
```

This will:
1. Build all components
2. Start 3 dummy backend servers
3. Start the reverse proxy
4. Wait for health checks to pass
5. Run a 30-second load test with 5 clients at 50 RPS each
6. Display comprehensive results

### Custom Configuration

```bash
# Run 2-minute test with 20 clients at 100 RPS each
TEST_DURATION=120s TEST_CLIENTS=20 TEST_RPS=100 ./test.sh full

# Start only backends (useful for manual testing)
./test.sh backends-only

# Start only proxy (backends already running)
./test.sh proxy-only

# Run load test against running proxy
./test.sh load-test

# View system logs
./test.sh logs

# Clean up all processes
./test.sh clean
```

## Components

### 1. Dummy Server (`cmd/dummy-server/main.go`)

A configurable HTTP server that simulates backend services.

**Features:**
- Health check endpoint (`/health`)
- Echo endpoint that returns request details
- Configurable latency and error rates
- Request counting and statistics
- Graceful shutdown

**Usage:**

```bash
# Start backend-1 on port 8081
go run cmd/dummy-server/main.go -port 8081 -name backend-1

# Start backend with 50ms latency
go run cmd/dummy-server/main.go -port 8081 -name backend-1 -latency 50

# Start backend with 5% error rate
go run cmd/dummy-server/main.go -port 8081 -name backend-1 -error-rate 0.05

# Start backend that becomes healthy after 10 seconds
go run cmd/dummy-server/main.go -port 8081 -name backend-1 -healthy-delay 10s
```

**Endpoints:**
- `GET /health` - Health check (returns JSON with status)
- `GET /` - Echo endpoint (returns request details)

**Response Example:**
```json
{
  "message": "Echo from backend-1",
  "method": "GET",
  "path": "/",
  "headers": {
    "User-Agent": "curl/7.68.0",
    "Accept": "*/*"
  },
  "remote_addr": "127.0.0.1:54321",
  "latency_ms": 10,
  "timestamp": "2024-01-15T10:30:45Z"
}
```

### 2. Load Generator (`cmd/loadgen/main.go`)

A sophisticated HTTP load testing tool that generates realistic traffic patterns.

**Features:**
- Configurable number of concurrent clients
- Per-client request rate control
- Realistic randomized HTTP headers
- Request tracking and statistics
- Comprehensive performance metrics

**Random Headers Generated:**
- User-Agent (5 different variants)
- Accept (JSON, HTML, wildcard)
- Accept-Language (multiple languages)
- Accept-Encoding (gzip, deflate, etc.)
- Cache-Control (various directives)
- X-Request-ID (unique identifiers)
- X-Forwarded-For (random IPs)
- X-Real-IP (random IPs)
- Authorization (Bearer tokens)
- Custom headers

**Usage:**

```bash
# Basic load test - 10 clients, 100 RPS each, 30 seconds
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=10 \
  -rps=100 \
  -duration=30s

# High-stress test - 50 clients, 500 RPS each
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=50 \
  -rps=500 \
  -duration=60s

# Gentle test with verbose output
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=2 \
  -rps=10 \
  -duration=30s \
  -verbose=true
```

**Output Example:**
```
============================================================
LOAD TEST RESULTS
============================================================
Total Requests:      150000
Successful:          149,850 (99.90%)
Errors:              150 (0.10%)

Latency:
  Average:           45ms
  Min:               12ms
  Max:               312ms

Throughput:
  Expected RPS:      1000 req/s
  Actual RPS:        998 req/s
  Efficiency:        99.80%
============================================================
```

## Test Scenarios

### Scenario 1: Basic Functionality Test
**Goal:** Verify proxy routing and health checks work

```bash
# Start 2 backends with low latency
NUM_BACKENDS=2 ./test.sh backends-only

# In another terminal
./test.sh proxy-only

# In another terminal
go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=2 -rps=10 -duration=20s
```

### Scenario 2: Load Distribution Test
**Goal:** Verify round-robin load balancing

```bash
# Monitor backend request counts
watch -n 1 'tail -5 .pids/backend-*.log'

# Run load test
TEST_CLIENTS=10 TEST_RPS=50 ./test.sh load-test

# Verify each backend received ~equal requests
```

### Scenario 3: High-Load Stress Test
**Goal:** Test proxy behavior under high load

```bash
# Run aggressive load test
TEST_DURATION=120s TEST_CLIENTS=50 TEST_RPS=200 ./test.sh full
```

### Scenario 4: Slow Backend Handling
**Goal:** Verify proxy handles slow backends gracefully

```bash
# Start backends with high latency
go run cmd/dummy-server/main.go -port 8081 -name backend-1 -latency 500
go run cmd/dummy-server/main.go -port 8082 -name backend-2 -latency 500
go run cmd/dummy-server/main.go -port 8083 -name backend-3 -latency 500

# Start proxy and run load test
go run ../cmd/main.go &
sleep 3
go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=20 -rps=50 -duration=30s
```

### Scenario 5: Backend Failure Handling
**Goal:** Test health check and failover behavior

```bash
# Start backends and proxy
./test.sh backends-only &
sleep 3
go run ../cmd/main.go &
sleep 3

# Run initial load test
go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=5 -rps=20 -duration=10s

# Kill one backend (notice system continues working)
kill $(cat .pids/backend-1.pid)

# Continue load test
go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=5 -rps=20 -duration=10s

# Restart backend (notice system rebalances)
go run cmd/dummy-server/main.go -port 8081 -name backend-1 &
sleep 3

# Final load test
go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=5 -rps=20 -duration=10s
```

### Scenario 6: Cache Effectiveness Test
**Goal:** Measure cache hit rates and performance impact

```bash
# With cache enabled (default)
TEST_CLIENTS=10 TEST_RPS=100 ./test.sh load-test

# With cache disabled
# Edit config.yaml: cache.disabled = true
go run ../cmd/main.go &
TEST_CLIENTS=10 TEST_RPS=100 go run cmd/loadgen/main.go -url=http://localhost:8080 -duration=30s
```

## Performance Benchmarks

### Expected Results (Hardware Dependent)

On a modern machine with 4 CPU cores:

**Light Load (5 clients, 50 RPS):**
- Success Rate: >99.9%
- Average Latency: 10-30ms
- Min/Max Latency: 5-100ms

**Medium Load (20 clients, 100 RPS):**
- Success Rate: >99%
- Average Latency: 20-50ms
- Min/Max Latency: 10-200ms

**High Load (50 clients, 500 RPS):**
- Success Rate: >95%
- Average Latency: 50-200ms
- Min/Max Latency: 20-1000ms

## Monitoring During Tests

### In Another Terminal

```bash
# Watch health check status
watch -n 1 'curl -s http://localhost:8085/readyz | jq'

# Monitor memory usage
watch -n 1 'ps aux | grep -E "reverse|dummy-server|loadgen" | grep -v grep'

# View live logs
tail -f .pids/proxy.log

# Monitor request distribution
watch -n 1 'grep -c "message" .pids/backend-*.log'
```

### Health Check Endpoints

The proxy exposes health checks on port 8085:

```bash
# Liveness probe (always returns 200 if proxy is running)
curl http://localhost:8085/healthz

# Readiness probe (returns 200 only if backends are healthy)
curl http://localhost:8085/readyz

# Backend health check
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
```

## Troubleshooting

### Port Already in Use

```bash
# Kill processes on ports 8080-8085
lsof -ti :8080-8085 | xargs kill -9

# Or run with different ports (requires config changes)
```

### Proxy Won't Start

```bash
# Check config validity
go run ../cmd/main.go

# Check backend connectivity
curl -v http://localhost:8081/health
curl -v http://localhost:8082/health
curl -v http://localhost:8083/health
```

### Load Test Fails

```bash
# Verify proxy is running
curl http://localhost:8080/health

# Verify backends are healthy
curl http://localhost:8080/readyz

# Check network connectivity
netstat -an | grep 8080

# View detailed logs
./test.sh logs
```

### High Error Rate

```bash
# Check system resources
top -b -n 1 | head -20

# Increase file descriptor limit (Linux)
ulimit -n 65536

# Reduce load or increase backend capacity
TEST_CLIENTS=5 TEST_RPS=20 ./test.sh load-test
```

## Configuration

All scripts respect environment variables for easy customization:

```bash
export TEST_DURATION=120s      # Default: 30s
export TEST_CLIENTS=20         # Default: 5
export TEST_RPS=100            # Default: 50
export NUM_BACKENDS=5          # Default: 3
export VERBOSE=true            # Default: false
```

## File Structure

```
scripts/
├── README.md                           # This file
├── test.sh                             # Main orchestration script
├── start-backends.sh                   # Start dummy servers only
├── run-load-test.sh                    # Run load test only
├── cmd/
│   ├── dummy-server/
│   │   └── main.go                     # Backend server implementation
│   └── loadgen/
│       └── main.go                     # Load generator implementation
└── .pids/                              # Process IDs and logs (created at runtime)
    ├── backend-1.pid/.log
    ├── backend-2.pid/.log
    ├── backend-3.pid/.log
    └── proxy.pid/.log
```

## Advanced Usage

### Custom Backend Configurations

```bash
# Backend with high latency (simulate slow service)
go run cmd/dummy-server/main.go -port 8081 -name slow-backend -latency 500

# Backend with errors (simulate flaky service)
go run cmd/dummy-server/main.go -port 8081 -name flaky-backend -error-rate 0.1

# Backend that starts unhealthy
go run cmd/dummy-server/main.go -port 8081 -name delayed-backend -healthy-delay 30s
```

### Capturing Detailed Metrics

```bash
# Run with verbose output and save to file
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=10 \
  -rps=100 \
  -duration=60s \
  -verbose=true > test_results.log 2>&1

# Analyze results
grep "Average" test_results.log
grep "Success Rate" test_results.log
```

## Best Practices

1. **Always start with small loads** - Verify basic functionality first
2. **Monitor system resources** - Watch CPU, memory, and file descriptors
3. **Incremental load increase** - Gradually increase clients/RPS to find limits
4. **Test failure scenarios** - Kill backends and verify graceful handling
5. **Review logs** - Check proxy and backend logs for errors
6. **Clean up properly** - Use `./test.sh clean` after tests
7. **Isolate variables** - Change one parameter at a time
8. **Repeat tests** - Run tests multiple times for consistent results

## Contributing

To add new test scenarios or improve the test suite:

1. Edit `test.sh` to add new commands
2. Create helper functions for common tasks
3. Update this README with usage instructions
4. Test thoroughly before committing

## References

- Reverse Proxy Main: `../cmd/main.go`
- Proxy Config: `../config.yaml`
- Load Balancer: `../internal/loadbalancer/`
- Health Checker: `../internal/observability/healthchecker.go`
