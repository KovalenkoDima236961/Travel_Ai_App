#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/prod-smoke-test.sh

Required environment:
  BASE_WEB_URL
  AUTH_SERVICE_URL
  TRIP_SERVICE_URL
  NOTIFICATION_SERVICE_URL
  TEST_USER_EMAIL
  TEST_USER_PASSWORD

Optional:
  SMOKE_CREATE_TEST_USER=true|false   default true
  SMOKE_TIMEOUT_SECONDS               default 180
  OPS_ADMIN_ACCESS_TOKEN              bearer token for admin ops check
  PUBLIC_METRICS_URL                  verifies metrics is not public
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

need_tool() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "$1 is required for production smoke tests" >&2
    exit 1
  }
}
need_tool curl
need_tool jq

require() {
  name="$1"
  eval "value=\${$name:-}"
  [ -n "$value" ] || { echo "$name is required" >&2; exit 1; }
}

require BASE_WEB_URL
require AUTH_SERVICE_URL
require TRIP_SERVICE_URL
require NOTIFICATION_SERVICE_URL
require TEST_USER_EMAIL
require TEST_USER_PASSWORD

BASE_WEB_URL="${BASE_WEB_URL%/}"
AUTH_SERVICE_URL="${AUTH_SERVICE_URL%/}"
TRIP_SERVICE_URL="${TRIP_SERVICE_URL%/}"
NOTIFICATION_SERVICE_URL="${NOTIFICATION_SERVICE_URL%/}"
SMOKE_CREATE_TEST_USER="${SMOKE_CREATE_TEST_USER:-true}"
SMOKE_TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-180}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

request() {
  method="$1"
  url="$2"
  body="${3:-}"
  token="${4:-}"
  out="$TMP_DIR/response.json"
  headers="$TMP_DIR/headers.txt"
  if [ -n "$token" ]; then
    auth_header="Authorization: Bearer $token"
  else
    auth_header="X-Smoke-No-Auth: true"
  fi
  if [ -n "$body" ]; then
    status="$(curl -sS -X "$method" "$url" -H "Accept: application/json" -H "Content-Type: application/json" -H "$auth_header" -D "$headers" -o "$out" -w '%{http_code}' --data "$body")"
  else
    status="$(curl -sS -X "$method" "$url" -H "Accept: application/json" -H "$auth_header" -D "$headers" -o "$out" -w '%{http_code}')"
  fi
  printf '%s' "$status"
}

assert_2xx() {
  status="$1"
  label="$2"
  case "$status" in
    2*) ;;
    *) echo "$label failed with HTTP $status: $(cat "$TMP_DIR/response.json")" >&2; exit 1 ;;
  esac
}

echo "==> Health checks"
for url in \
  "$AUTH_SERVICE_URL/health" \
  "$TRIP_SERVICE_URL/health" \
  "$NOTIFICATION_SERVICE_URL/health" \
  "$BASE_WEB_URL"; do
  status="$(request GET "$url")"
  assert_2xx "$status" "GET $url"
done

credentials="$(jq -n --arg email "$TEST_USER_EMAIL" --arg password "$TEST_USER_PASSWORD" '{email:$email,password:$password}')"

echo "==> Auth"
if [ "$SMOKE_CREATE_TEST_USER" = "true" ]; then
  status="$(request POST "$AUTH_SERVICE_URL/auth/register" "$credentials")"
  case "$status" in
    2*) ;;
    409) status="$(request POST "$AUTH_SERVICE_URL/auth/login" "$credentials")"; assert_2xx "$status" "login after existing user" ;;
    *) echo "register failed with HTTP $status: $(cat "$TMP_DIR/response.json")" >&2; exit 1 ;;
  esac
else
  status="$(request POST "$AUTH_SERVICE_URL/auth/login" "$credentials")"
  assert_2xx "$status" "login"
fi
ACCESS_TOKEN="$(jq -r '.accessToken // empty' "$TMP_DIR/response.json")"
[ -n "$ACCESS_TOKEN" ] || { echo "Auth response did not include accessToken" >&2; exit 1; }

