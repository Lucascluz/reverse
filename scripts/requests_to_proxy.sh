#!/usr/bin/env bash
#
# Make requests to the proxy and stream proxy/backend logs so you can see
# the logs from the proxy and from each backend as requests begin.
#
# This script intentionally varies request URLs (unique nonce query), mixes
# a few endpoints and methods, and toggles cacheable/non-cacheable responses
# so the proxy cannot serve the exact same cached response for all requests.
#
# Usage:
#   ./scripts/requests_to_proxy.sh R_PER_SERVER [NUM_SERVERS] [PROXY_URL] [PATH] [CONCURRENCY] [BACKEND_PORTS...]
#
# Arguments:
#   R_PER_SERVER   Required. How many requests each server should receive (used to compute total requests = R_PER_SERVER * NUM_SERVERS)
#   NUM_SERVERS    Optional. Defaults to 2.
#   PROXY_URL      Optional. Defaults to http://localhost:8080
#   PATH           Optional. Defaults to /data
#   CONCURRENCY    Optional. Number of concurrent clients. Defaults to 20.
#   BACKEND_PORTS  Optional. Space-separated list of backend ports to tail logs for (if omitted, tails logs/backend-*.log)
#
set -euo pipefail

R_PER_SERVER="${1:-}"
if [ -z "${R_PER_SERVER}" ]; then
  echo "Usage: $0 R_PER_SERVER [NUM_SERVERS] [PROXY_URL] [PATH] [CONCURRENCY] [BACKEND_PORTS...]" >&2
  exit 2
fi

NUM_SERVERS="${2:-2}"
PROXY_URL="${3:-http://localhost:8080}"
PATH_REQ="${4:-/data}"
CONCURRENCY="${5:-20}"
shift 5 || true
BACKEND_PORTS=("$@")

if ! [[ "${R_PER_SERVER}" =~ ^[0-9]+$ ]] || [ "${R_PER_SERVER}" -le 0 ]; then
  echo "R_PER_SERVER must be a positive integer" >&2
  exit 2
fi
if ! [[ "${NUM_SERVERS}" =~ ^[0-9]+$ ]] || [ "${NUM_SERVERS}" -le 0 ]; then
  echo "NUM_SERVERS must be a positive integer" >&2
  exit 2
fi
if ! [[ "${CONCURRENCY}" =~ ^[0-9]+$ ]] || [ "${CONCURRENCY}" -le 0 ]; then
  echo "CONCURRENCY must be a positive integer" >&2
  exit 2
fi

LOG_DIR="logs"
mkdir -p "${LOG_DIR}"
mkdir -p "${LOG_DIR}/clients"

TOTAL=$((R_PER_SERVER * NUM_SERVERS))
echo "Will send total=${TOTAL} requests (R_PER_SERVER=${R_PER_SERVER} * NUM_SERVERS=${NUM_SERVERS})"
echo "Proxy URL: ${PROXY_URL}${PATH_REQ}"
echo "Concurrency: ${CONCURRENCY}"
echo

# Start tails to stream logs. We prefer logs/proxy.log and logs/backend-<port>.log
TAIL_PIDS=()

# Proxy log
PROXY_LOG="${LOG_DIR}/proxy.log"
if [ -f "${PROXY_LOG}" ]; then
  echo "Tailing proxy log: ${PROXY_LOG}"
  tail -n +1 -F "${PROXY_LOG}" &
  TAIL_PIDS+=("$!")
else
  echo "Proxy log ${PROXY_LOG} not found. If you want to see proxy logs, start proxy redirecting output to ${PROXY_LOG} (e.g., ./proxy > ${PROXY_LOG} 2>&1)."
fi

# Backend logs
if [ "${#BACKEND_PORTS[@]}" -gt 0 ]; then
  for port in "${BACKEND_PORTS[@]}"; do
    lf="${LOG_DIR}/backend-${port}.log"
    if [ -f "${lf}" ]; then
      echo "Tailing backend log: ${lf}"
      tail -n +1 -F "${lf}" &
      TAIL_PIDS+=("$!")
    else
      echo "Backend log ${lf} not found. If you want to see backend logs, start backend redirecting output to ${lf} (e.g., ./backend_server -port ${port} > ${lf} 2>&1)."
    fi
  done
else
  # Tail any backend logs that exist
  shopt -s nullglob
  backend_logs=( "${LOG_DIR}"/backend-*.log )
  if [ "${#backend_logs[@]}" -gt 0 ]; then
    for lf in "${backend_logs[@]}"; do
      echo "Tailing backend log: ${lf}"
      tail -n +1 -F "${lf}" &
      TAIL_PIDS+=("$!")
    done
  else
    echo "No backend logs found in ${LOG_DIR}/backend-*.log. Start your backend(s) with output redirected to logs/backend-<port>.log to see them here."
  fi
fi

# Cleanup on exit: kill tails
cleanup() {
  for pid in "${TAIL_PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  wait || true
}
trap cleanup INT TERM EXIT

