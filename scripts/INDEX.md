# Load Testing Scripts - Index & Quick Reference

## 📚 Documentation Files

### Quick Start (Read First!)
- **`QUICKSTART_LOADTEST.md`** (5 minutes)
  - Fastest way to get started
  - One-command setup
  - Troubleshooting quick fixes

### Comprehensive Guides
- **`scripts/README.md`** (Detailed)
  - Component documentation
  - Test scenarios
  - Performance benchmarks
  - Advanced usage

- **`LOADTESTING.md`** (Complete Reference)
  - Architecture overview
  - Configuration details
  - All test scenarios
  - Integration examples

## 🚀 Quick Start Commands

```bash
# One-command test (recommended)
cd reverse/scripts
chmod +x test.sh
./test.sh full

# Custom parameters
TEST_DURATION=120s TEST_CLIENTS=20 TEST_RPS=100 ./test.sh full

# Individual components
./test.sh backends-only    # Start only backend servers
./test.sh proxy-only       # Start only reverse proxy
./test.sh load-test        # Run load test only
./test.sh logs             # View system logs
./test.sh clean            # Stop everything
```

## 📂 File Structure

```
scripts/
├── INDEX.md                              # ← You are here
├── README.md                             # Detailed documentation
├── test.sh                               # Main test orchestration script
├── start-backends.sh                     # Helper: start backends only
├── run-load-test.sh                      # Helper: run load test only
├── cmd/
│   ├── dummy-server/main.go             # Simulates backend services
│   └── loadgen/main.go                  # HTTP load generator
└── .pids/                               # Runtime files (auto-created)
    ├── proxy.log / proxy.pid
    ├── backend-1.log / backend-1.pid
    ├── backend-2.log / backend-2.pid
    └── backend-3.log / backend-3.pid
```

## 🛠️ Components

### Dummy Server
**What:** Simulates backend HTTP services
**Command:** `go run cmd/dummy-server/main.go -port 8081 -name backend-1`
**Flags:**
- `-port` : Server port (default: 8081)
- `-name` : Server name (default: "backend")
- `-latency` : Response latency in ms (default: 10)
- `-error-rate` : Fraction of requests that fail (default: 0.0)
- `-healthy-delay` : Delay before marking healthy (default: 0)

**Endpoints:**
- `GET /health` - Health check (JSON response)
- `GET /` - Echo endpoint (returns request details)