echo "==> Create smoke trip"
trip_payload="$(jq -n --arg title "Smoke Test City ${STAMP}" '{
  destination: $title,
  startDate: "2026-08-01",
  days: 1,
  budgetAmount: 250,
  budgetCurrency: "EUR",
  travelers: 1,
  interests: ["smoke-test"],
  pace: "balanced"
}')"
status="$(request POST "$TRIP_SERVICE_URL/trips" "$trip_payload" "$ACCESS_TOKEN")"
assert_2xx "$status" "create trip"
TRIP_ID="$(jq -r '.id // empty' "$TMP_DIR/response.json")"
TRIP_REVISION="$(jq -r '.itineraryRevision // 0' "$TMP_DIR/response.json")"
[ -n "$TRIP_ID" ] || { echo "Trip response did not include id" >&2; exit 1; }

echo "==> Create and poll generation job"
job_payload="$(jq -n --argjson revision "$TRIP_REVISION" '{
  jobType: "full_generation",
  expectedItineraryRevision: $revision,
  instruction: "Production smoke test generation"
}')"
status="$(request POST "$TRIP_SERVICE_URL/trips/$TRIP_ID/generation-jobs" "$job_payload" "$ACCESS_TOKEN")"
assert_2xx "$status" "create generation job"
JOB_ID="$(jq -r '.job.id // empty' "$TMP_DIR/response.json")"
[ -n "$JOB_ID" ] || { echo "Generation job response did not include job.id" >&2; exit 1; }

deadline=$(( $(date +%s) + SMOKE_TIMEOUT_SECONDS ))
job_status=""
while [ "$(date +%s)" -lt "$deadline" ]; do
  status="$(request GET "$TRIP_SERVICE_URL/trips/$TRIP_ID/generation-jobs/$JOB_ID" "" "$ACCESS_TOKEN")"
  assert_2xx "$status" "get generation job"
  job_status="$(jq -r '.job.status // empty' "$TMP_DIR/response.json")"
  case "$job_status" in
    completed|failed|cancelled) break ;;
  esac
  sleep 5
done

if [ "$job_status" != "completed" ]; then
  echo "generation job did not complete; final status=${job_status:-unknown}" >&2
  cat "$TMP_DIR/response.json" >&2
  exit 1
fi

status="$(request GET "$TRIP_SERVICE_URL/trips/$TRIP_ID" "" "$ACCESS_TOKEN")"
assert_2xx "$status" "fetch completed trip"
itinerary_days="$(jq '[.itinerary.days[]?] | length' "$TMP_DIR/response.json")"
[ "$itinerary_days" -gt 0 ] || { echo "Completed trip does not include itinerary days" >&2; exit 1; }

echo "==> Notifications"
status="$(request GET "$NOTIFICATION_SERVICE_URL/notifications/unread-count" "" "$ACCESS_TOKEN")"
assert_2xx "$status" "notification unread count"

echo "==> Ops access"
status="$(request GET "$TRIP_SERVICE_URL/ops/jobs/summary" "" "$ACCESS_TOKEN")"
case "$status" in
  2*) echo "ops endpoint unexpectedly allowed normal user" >&2; exit 1 ;;
  *) echo "normal user ops check returned HTTP $status as expected" ;;
esac

if [ -n "${OPS_ADMIN_ACCESS_TOKEN:-}" ]; then
  status="$(request GET "$TRIP_SERVICE_URL/ops/jobs/summary" "" "$OPS_ADMIN_ACCESS_TOKEN")"
  assert_2xx "$status" "admin ops summary"
fi

if [ -n "${PUBLIC_METRICS_URL:-}" ]; then
  status="$(request GET "$PUBLIC_METRICS_URL")"
  case "$status" in
    2*) echo "metrics endpoint is publicly reachable" >&2; exit 1 ;;
    *) echo "public metrics check returned HTTP $status as expected" ;;
  esac
fi

if [ "${SMOKE_DELETE_TRIP:-true}" = "true" ]; then
  status="$(request DELETE "$TRIP_SERVICE_URL/trips/$TRIP_ID" "" "$ACCESS_TOKEN")"
  case "$status" in
    2*|404|405) ;;
    *) echo "cleanup delete returned HTTP $status" >&2 ;;
  esac
fi

echo "Production smoke test passed for trip $TRIP_ID and job $JOB_ID"
