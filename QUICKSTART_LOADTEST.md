# Quick Start: Load Testing the Reverse Proxy

This guide will get you running load tests in under 5 minutes.

## Prerequisites

- Go 1.16+
- curl (for health checks)
- Bash

## Option 1: Fully Automated (Recommended)

The easiest way to run everything at once:

```bash
cd reverse/scripts
chmod +x test.sh
./test.sh full
```

This single command will:
1. Build the proxy and all test tools
2. Start 3 dummy backend servers (ports 8081-8083)
3. Start the reverse proxy (port 8080)
4. Wait for health checks to pass
5. Run a 30-second load test with 5 clients at 50 RPS
6. Display comprehensive results
7. Keep running for manual testing (Ctrl+C to stop)

Expected output:
```
==================================================
LOAD TEST RESULTS
==================================================
Total Requests:      7500
Successful:          7500 (100.00%)
Errors:              0 (0.00%)

Latency:
  Average:           25ms
  Min:               5ms
  Max:               145ms

Throughput:
  Expected RPS:      250 req/s
  Actual RPS:        250 req/s
  Efficiency:        100.00%
==================================================
```

## Option 2: Manual Control (Three Terminals)

For more control, run components separately:

### Terminal 1: Start Backend Servers
```bash
cd reverse/scripts
./test.sh backends-only
```

Output:
```
[INFO] Starting Backend Servers
[INFO] Starting backend-1 on port 8081...
[INFO] Starting backend-2 on port 8082...
[INFO] Starting backend-3 on port 8083...
[INFO] All backends started successfully
```

### Terminal 2: Start Reverse Proxy
```bash
cd reverse
go run ./cmd/main.go
```

Output:
```
[reverse] Proxy server listening on :8080
[reverse] Probe server listening on :8085
[reverse] application initialized successfully
```

### Terminal 3: Run Load Test
```bash
cd reverse/scripts
./test.sh load-test
```

## Customizing the Load Test

Change the test parameters using environment variables:

### Light Test (for quick validation)
```bash
TEST_DURATION=10s TEST_CLIENTS=2 TEST_RPS=10 ./test.sh full
```

### Medium Test (realistic scenario)
```bash
TEST_DURATION=60s TEST_CLIENTS=10 TEST_RPS=100 ./test.sh full
```

### Heavy Test (stress testing)
```bash
TEST_DURATION=120s TEST_CLIENTS=50 TEST_RPS=500 ./test.sh full
```

### Custom Test
```bash
TEST_DURATION=45s TEST_CLIENTS=15 TEST_RPS=200 NUM_BACKENDS=5 ./test.sh full
```

## Monitoring During Tests

While load test is running, you can check system health:

```bash
# Check if proxy is alive (should return 200)
curl http://localhost:8080/healthz

# Check if backends are healthy (should return 200)
curl http://localhost:8080/readyz

# Check individual backends
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
```

## Understanding the Results

The load test displays key metrics:

| Metric | Meaning |
|--------|---------|
| Total Requests | How many HTTP requests were made |
| Successful | Requests with 2xx/3xx status codes |
| Errors | Requests that failed (4xx/5xx or timeout) |
| Average Latency | Mean response time |
| Min/Max Latency | Best/worst response times |
| Expected RPS | The target requests per second |
| Actual RPS | What was actually achieved |
| Efficiency | Percentage of target rate achieved |

**Good results:**
- Success rate > 99%
- Efficiency > 95%
- Latency < 100ms

## Troubleshooting

### Port Already in Use
```bash
# Kill processes on proxy/backend ports
lsof -ti :8080-8085 | xargs kill -9

# Then retry
./test.sh full
```

### Proxy Won't Start
```bash
# Check backends are running
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Check proxy logs
tail -20 .pids/proxy.log
```

### Load Test Fails
```bash
# Verify proxy is responding
curl -v http://localhost:8080/health

# View load generator output with details
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=2 \
  -rps=10 \
  -duration=10s \
  -verbose=true
```

### High Error Rate
- Reduce load: `TEST_CLIENTS=5 TEST_RPS=20`
- Check backend health: `curl http://localhost:8080/readyz`
- View logs: `./test.sh logs`

## Next Steps

1. **Try different load profiles** - See how the proxy performs under various loads
2. **Monitor resource usage** - Watch CPU/memory with `top`
3. **Test failure scenarios** - Kill a backend and watch recovery
4. **Review the code** - See how the test tools work in `cmd/dummy-server/` and `cmd/loadgen/`
5. **Customize the proxy** - Adjust settings in `config.yaml`

## Environment Variables

```bash
TEST_DURATION    # How long to run the load test (default: 30s)
TEST_CLIENTS     # Number of concurrent clients (default: 5)
TEST_RPS         # Requests per second per client (default: 50)
NUM_BACKENDS     # Number of dummy backends (default: 3)
```

Example:
```bash
export TEST_DURATION=120s
export TEST_CLIENTS=20
export TEST_RPS=100
./test.sh full
```

## Common Commands

```bash
# Start everything
./test.sh full

# Stop everything
./test.sh clean

# Start only backends
./test.sh backends-only

# Start only proxy (backends must be running)
./test.sh proxy-only

# Run load test (proxy must be running)
./test.sh load-test

# View logs
./test.sh logs

# Manual load test (advanced)
go run cmd/loadgen/main.go \
  -url=http://localhost:8080 \
  -clients=10 \
  -rps=100 \
  -duration=30s
```

## What's Being Tested

The load test creates realistic traffic patterns:

- **Random headers**: User-Agent, Accept, Cache-Control, Authorization, etc.
- **Multiple paths**: Tests routing to different endpoints
- **Various response times**: Backends have different latencies
- **Header variations**: Tests proxy's ability to handle diverse clients
- **Concurrent requests**: Multiple clients making requests simultaneously

This simulates real-world usage and ensures the proxy can handle:
- ✓ Load balancing across multiple backends
- ✓ Health checking and failover
- ✓ Request routing with proper headers
- ✓ Rate limiting per IP
- ✓ Caching (if enabled)
- ✓ Concurrent request handling

## Performance Expectations

On a modern machine with 4 CPU cores:

- **Light load** (5 clients, 50 RPS): <30ms latency, 99.9% success
- **Medium load** (20 clients, 100 RPS): <50ms latency, 99% success
- **Heavy load** (50 clients, 500 RPS): <200ms latency, 95% success

Actual results depend on your hardware and backend configuration.

## Getting Help

For detailed information about:
- **Load testing**: See `scripts/README.md`
- **Proxy configuration**: See `config.yaml`
- **Test components**: See `scripts/cmd/dummy-server/main.go` and `scripts/cmd/loadgen/main.go`
- **Proxy architecture**: See `internal/` directory

## Quick Checklist

- [ ] Go is installed (`go version`)
- [ ] curl is available (`curl --version`)
- [ ] Proxy builds: `cd reverse && go build ./cmd/main.go`
- [ ] Run quick test: `cd scripts && ./test.sh full`
- [ ] Verify success rate > 99%
- [ ] Monitor health endpoints work
- [ ] Review load test results

That's it! You're ready to load test the reverse proxy.