### Load Generator
**What:** Generates HTTP traffic with random headers
**Command:** `go run cmd/loadgen/main.go -url=http://localhost:8080 -clients=5 -rps=50`
**Flags:**
- `-url` : Target proxy URL (default: http://localhost:8080)
- `-clients` : Concurrent clients (default: 10)
- `-rps` : Requests per second per client (default: 100)
- `-duration` : Test duration (default: 30s)
- `-timeout` : Request timeout (default: 30s)
- `-verbose` : Verbose output (default: false)

**Random Headers Generated:**
- User-Agent (5 variants)
- Accept (JSON, HTML, wildcard)
- Accept-Language (multiple languages)
- Accept-Encoding (gzip, deflate, etc.)
- Cache-Control (various directives)
- X-Request-ID (unique per request)
- X-Forwarded-For (random IPs)
- X-Real-IP (random IPs)
- Authorization (Bearer tokens)
- Custom headers

### Test Orchestration
**What:** Automates entire testing workflow
**Command:** `./test.sh [command]`
**Subcommands:**
- `full` - Complete test suite (build, backends, proxy, load test)
- `backends-only` - Start only dummy backends
- `proxy-only` - Start only reverse proxy
- `load-test` - Run load test (assumes proxy is running)
- `logs` - Display system logs
- `clean` - Stop all processes

**Environment Variables:**
- `TEST_DURATION` - Load test duration (default: 30s)
- `TEST_CLIENTS` - Number of concurrent clients (default: 5)
- `TEST_RPS` - Requests per second per client (default: 50)
- `NUM_BACKENDS` - Number of backend servers (default: 3)

## 📊 Typical Workflow

### Phase 1: Build
```bash
cd reverse/scripts
chmod +x test.sh
```

### Phase 2: Start Components
```bash
# Option A: Fully automated
./test.sh full

# Option B: Manual control
./test.sh backends-only &      # Terminal 1
go run ../cmd/main.go &        # Terminal 2
./test.sh load-test            # Terminal 3
```

### Phase 3: Monitor
```bash
# Health checks
curl http://localhost:8080/healthz    # Liveness
curl http://localhost:8080/readyz     # Readiness

# Individual backends
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Logs
./test.sh logs
tail -f .pids/proxy.log
tail -f .pids/backend-*.log
```

### Phase 4: Analyze Results
- Check success rate (target: >99%)
- Review latency metrics (p50, p99, max)
- Verify throughput efficiency (target: >95%)
- Check for errors in logs

## 🧪 Common Test Scenarios

### Light Load (Verification)
```bash
TEST_DURATION=30s TEST_CLIENTS=2 TEST_RPS=10 ./test.sh full
```
**Use Case:** Quick validation that everything works
**Expected:** Success rate >99%, latency <50ms

### Medium Load (Normal)
```bash
TEST_DURATION=60s TEST_CLIENTS=10 TEST_RPS=100 ./test.sh full
```
**Use Case:** Typical load testing
**Expected:** Success rate >99%, latency <100ms

### Heavy Load (Stress)
```bash
TEST_DURATION=120s TEST_CLIENTS=50 TEST_RPS=500 ./test.sh full
```
**Use Case:** Find system limits
**Expected:** Success rate >95%, latency <500ms

### Slow Backends
```bash
go run cmd/dummy-server/main.go -port 8081 -latency 500 &
go run cmd/dummy-server/main.go -port 8082 -latency 500 &
go run cmd/dummy-server/main.go -port 8083 -latency 500 &
sleep 2
go run ../cmd/main.go &
sleep 2
./test.sh load-test
```

### Failure Recovery
```bash
./test.sh backends-only &
sleep 2
go run ../cmd/main.go &
sleep 2
./test.sh load-test &
# In another terminal: kill a backend
# Observe system continues working
# Restart backend and observe recovery
```

## 🔍 Port Mapping

| Component | Port | Purpose |
|-----------|------|---------|
| Reverse Proxy | 8080 | Main proxy traffic |
| Proxy Probes | 8085 | Health checks |
| Backend 1 | 8081 | Test backend |
| Backend 2 | 8082 | Test backend |
| Backend 3 | 8083 | Test backend |

## 📈 Understanding Results

```
Total Requests:      150000
Successful:          149850 (99.90%)
Errors:              150 (0.10%)

Latency:
  Average:           45ms
  Min:               12ms
  Max:               312ms

Throughput:
  Expected RPS:      1000 req/s
  Actual RPS:        998 req/s
  Efficiency:        99.80%
```

**Metrics Explained:**
- **Total Requests** - How many HTTP requests were made
- **Successful** - Requests with 2xx/3xx status codes
- **Errors** - Requests that failed (4xx/5xx or timeout)
- **Average Latency** - Mean response time
- **Min/Max Latency** - Best/worst response times
- **Expected RPS** - Target requests per second
- **Actual RPS** - What was actually achieved
- **Efficiency** - How well target rate was met (%)

**Good Results:**
- Success rate > 99%
- Efficiency > 95%
- Latency < 100ms

## 🐛 Troubleshooting

### Port Already in Use
```bash
lsof -ti :8080-8085 | xargs kill -9
```

### Proxy Won't Start
```bash
# Verify backends are running
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Check logs
tail -50 .pids/proxy.log
```

### High Error Rate
```bash
# Reduce load
TEST_CLIENTS=2 TEST_RPS=10 ./test.sh load-test

# Check readiness
curl http://localhost:8080/readyz

# View logs
./test.sh logs
```

### Process Cleanup
```bash
# Kill everything
./test.sh clean

# Or manually
lsof -ti :8080-8085 | xargs kill -9
ps aux | grep -E "(dummy-server|loadgen|reverse)" | awk '{print $2}' | xargs kill -9
```

## 📋 Checklist for First Run

- [ ] Read `QUICKSTART_LOADTEST.md` (5 mins)
- [ ] Go is installed (`go version`)
- [ ] In `reverse/scripts` directory
- [ ] Run `chmod +x test.sh`
- [ ] Run `./test.sh full`
- [ ] Verify success rate > 99%
- [ ] Check latency < 100ms
- [ ] Review logs with `./test.sh logs`

## 🎯 Next Steps

1. **Try light test first** - Verify basic functionality
2. **Review documentation** - Read relevant `.md` files
3. **Experiment with loads** - Try different parameters
4. **Test scenarios** - Kill backends, watch recovery
5. **Monitor systems** - Watch logs and metrics
6. **Customize** - Adjust proxy config for your needs

## 📖 Documentation Map

```
QUICKSTART_LOADTEST.md      ← Start here (5 min read)
        ↓
scripts/README.md           ← Component details
        ↓
LOADTESTING.md             ← Complete reference
        ↓
config.yaml                ← Proxy configuration
```

## 🔗 Related Files

- **Proxy Configuration:** `../config.yaml`
- **Proxy Source:** `../cmd/main.go`
- **Scan Report:** `../SCAN_REPORT.md`
- **Architecture:** `../internal/`

## 💡 Tips

- Use `watch` for real-time monitoring:
  ```bash
  watch -n 1 'curl -s http://localhost:8080/readyz'
  ```

- Capture results to file:
  ```bash
  ./test.sh load-test > test_results.txt 2>&1
  ```

- Run tests in background:
  ```bash
  nohup ./test.sh full > test.log 2>&1 &
  ```

- Check system resources during test:
  ```bash
  top -b -n 1 | head -20
  ```

## ❓ FAQ

**Q: What's the difference between `/healthz` and `/readyz`?**
A: `/healthz` checks if proxy is running. `/readyz` checks if backends are healthy.

**Q: How do I test with different backend latencies?**
A: Use the `-latency` flag: `go run cmd/dummy-server/main.go -latency 500`

**Q: Can I run tests with more than 3 backends?**
A: Yes! Use `NUM_BACKENDS=5 ./test.sh full`

**Q: What headers does the load generator use?**
A: User-Agent, Accept, Cache-Control, Authorization, X-Request-ID, and more (randomized each request).

**Q: How do I simulate backend failures?**
A: Kill the backend process: `kill $(cat .pids/backend-1.pid)`

**Q: Can I use this with CI/CD?**
A: Yes! See `LOADTESTING.md` for CI/CD examples.

---

**Last Updated:** 2024
**Status:** Ready to use
**Tested On:** Linux, macOS, Windows (WSL)