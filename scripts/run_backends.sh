#!/usr/bin/env bash
# Run N instances of tests/backend/server.go on incremental ports.
# Usage: ./scripts/run_backends.sh N [start_port]
# Example: ./scripts/run_backends.sh 5        # starts 5 instances on ports 8081-8085
#          ./scripts/run_backends.sh 3 9000  # starts on 9000-9002

set -euo pipefail

PROG="tests/backend/server.go"
BIN_DIR="bin"
BIN_PATH="${BIN_DIR}/backend_server"
LOG_DIR="logs"

if ! command -v go >/dev/null 2>&1; then
  echo "go toolchain not found in PATH. Please install Go." >&2
  exit 1
fi

if [ ! -f "${PROG}" ]; then
  echo "Go program not found at ${PROG}" >&2
  exit 1
fi

if [ $# -lt 1 ]; then
  echo "Usage: $0 N [start_port]" >&2
  exit 2
fi

N="$1"
if ! [[ "${N}" =~ ^[0-9]+$ ]] || [ "${N}" -le 0 ]; then
  echo "N must be a positive integer" >&2
  exit 2
fi

START_PORT="${2:-8081}"
if ! [[ "${START_PORT}" =~ ^[0-9]+$ ]]; then
  echo "start_port must be an integer" >&2
  exit 2
fi

mkdir -p "${BIN_DIR}" "${LOG_DIR}"

echo "Building ${PROG} -> ${BIN_PATH} ..."
go build -o "${BIN_PATH}" "${PROG}"

pids=()
ports=()

cleanup() {
  echo
  echo "Stopping ${#pids[@]} backend(s)..."
  for pid in "${pids[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "  Killing PID $pid"
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  # wait for them to exit
  wait || true
  exit 0
}

trap cleanup INT TERM

echo "Starting ${N} backend instance(s) starting at port ${START_PORT}..."

for i in $(seq 0 $((N - 1))); do
  port=$((START_PORT + i))
  log_file="${LOG_DIR}/backend-${port}.log"
  echo "  -> Port ${port} (log: ${log_file})"
  # Start server in background, redirecting stdout/stderr to log file
  "${BIN_PATH}" -port "${port}" > "${log_file}" 2>&1 &
  pid=$!
  pids+=("${pid}")
  ports+=("${port}")
  # small stagger to avoid race on identical logs / console flooding
  sleep 0.05
done

echo
echo "Launched ${#pids[@]} backend(s):"
for idx in "${!pids[@]}"; do
  printf "  Port %s -> PID %s\n" "${ports[$idx]}" "${pids[$idx]}"
done

echo
echo "Logs: ${LOG_DIR}/backend-<port>.log"
echo "Press Ctrl-C to stop all backends."

# Wait for all background jobs (this keeps the script running so the trap can catch Ctrl-C)
wait