# Small set of endpoints + modes to mix requests
ENDPOINTS=( "/data" "/data" "/forward" "/vary" "/slow" )
# The duplicated "/data" increases probability of hitting it (most common case)
METHODS=( "GET" "GET" "POST" ) # some POSTs to bypass simple GET caching rules

# Prepare worker totals
reqs_per_worker=$(( (TOTAL + CONCURRENCY - 1) / CONCURRENCY ))
echo "Spawning ${CONCURRENCY} worker(s); each will send up to ${reqs_per_worker} requests (total target ${TOTAL})"
echo

pids_workers=()
client_log_prefix="${LOG_DIR}/clients/client"

# Worker function: uses curl and logs a short CSV line per request.
worker() {
  local id="$1"
  local n="$2"
  local base="$3"
  local default_path="$4"
  local logfile="${client_log_prefix}-${id}.log"
  echo "client_id,timestamp,req_no,url,method,status,backend_port,x_request_id,cacheable" > "${logfile}"

  for ((i=1;i<=n;i++)); do
    # Build request: pick endpoint and method, and add unique nonce to avoid caching
    idx=$(( (RANDOM % ${#ENDPOINTS[@]}) ))
    endpoint="${ENDPOINTS[$idx]}"
    method="${METHODS[$((RANDOM % ${#METHODS[@]} )) ]}"

    # Use default_path when endpoint == /data for consistency
    if [ "$endpoint" = "/data" ] && [ -n "${default_path:-}" ]; then
      endpoint="${default_path}"
    fi

    # Decide cacheable sometimes (only used for /data)
    cacheable="0"
    if [ "${endpoint#/}" = "${PATH_REQ#/}" ] || [ "${endpoint}" = "/data" ]; then
      # Make roughly 30% of requests cacheable to exercise cache paths
      if [ $((RANDOM % 100)) -lt 30 ]; then
        cacheable="1"
      fi
    fi

    # Unique nonce so proxy can't collapse all requests into one cache entry
    nonce="$(date +%s%N)-${id}-${i}-$RANDOM"

    # Build URL
    url="${base}${endpoint}"
    # Add query params (cache and nonce)
    if [[ "$url" == *"?"* ]]; then
      url="${url}&_nonce=${nonce}&cache=${cacheable}"
    else
      url="${url}?_nonce=${nonce}&cache=${cacheable}"
    fi

    hdr_file=$(mktemp)
    body_file=$(mktemp)

    # Use headers to further avoid caches and to show forwarded headers
    # For POST include a small JSON body to ensure proxy forwards it
    if [ "$method" = "POST" ]; then
      curl -s -S -X POST -H "Content-Type: application/json" -H "Cache-Control: no-cache" -H "X-Request-Nonce: ${nonce}" -d "{\"client\":${id},\"i\":${i}}" -D "${hdr_file}" -o "${body_file}" "${url}" || true
    else
      curl -s -S -G -H "Cache-Control: no-cache" -H "X-Request-Nonce: ${nonce}" -D "${hdr_file}" -o "${body_file}" "${url}" || true
    fi

    status=$(awk 'NR==1{print $2}' "${hdr_file}" | tr -d '\r' || echo "")
    backend_port=$(awk 'tolower($0) ~ /^x-backend-port:/ {print $2}' "${hdr_file}" | tr -d '\r' || echo "")
    x_request_id=$(awk 'tolower($0) ~ /^x-request-id:/ {print $2}' "${hdr_file}" | tr -d '\r' || echo "")
    timestamp=$(date -Iseconds)

    printf "%s,%s,%d,%s,%s,%s,%s,%s,%s\n" "${id}" "${timestamp}" "${i}" "${url}" "${method}" "${status:-}" "${backend_port:-}" "${x_request_id:-}" "${cacheable}" >> "${logfile}"

    rm -f "${hdr_file}" "${body_file}"

    # small jitter so workers are not perfectly synced
    sleep 0.01
  done
}

# Launch workers
for w in $(seq 1 "${CONCURRENCY}"); do
  worker "${w}" "${reqs_per_worker}" "${PROXY_URL}" "${PATH_REQ}" &
  pids_workers+=("$!")
  sleep 0.005
done

# Wait for workers
wait "${pids_workers[@]}" || true

echo
echo "All client workers finished. Requests done."

# Aggregate backend contact counts from client logs and print summary
echo
echo "Summary of backend hits (collected from X-Backend-Port headers in client logs):"
declare -A counts
for lf in "${LOG_DIR}/clients/"*.log; do
  if [ ! -f "$lf" ]; then
    continue
  fi
  awk -F, 'NR>1 && $6 != "" { print $6 }' "$lf" | while read -r port; do
    counts["$port"]=$((counts["$port"] + 1))
  done
done

if [ "${#counts[@]}" -eq 0 ]; then
  echo "  (no backend_port values found in client logs)"
else
  for k in "${!counts[@]}"; do
    printf "  backend %s -> %d requests\n" "$k" "${counts[$k]}"
  done
fi

echo
echo "Tails for proxy/backend logs are still running; press Ctrl-C to stop them and exit."
# Keep tails running until user stops script (Ctrl-C) so you can continue to observe logs.
wait
