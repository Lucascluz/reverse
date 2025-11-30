#!/usr/bin/env bash
#
# Run N backends and C clients that make R requests each to a proxy.
#
# Usage:
#   ./scripts/run_backends_and_clients.sh N C R [start_port] [proxy_url] [path]
#
# Arguments:
#   N           Number of backend instances to start (ports start at start_port)
#   C           Number of concurrent clients to start
#   R           Requests per client
#   start_port  Optional. Default: 8081
#   proxy_url   Optional. Default: http://localhost:8080
#   path        Optional. Default: /data
#
# Examples:
#   ./scripts/run_backends_and_clients.sh 5 5 100
#     -> start 5 backends on ports 8081..8085, 5 clients, each sending 100 requests to http://localhost:8080/data
#
# Logs:
#   - Backend logs: logs/backend-<port>.log
#   - Client logs:  logs/client-<id>.log
#
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

if [ $# -lt 3 ]; then
  echo "Usage: $0 N C R [start_port] [proxy_url] [path]" >&2
  echo "  N = number of backends" >&2
  echo "  C = number of clients" >&2
  echo "  R = requests per client" >&2
  exit 2
fi

N="$1"
C="$2"
R="$3"
START_PORT="${4:-8081}"
PROXY_URL="${5:-http://localhost:8080}"
PATH_REQ="${6:-/data}"

if ! [[ "${N}" =~ ^[0-9]+$ ]] || [ "${N}" -le 0 ]; then
  echo "N must be a positive integer" >&2
  exit 2
fi
if ! [[ "${C}" =~ ^[0-9]+$ ]] || [ "${C}" -le 0 ]; then
  echo "C must be a positive integer" >&2
  exit 2
fi
if ! [[ "${R}" =~ ^[0-9]+$ ]] || [ "${R}" -le 0 ]; then
  echo "R must be a positive integer" >&2
  exit 2
fi
if ! [[ "${START_PORT}" =~ ^[0-9]+$ ]]; then
  echo "start_port must be an integer" >&2
  exit 2
fi

mkdir -p "${BIN_DIR}" "${LOG_DIR}"

echo "Building ${PROG} -> ${BIN_PATH} ..."
go build -o "${BIN_PATH}" "${PROG}"

backend_pids=()
client_pids=()
backend_ports=()

cleanup() {
  echo
  echo "Stopping ${#client_pids[@]} client(s) and ${#backend_pids[@]} backend(s)..."
  for pid in "${client_pids[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "  Killing client PID $pid"
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  for pid in "${backend_pids[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "  Killing backend PID $pid"
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
  "${BIN_PATH}" -port "${port}" > "${log_file}" 2>&1 &
  pid=$!
  backend_pids+=("${pid}")
  backend_ports+=("${port}")
  sleep 0.05
done

echo
echo "Launched ${#backend_pids[@]} backend(s):"
for idx in "${!backend_pids[@]}"; do
  printf "  Port %s -> PID %s\n" "${backend_ports[$idx]}" "${backend_pids[$idx]}"
done

echo
echo "Starting ${C} client(s). Each will perform ${R} request(s) to ${PROXY_URL}${PATH_REQ}"
echo "Client logs will be written to ${LOG_DIR}/client-<id>.log"
sleep 0.5

# Clients will use bash associative arrays to track last seen request-id per backend+path
# We will spawn client functions in background processes.

client_worker() {
  local client_id="$1"
  local reqs="$2"
  local base_url="$3"
  local path="$4"
  local logfile="${LOG_DIR}/client-${client_id}.log"

  # declare local associative array for last_seen
  declare -A last_seen=()

  echo "client_id,timestamp,req_no,url,status,backend_port,x_request_id,cached,body_snippet" > "${logfile}"

  for ((i=1;i<=reqs;i++)); do
    url="${base_url}${path}"

    hdr_file=$(mktemp)
    body_file=$(mktemp)

    # Perform request, capture headers and body. Follow redirects but keep headers in hdr_file.
    curl -s -S -D "${hdr_file}" -o "${body_file}" -L "${url}" || true

    status=$(awk 'NR==1{print $2}' "${hdr_file}" | tr -d '\r' || echo "")
    # Try cache header common names
    cache_hdr=$(awk 'tolower($0) ~ /^x-cache|^x-cache-status|^x-accel-cache-status/ {print $0}' "${hdr_file}" | head -n1 | tr -d '\r' || echo "")

    # Extract X-Backend-Port or fallback to parsing body for "Backend:<port>"
    backend_port=$(awk 'tolower($0) ~ /^x-backend-port:/ {print $2}' "${hdr_file}" | tr -d '\r' || echo "")
    if [ -z "${backend_port}" ]; then
      # try to extract from response body: "Backend:1234"
      backend_port=$(sed -n '1,4p' "${body_file}" | tr -d '\r' | grep -o 'Backend:[0-9]*' | head -n1 | cut -d: -f2 || echo "")
    fi

    x_request_id=$(awk 'tolower($0) ~ /^x-request-id:/ {print $2}' "${hdr_file}" | tr -d '\r' || echo "")

    # Determine cached:
    cached="unknown"
    if [ -n "${cache_hdr}" ]; then
      # examine cache header for common "HIT" tokens
      if echo "${cache_hdr}" | grep -iqE 'hit|HIT|HIT-OC|HIT$'; then
        cached="true"
      else
        # if it contains miss or miss-like token
        if echo "${cache_hdr}" | grep -iqE 'miss|MISS|MISS$'; then
          cached="false"
        else
          cached="unknown"
        fi
      fi
    else
      # fallback: use last_seen heuristic based on backend_port + path
      if [ -n "${backend_port}" ] && [ -n "${x_request_id}" ]; then
        key="${backend_port}:${path}"
        prev="${last_seen[$key]:-}"
        if [ -n "${prev}" ] && [ "${x_request_id}" = "${prev}" ]; then
          cached="true"
        else
          cached="false"
          last_seen[$key]="${x_request_id}"
        fi
      else
        cached="unknown"
      fi
    fi

    # Body snippet
    body_snip=$(tr -d '\r' < "${body_file}" | sed -n '1,2p' | tr '\n' ' ' | cut -c1-200 | sed 's/,/ /g')
    timestamp=$(date -Iseconds)

    # Log CSV line
    printf "%s,%s,%d,%s,%s,%s,%s,%s,%s\n" \
      "${client_id}" "${timestamp}" "${i}" "${url}" "${status:-}" "${backend_port:-}" "${x_request_id:-}" "${cached}" "${body_snip}" >> "${logfile}"

    rm -f "${hdr_file}" "${body_file}"

    # small jitter so clients are not perfectly synced
    sleep 0.02
  done
}

# Start clients
for cid in $(seq 1 "${C}"); do
  client_worker "${cid}" "${R}" "${PROXY_URL}" "${PATH_REQ}" > /dev/null 2>&1 &
  client_pids+=("$!")
  sleep 0.02
done

echo
echo "Launched ${#client_pids[@]} client(s):"
for pid in "${client_pids[@]}"; do
  echo "  PID $pid"
done

echo
echo "Waiting for clients to finish..."
# Wait for clients to exit
wait "${client_pids[@]}" || true

echo "Clients finished. Backends are still running. Press Ctrl-C to stop backends (or run the cleanup)."

# If script reaches here (clients done), optionally wait forever to keep backends running,
# or exit and leave backends running. We'll wait to allow inspection; user can Ctrl-C.
wait
