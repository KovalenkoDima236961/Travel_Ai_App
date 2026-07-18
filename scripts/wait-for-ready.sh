#!/usr/bin/env bash
set -euo pipefail

TIMEOUT_SECONDS=90
declare -a TARGETS=()

usage() {
  cat <<'USAGE'
Usage: scripts/wait-for-ready.sh [core|ai|all|URL ...] [--timeout SECONDS]

Polls each /ready endpoint until every target responds with HTTP 2xx. `core`
checks the local services required for the mock-first application; `ai` checks
the AI Planning Service; `all` checks both.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --timeout)
      [[ $# -ge 2 && "$2" =~ ^[1-9][0-9]*$ ]] || { echo "--timeout needs a positive number" >&2; exit 2; }
      TIMEOUT_SECONDS="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    core)
      TARGETS+=(
        "auth-service=http://localhost:8082/ready"
        "user-service=http://localhost:8083/ready"
        "trip-service=http://localhost:8080/ready"
        "notification-service=http://localhost:8086/ready"
        "worker-service=http://localhost:8090/ready"
        "external-integrations-service=http://localhost:8084/ready"
        "web-app=http://localhost:3000/api/ready"
      )
      shift
      ;;
    ai)
      TARGETS+=("ai-planning-service=http://localhost:8000/ready")
      shift
      ;;
    all)
      TARGETS+=(
        "auth-service=http://localhost:8082/ready"
        "user-service=http://localhost:8083/ready"
        "trip-service=http://localhost:8080/ready"
        "notification-service=http://localhost:8086/ready"
        "worker-service=http://localhost:8090/ready"
        "external-integrations-service=http://localhost:8084/ready"
        "web-app=http://localhost:3000/api/ready"
        "ai-planning-service=http://localhost:8000/ready"
      )
      shift
      ;;
    *)
      TARGETS+=("${1}=${1%/}/ready")
      shift
      ;;
  esac
done

[[ ${#TARGETS[@]} -gt 0 ]] || { usage >&2; exit 2; }
command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }

deadline=$((SECONDS + TIMEOUT_SECONDS))
declare -A READY=()
for target in "${TARGETS[@]}"; do READY["${target%%=*}"]=false; done

while (( SECONDS < deadline )); do
  all_ready=true
  for target in "${TARGETS[@]}"; do
    name="${target%%=*}"
    url="${target#*=}"
    [[ "${READY[$name]}" == true ]] && continue
    if curl --fail --silent --show-error --max-time 3 "${url}" >/dev/null 2>&1; then
      READY["${name}"]=true
      echo "READY ${name} (${url})"
    else
      all_ready=false
    fi
  done
  [[ "${all_ready}" == true ]] && { echo "All requested services are ready."; exit 0; }
  sleep 1
done

echo "Timed out after ${TIMEOUT_SECONDS}s waiting for readiness:" >&2
for target in "${TARGETS[@]}"; do
  name="${target%%=*}"
  [[ "${READY[$name]}" == true ]] || echo "- ${name}: ${target#*=}" >&2
done
exit 1
