#!/usr/bin/env bash
set -euo pipefail

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
WORKER_SERVICE_URL="${WORKER_SERVICE_URL:-http://localhost:8090}"
ACCESS_TOKEN="${PERF_ACCESS_TOKEN:-${WORKER_SMOKE_ACCESS_TOKEN:-}}"
TRIP_ID="${PERF_TRIP_ID:-${WORKER_SMOKE_TRIP_ID:-}}"
POLL_ATTEMPTS="${WORKER_SMOKE_POLL_ATTEMPTS:-90}"

if [[ -z "${ACCESS_TOKEN}" || -z "${TRIP_ID}" ]]; then
  echo "Set WORKER_SMOKE_ACCESS_TOKEN/PERF_ACCESS_TOKEN and WORKER_SMOKE_TRIP_ID/PERF_TRIP_ID." >&2
  exit 1
fi
command -v curl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1 || {
  echo "curl and jq are required." >&2
  exit 1
}

SMOKE_TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${SMOKE_TMP_DIR}"' EXIT

request() {
  local method="$1" path="$2" payload="${3:-}"
  local args=(-sS -o "${SMOKE_TMP_DIR}/body.json" -w '%{http_code}' -X "${method}" -H "Authorization: Bearer ${ACCESS_TOKEN}")
  if [[ -n "${payload}" ]]; then
    args+=(-H 'Content-Type: application/json' --data "${payload}")
  fi
  LAST_STATUS="$(curl "${args[@]}" "${TRIP_SERVICE_URL}${path}")"
}

poll_job() {
  local job_id="$1" expected="$2"
  for ((attempt = 1; attempt <= POLL_ATTEMPTS; attempt++)); do
    request GET "/trips/${TRIP_ID}/generation-jobs/${job_id}"
    [[ "${LAST_STATUS}" -ge 200 && "${LAST_STATUS}" -lt 300 ]] || return 1
    status="$(jq -r '.job.status // empty' "${SMOKE_TMP_DIR}/body.json")"
    case "${status}" in
      queued|running) sleep 2 ;;
      *) [[ "${status}" == "${expected}" ]]; return ;;
    esac
  done
  return 1
}

request GET "/trips/${TRIP_ID}"
[[ "${LAST_STATUS}" -ge 200 && "${LAST_STATUS}" -lt 300 ]] || { echo "Trip lookup failed." >&2; exit 1; }
revision="$(jq -r '.trip.itineraryRevision // .itineraryRevision // 0' "${SMOKE_TMP_DIR}/body.json")"

echo "Queueing normal generation job..."
payload="$(jq -nc --argjson revision "${revision}" '{jobType:"full_generation",expectedItineraryRevision:$revision}')"
request POST "/trips/${TRIP_ID}/generation-jobs" "${payload}"
[[ "${LAST_STATUS}" == "202" ]] || { echo "Job create failed: HTTP ${LAST_STATUS}" >&2; exit 1; }
job_id="$(jq -r '.job.id // empty' "${SMOKE_TMP_DIR}/body.json")"
poll_job "${job_id}" completed || { echo "Normal job did not complete." >&2; exit 1; }

# Mock deployments may expose deterministic failure triggers through an
# instruction. These are optional because production must never expose a test
# failure switch. When set, the script verifies the public terminal behavior;
# retry/DLQ counters below provide the queue-side evidence.
if [[ -n "${WORKER_SMOKE_TRANSIENT_INSTRUCTION:-}" ]]; then
  payload="$(jq -nc --argjson revision "${revision}" --arg instruction "${WORKER_SMOKE_TRANSIENT_INSTRUCTION}" '{jobType:"full_generation",expectedItineraryRevision:$revision,instruction:$instruction}')"
  request POST "/trips/${TRIP_ID}/generation-jobs" "${payload}"
  transient_id="$(jq -r '.job.id // empty' "${SMOKE_TMP_DIR}/body.json")"
  poll_job "${transient_id}" completed || { echo "Transient fixture did not retry to completion." >&2; exit 1; }
fi
if [[ -n "${WORKER_SMOKE_PERMANENT_INSTRUCTION:-}" ]]; then
  payload="$(jq -nc --argjson revision "${revision}" --arg instruction "${WORKER_SMOKE_PERMANENT_INSTRUCTION}" '{jobType:"full_generation",expectedItineraryRevision:$revision,instruction:$instruction}')"
  request POST "/trips/${TRIP_ID}/generation-jobs" "${payload}"
  permanent_id="$(jq -r '.job.id // empty' "${SMOKE_TMP_DIR}/body.json")"
  poll_job "${permanent_id}" failed || { echo "Permanent fixture did not fail in a controlled state." >&2; exit 1; }
fi

metrics="$(curl -sS "${WORKER_SERVICE_URL}/metrics")"
for metric in worker_messages_consumed_total worker_messages_acked_total worker_messages_retried_total worker_messages_dead_lettered_total; do
  grep -q "${metric}" <<<"${metrics}" || { echo "Missing worker metric ${metric}." >&2; exit 1; }
done
echo "Worker reliability smoke passed. Optional failure fixtures: transient=${WORKER_SMOKE_TRANSIENT_INSTRUCTION:+enabled}, permanent=${WORKER_SMOKE_PERMANENT_INSTRUCTION:+enabled}."

