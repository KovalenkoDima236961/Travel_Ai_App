#!/usr/bin/env bash
set -euo pipefail

# Lightweight local/staging read-path check. It intentionally measures only
# compact GET endpoints and never records response bodies or credentials.
TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://localhost:8082}"
NOTIFICATION_SERVICE_URL="${NOTIFICATION_SERVICE_URL:-http://localhost:8086}"
ITERATIONS="${PERF_SMOKE_ITERATIONS:-10}"
P95_THRESHOLD_MS="${PERF_SMOKE_P95_THRESHOLD_MS:-1500}"
ACCESS_TOKEN="${PERF_ACCESS_TOKEN:-}"
TRIP_ID="${PERF_TRIP_ID:-}"
JOB_ID="${PERF_JOB_ID:-}"

for command_name in curl jq awk sort sed mktemp; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    echo "${command_name} is required." >&2
    exit 1
  }
done

perf_tmp_dir="$(mktemp -d)"
trap 'rm -rf "${perf_tmp_dir}"' EXIT

if [[ -z "${ACCESS_TOKEN}" ]]; then
  if [[ -z "${PERF_SMOKE_EMAIL:-}" || -z "${PERF_SMOKE_PASSWORD:-}" ]]; then
    echo "Set PERF_ACCESS_TOKEN, or PERF_SMOKE_EMAIL and PERF_SMOKE_PASSWORD." >&2
    exit 1
  fi
  login_payload="$(jq -nc --arg email "${PERF_SMOKE_EMAIL}" --arg password "${PERF_SMOKE_PASSWORD}" '{email:$email,password:$password}')"
  login_status="$(curl -sS -o "${perf_tmp_dir}/login.json" -w '%{http_code}' -H 'Content-Type: application/json' --data "${login_payload}" "${AUTH_SERVICE_URL}/auth/login")"
  if [[ "${login_status}" -lt 200 || "${login_status}" -ge 300 ]]; then
    echo "Login failed with HTTP ${login_status}." >&2
    exit 1
  fi
  ACCESS_TOKEN="$(jq -r '.accessToken // empty' "${perf_tmp_dir}/login.json")"
fi

if [[ -z "${TRIP_ID}" ]]; then
  list_status="$(curl -sS -o "${perf_tmp_dir}/trips.json" -w '%{http_code}' -H "Authorization: Bearer ${ACCESS_TOKEN}" "${TRIP_SERVICE_URL}/trips?limit=1")"
  if [[ "${list_status}" -lt 200 || "${list_status}" -ge 300 ]]; then
    echo "Could not list trips (HTTP ${list_status})." >&2
    exit 1
  fi
  TRIP_ID="$(jq -r '.items[0].id // .trips[0].id // empty' "${perf_tmp_dir}/trips.json")"
fi

if [[ -z "${TRIP_ID}" ]]; then
  echo "No trip found. Seed a test trip or set PERF_TRIP_ID." >&2
  exit 1
fi

endpoints=(
  "trips|${TRIP_SERVICE_URL}/trips?limit=20"
  "trip_detail|${TRIP_SERVICE_URL}/trips/${TRIP_ID}"
  "command_center|${TRIP_SERVICE_URL}/trips/${TRIP_ID}/command-center-summary"
  "health|${TRIP_SERVICE_URL}/trips/${TRIP_ID}/health"
  "verification|${TRIP_SERVICE_URL}/trips/${TRIP_ID}/verification"
  "budget_summary|${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary"
  "library|${TRIP_SERVICE_URL}/trips/library?limit=30"
  "search|${TRIP_SERVICE_URL}/search?q=Paris&limit=10"
  "unread_count|${NOTIFICATION_SERVICE_URL}/notifications/unread-count"
)
if [[ -n "${JOB_ID}" ]]; then
  endpoints+=("generation_job|${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generation-jobs/${JOB_ID}")
fi

failed=0
echo "Backend performance smoke: trip=${TRIP_ID}, iterations=${ITERATIONS}, p95 threshold=${P95_THRESHOLD_MS}ms"
for definition in "${endpoints[@]}"; do
  label="${definition%%|*}"
  url="${definition#*|}"
  timings="${perf_tmp_dir}/${label}.timings"
  statuses="${perf_tmp_dir}/${label}.statuses"
  : >"${timings}"
  : >"${statuses}"

  for ((iteration = 1; iteration <= ITERATIONS; iteration++)); do
    result="$(curl -sS -o /dev/null -w '%{http_code} %{time_total}' -H "Authorization: Bearer ${ACCESS_TOKEN}" "${url}")"
    status="${result%% *}"
    seconds="${result#* }"
    awk -v seconds="${seconds}" 'BEGIN { printf "%.3f\n", seconds * 1000 }' >>"${timings}"
    echo "${status}" >>"${statuses}"
    if [[ "${status}" -ge 500 ]]; then
      failed=1
    fi
  done

  average="$(awk '{sum += $1} END {if (NR) printf "%.1f", sum / NR; else print "0"}' "${timings}")"
  sorted="${perf_tmp_dir}/${label}.sorted"
  sort -n "${timings}" >"${sorted}"
  p95_index="$(awk -v count="${ITERATIONS}" 'BEGIN { value=int(count*0.95); if (value < count*0.95) value++; if (value < 1) value=1; print value }')"
  p95="$(sed -n "${p95_index}p" "${sorted}")"
  status_counts="$(sort "${statuses}" | uniq -c | awk '{printf "%s%s=%s", separator, $2, $1; separator=","}')"
  printf '%-20s avg=%7sms p95=%7sms status={%s}\n' "${label}" "${average}" "${p95}" "${status_counts}"
  if awk -v p95="${p95}" -v threshold="${P95_THRESHOLD_MS}" 'BEGIN { exit !(p95 > threshold) }'; then
    echo "${label}: p95 ${p95}ms exceeds ${P95_THRESHOLD_MS}ms." >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  echo "Backend performance smoke failed." >&2
  exit 1
fi
echo "Backend performance smoke passed."
