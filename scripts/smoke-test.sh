#!/usr/bin/env bash
set -euo pipefail

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://localhost:8082}"
USER_SERVICE_URL="${SMOKE_USER_SERVICE_URL:-${USER_SERVICE_URL:-http://localhost:8083}}"
AI_PLANNING_SERVICE_URL="${SMOKE_AI_PLANNING_SERVICE_URL:-${AI_PLANNING_SERVICE_URL:-http://localhost:8000}}"
EXTERNAL_INTEGRATIONS_SERVICE_URL="${SMOKE_EXTERNAL_INTEGRATIONS_SERVICE_URL:-${NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL:-http://localhost:8084}}"
NOTIFICATION_SERVICE_URL="${SMOKE_NOTIFICATION_SERVICE_URL:-${NOTIFICATION_SERVICE_URL:-http://localhost:8086}}"
WORKER_SERVICE_URL="${SMOKE_WORKER_SERVICE_URL:-${WORKER_SERVICE_URL:-http://localhost:8090}}"
WEB_APP_URL="${WEB_APP_URL:-http://localhost:3000}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
INTERNAL_SERVICE_TOKEN_FOR_SMOKE="${SMOKE_INTERNAL_SERVICE_TOKEN:-${INTERNAL_SERVICE_TOKEN:-dev-internal-service-token}}"

if [[ "${USER_SERVICE_URL}" == "http://user-service:8083" ]]; then
  USER_SERVICE_URL="http://localhost:${USER_SERVICE_PORT:-8083}"
fi
if [[ "${NOTIFICATION_SERVICE_URL}" == "http://notification-service:8086" ]]; then
  NOTIFICATION_SERVICE_URL="http://localhost:${NOTIFICATION_SERVICE_PORT:-8086}"
fi
if [[ "${AI_PLANNING_SERVICE_URL}" == "http://ai-planning-service:8000" ]]; then
  AI_PLANNING_SERVICE_URL="http://localhost:${AI_HTTP_PORT:-8000}"
fi
if [[ "${EXTERNAL_INTEGRATIONS_SERVICE_URL}" == "http://external-integrations-service:8084" ]]; then
  EXTERNAL_INTEGRATIONS_SERVICE_URL="http://localhost:${EXTERNAL_INTEGRATIONS_SERVICE_PORT:-8084}"
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required to run the smoke test." >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to run the smoke test. Install jq and try again." >&2
  exit 1
fi

LAST_STATUS=""
LAST_BODY=""

request() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  local response_file
  response_file="$(mktemp)"

  if [[ -n "${body}" ]]; then
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        -H "Content-Type: application/json" \
        --data "${body}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  else
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  fi

  LAST_BODY="$(cat "${response_file}")"
  rm -f "${response_file}"
}

request_with_bearer() {
  local method="$1"
  local url="$2"
  local token="$3"
  local body="${4:-}"
  local response_file
  response_file="$(mktemp)"

  if [[ -n "${body}" ]]; then
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${token}" \
        --data "${body}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  else
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        -H "Authorization: Bearer ${token}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  fi

  LAST_BODY="$(cat "${response_file}")"
  rm -f "${response_file}"
}

request_receipt_upload() {
  local url="$1"
  local token="$2"
  local file_path="$3"
  local filename="$4"
  local expense_id="${5:-}"
  local response_file
  response_file="$(mktemp)"
  local curl_args=(
    -sS
    -o "${response_file}"
    -w "%{http_code}"
    -X POST
    -H "Authorization: Bearer ${token}"
    -F "file=@${file_path};filename=${filename};type=image/png"
    -F "runOcr=true"
  )
  if [[ -n "${expense_id}" ]]; then
    curl_args+=(-F "expenseId=${expense_id}")
  fi
  if ! LAST_STATUS="$(curl "${curl_args[@]}" "${url}")"; then
    LAST_BODY="$(cat "${response_file}")"
    rm -f "${response_file}"
    return 1
  fi
  LAST_BODY="$(cat "${response_file}")"
  rm -f "${response_file}"
}

request_with_internal_token() {
  local method="$1"
  local url="$2"
  local token="$3"
  local body="${4:-}"
  local response_file
  response_file="$(mktemp)"

  if [[ -n "${body}" ]]; then
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        -H "Content-Type: application/json" \
        -H "X-Internal-Service-Token: ${token}" \
        --data "${body}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  else
    if ! LAST_STATUS="$(
      curl -sS -o "${response_file}" -w "%{http_code}" \
        -X "${method}" \
        -H "X-Internal-Service-Token: ${token}" \
        "${url}"
    )"; then
      LAST_BODY="$(cat "${response_file}")"
      rm -f "${response_file}"
      return 1
    fi
  fi

  LAST_BODY="$(cat "${response_file}")"
  rm -f "${response_file}"
}

assert_2xx() {
  local label="$1"
  if [[ ! "${LAST_STATUS}" =~ ^2 ]]; then
    echo "${label} failed with HTTP ${LAST_STATUS}" >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

assert_status() {
  local label="$1"
  local expected="$2"
  if [[ "${LAST_STATUS}" != "${expected}" ]]; then
    echo "${label} expected HTTP ${expected}, got HTTP ${LAST_STATUS}" >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

assert_metrics_contains() {
  local label="$1"
  local url="$2"
  local metric_name="$3"

  request GET "${url}"
  assert_2xx "${label}"
  if [[ "${LAST_BODY}" != *"# HELP"* && "${LAST_BODY}" != *"# TYPE"* ]]; then
    echo "${label}: response does not look like Prometheus text exposition." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  if [[ "${LAST_BODY}" != *"${metric_name}"* ]]; then
    echo "${label}: missing metric family '${metric_name}'." >&2
    exit 1
  fi
}

check_prometheus_targets_if_requested() {
  if [[ "${SMOKE_CHECK_PROMETHEUS_TARGETS:-false}" != "true" ]]; then
    return 0
  fi

  request GET "${PROMETHEUS_URL}/api/v1/targets"
  assert_2xx "Prometheus target health"
  if ! jq -e '.status == "success" and ([.data.activeTargets[] | select(.health == "up")] | length > 0)' <<<"${LAST_BODY}" >/dev/null; then
    echo "Prometheus did not report any active targets as up." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

poll_generation_job() {
  local label="$1"
  local trip_id="$2"
  local job_id="$3"
  local token="$4"
  local max_attempts="${5:-60}"
  local attempt
  local job_status

  for ((attempt = 1; attempt <= max_attempts; attempt++)); do
    request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${trip_id}/generation-jobs/${job_id}" "${token}"
    assert_2xx "${label} job poll"
    job_status="$(jq -r '.job.status // empty' <<<"${LAST_BODY}")"
    case "${job_status}" in
      completed|failed|cancelled)
        return 0
        ;;
      queued|running)
        sleep 2
        ;;
      *)
        echo "${label}: unexpected generation job status '${job_status}'" >&2
        echo "${LAST_BODY}" >&2
        exit 1
        ;;
    esac
  done

  echo "${label}: generation job did not finish after ${max_attempts} attempts" >&2
  echo "${LAST_BODY}" >&2
  exit 1
}

# assert_activity_has fails unless the last activity response (LAST_BODY)
# contains at least one event of the given type.
assert_activity_has() {
  local label="$1"
  local event_type="$2"
  if ! jq -e --arg t "${event_type}" '.items | any(.eventType == $t)' <<<"${LAST_BODY}" >/dev/null 2>&1; then
    echo "${label}: activity feed missing expected event type '${event_type}'" >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

assert_activity_stream_opens() {
  local label="$1"
  local token="$2"
  local response_file
  local status
  response_file="$(mktemp)"
  status="$(
    curl -s --max-time 2 -o "${response_file}" -w "%{http_code}" \
      -H "Accept: text/event-stream" \
      -H "Authorization: Bearer ${token}" \
      "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream" || true
  )"
  rm -f "${response_file}"
  if [[ "${status}" != "200" ]]; then
    echo "${label}: expected activity stream HTTP 200, got HTTP ${status}" >&2
    exit 1
  fi
}

# fetch_notifications loads a user's notifications into LAST_BODY. Trip Service
# creates notifications synchronously, but a tiny retry guards against scheduling
# jitter between the action returning and the row being visible.
fetch_notifications() {
  local token="$1"
  local attempt
  for attempt in 1 2 3 4 5; do
    request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications?limit=100" "${token}"
    if [[ "${LAST_STATUS}" =~ ^2 ]]; then
      return 0
    fi
    sleep 0.5
  done
  echo "Fetching notifications failed with HTTP ${LAST_STATUS}" >&2
  echo "${LAST_BODY}" >&2
  exit 1
}

# assert_notification_has fails unless the given user has at least one
# notification of the given type, retrying briefly to absorb timing.
assert_notification_has() {
  local label="$1"
  local token="$2"
  local notification_type="$3"
  local attempt
  for attempt in 1 2 3 4 5; do
    fetch_notifications "${token}"
    if jq -e --arg t "${notification_type}" '.items | any(.type == $t)' <<<"${LAST_BODY}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "${label}: notifications missing expected type '${notification_type}'" >&2
  echo "${LAST_BODY}" >&2
  exit 1
}

# assert_notification_absent fails if the given user has a notification of the
# given type. It fetches once; use it after a synchronous action has returned.
assert_notification_absent() {
  local label="$1"
  local token="$2"
  local notification_type="$3"
  fetch_notifications "${token}"
  if jq -e --arg t "${notification_type}" '.items | any(.type == $t)' <<<"${LAST_BODY}" >/dev/null 2>&1; then
    echo "${label}: notifications unexpectedly contained type '${notification_type}'" >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

# unread_count returns the unread notification count for a user.
unread_count() {
  local token="$1"
  request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/unread-count" "${token}"
  assert_2xx "Unread notification count"
  jq -r '.count // 0' <<<"${LAST_BODY}"
}

echo "Checking Auth Service health..."
request GET "${AUTH_SERVICE_URL}/health"
assert_2xx "Auth Service health check"

echo "Checking Trip Service health..."
request GET "${TRIP_SERVICE_URL}/health"
assert_2xx "Trip Service health check"

if [[ "${SMOKE_EXPECT_WORKER_SERVICE:-true}" == "true" ]]; then
  echo "Checking Worker Service health..."
  request GET "${WORKER_SERVICE_URL}/health"
  assert_2xx "Worker Service health check"

  echo "Checking Worker Service readiness..."
  request GET "${WORKER_SERVICE_URL}/ready"
  assert_2xx "Worker Service readiness check"
fi

echo "Checking User Service health..."
request GET "${USER_SERVICE_URL}/health"
assert_2xx "User Service health check"

echo "Checking AI Planning Service health..."
request GET "${AI_PLANNING_SERVICE_URL}/health"
assert_2xx "AI Planning Service health check"

echo "Checking AI generation repair endpoint..."
AI_REPAIR_PAYLOAD="$(
  jq -nc '{
    generationType:"full_itinerary",
    currentOutput:{
      destination:"Vienna",
      currency:"EUR",
      days:[{
        day:1,
        title:"Arrival",
        primaryStopId:"stop_1",
        items:[{
          time:"09:00",
          endTime:"10:00",
          type:"activity",
          name:"Old town walk",
          estimatedCost:{amount:12,currency:"EUR",category:"activity"}
        }]
      }]
    },
    validationIssues:[
      {
        id:"activity_before_transport_arrival:day_1:item_0:leg_1",
        category:"transport",
        severity:"critical",
        title:"Activity starts before transport arrival",
        fixability:"fixable_by_ai",
        dayNumber:1,
        itemIndex:0,
        routeLegId:"leg_1"
      },
      {
        id:"transfer_item_missing_or_mismatch:leg_1",
        category:"transport",
        severity:"high",
        title:"Selected transport is missing from itinerary",
        fixability:"fixable_by_ai",
        dayNumber:1,
        routeLegId:"leg_1"
      }
    ],
    planningContext:{
      trip:{Destination:"Vienna",Days:1,BudgetCurrency:"EUR"},
      route:{
        stops:[{id:"stop_1",destination:"Vienna"}],
        legs:[{
          id:"leg_1",
          fromStopId:"origin",
          toStopId:"stop_1",
          fromName:"Bratislava",
          toName:"Vienna",
          mode:"train",
          selectedTransportOption:{
            id:"opt_1",
            mode:"train",
            provider:"mock",
            departureDate:"2026-09-10",
            departureTime:"08:00",
            arrivalDate:"2026-09-10",
            arrivalTime:"11:00",
            durationMinutes:180,
            estimatedPrice:{amount:18,currency:"EUR"}
          }
        }]
      }
    },
    repairScope:{type:"full_output"},
    constraints:{
      preserveUnaffectedDays:true,
      preserveUserEditedItems:true,
      outputLanguage:"en"
    }
  }'
)"
request POST "${AI_PLANNING_SERVICE_URL}/repair-generation-output" "${AI_REPAIR_PAYLOAD}"
assert_2xx "AI generation repair"
if ! jq -e '
  .repairedOutput.days[0].items[] | select(.transfer.legId == "leg_1")
' <<<"${LAST_BODY}" >/dev/null; then
  echo "AI generation repair did not add the expected transfer item." >&2
  exit 1
fi

echo "Checking External Integrations Service health..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/health"
assert_2xx "External Integrations Service health check"

echo "Checking Notification Service health..."
request GET "${NOTIFICATION_SERVICE_URL}/health"
assert_2xx "Notification Service health check"

echo "Checking Notification Service readiness..."
request GET "${NOTIFICATION_SERVICE_URL}/ready"
assert_2xx "Notification Service readiness check"

if [[ "${SMOKE_EXPECT_OBSERVABILITY:-true}" == "true" ]]; then
  echo "Checking service metrics endpoints..."
  assert_metrics_contains "Auth Service metrics" "${AUTH_SERVICE_URL}/metrics" "http_requests_total"
  assert_metrics_contains "Trip Service metrics" "${TRIP_SERVICE_URL}/metrics" "http_requests_total"
  assert_metrics_contains "User Service metrics" "${USER_SERVICE_URL}/metrics" "http_requests_total"
  assert_metrics_contains "AI Planning Service metrics" "${AI_PLANNING_SERVICE_URL}/metrics" "http_requests_total"
  assert_metrics_contains "External Integrations Service metrics" "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/metrics" "http_requests_total"
  assert_metrics_contains "Notification Service metrics" "${NOTIFICATION_SERVICE_URL}/metrics" "http_requests_total"
  if [[ "${SMOKE_EXPECT_WORKER_SERVICE:-true}" == "true" ]]; then
    assert_metrics_contains "Worker Service metrics" "${WORKER_SERVICE_URL}/metrics" "http_requests_total"
  fi
  check_prometheus_targets_if_requested
fi

echo "Checking Notification Service stream requires auth..."
request GET "${NOTIFICATION_SERVICE_URL}/notifications/stream"
assert_status "Notification stream requires auth" "401"

echo "Checking Trip Service presence stream requires auth..."
request GET "${TRIP_SERVICE_URL}/trips/00000000-0000-0000-0000-000000000001/presence/stream"
assert_status "Trip presence stream requires auth" "401"

echo "Checking Trip Service presence state requires auth..."
request POST "${TRIP_SERVICE_URL}/trips/00000000-0000-0000-0000-000000000001/presence/state" '{"state":"viewing"}'
assert_status "Trip presence state requires auth" "401"

echo "Checking Trip Service edit lock requires auth..."
request GET "${TRIP_SERVICE_URL}/trips/00000000-0000-0000-0000-000000000001/edit-lock"
assert_status "Trip edit lock requires auth" "401"

PLACE_PROVIDER_MODE="${PLACE_PROVIDER:-mock}"
PLACE_PROVIDER_FALLBACK="${PLACE_PROVIDER_FALLBACK_TO_MOCK:-true}"

if [[ "${PLACE_PROVIDER_MODE}" == "foursquare" && -n "${FOURSQUARE_API_KEY:-}" ]]; then
  echo "Searching places with Foursquare provider..."
else
  echo "Searching mock places..."
fi
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/places/search?query=Colosseum&destination=Rome"
assert_2xx "Place search"

PLACE_JSON="$(jq -c '.items[0] // null' <<<"${LAST_BODY}")"
if [[ "${PLACE_JSON}" == "null" ]]; then
  echo "Place search did not return any results." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
PLACE_ID="$(jq -r '.providerPlaceId // empty' <<<"${PLACE_JSON}")"
PLACE_NAME="$(jq -r '.name // empty' <<<"${PLACE_JSON}")"
PLACE_PROVIDER_NAME="$(jq -r '.provider // empty' <<<"${PLACE_JSON}")"
PLACE_OPENING_HOURS_COUNT="$(jq '.openingHours | length' <<<"${PLACE_JSON}")"
if [[ -z "${PLACE_ID}" || -z "${PLACE_NAME}" || -z "${PLACE_PROVIDER_NAME}" ]]; then
  echo "Place search did not return a normalized place." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${PLACE_PROVIDER_MODE}" == "foursquare" && -n "${FOURSQUARE_API_KEY:-}" ]]; then
  if [[ "${PLACE_PROVIDER_NAME}" != "foursquare" ]]; then
    if [[ "${PLACE_PROVIDER_FALLBACK}" == "true" && "${PLACE_PROVIDER_NAME}" == "mock" ]]; then
      echo "Foursquare search fell back to mock provider."
    else
      echo "Expected Foursquare provider result, got '${PLACE_PROVIDER_NAME}'." >&2
      echo "${LAST_BODY}" >&2
      exit 1
    fi
  else
    echo "Foursquare place search returned ${PLACE_NAME}."
  fi
else
  if [[ "${PLACE_PROVIDER_NAME}" != "mock" || "${PLACE_NAME}" != "Colosseum" ]]; then
    echo "Mock place search did not return Colosseum as the first result." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  if [[ "${PLACE_OPENING_HOURS_COUNT}" -lt 1 ]]; then
    echo "Mock place search result did not include openingHours." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi

echo "Fetching place details..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/places/${PLACE_ID}"
assert_2xx "Place details"

DETAIL_PLACE_ID="$(jq -r '.providerPlaceId // empty' <<<"${LAST_BODY}")"
DETAIL_PROVIDER_NAME="$(jq -r '.provider // empty' <<<"${LAST_BODY}")"
DETAIL_OPENING_HOURS_COUNT="$(jq '.openingHours | length' <<<"${LAST_BODY}")"
if [[ "${DETAIL_PLACE_ID}" != "${PLACE_ID}" ]]; then
  echo "Place details did not return the requested place." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${DETAIL_PROVIDER_NAME}" == "mock" && "${DETAIL_OPENING_HOURS_COUNT}" -lt 1 ]]; then
  echo "Mock place details did not include expected openingHours." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
PLACE_JSON="$(jq -c '.' <<<"${LAST_BODY}")"
PLACE_PROVIDER_NAME="${DETAIL_PROVIDER_NAME}"

echo "Estimating a mock walking route..."
ROUTE_PAYLOAD='{"mode":"walking","stops":[{"name":"Colosseum","latitude":41.8902,"longitude":12.4922},{"name":"Trevi Fountain","latitude":41.9009,"longitude":12.4833}]}'
request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" "${ROUTE_PAYLOAD}"
assert_2xx "Route estimate"

ROUTE_PROVIDER_NAME="$(jq -r '.provider // empty' <<<"${LAST_BODY}")"
ROUTE_DISTANCE="$(jq -r '.distanceKm // 0' <<<"${LAST_BODY}")"
ROUTE_DURATION="$(jq -r '.durationMinutes // 0' <<<"${LAST_BODY}")"
ROUTE_SEGMENTS="$(jq -r '.segments | length' <<<"${LAST_BODY}")"
if [[ "${ROUTE_PROVIDER_NAME}" != "mock" ]]; then
  echo "Route estimate did not report the mock provider (got '${ROUTE_PROVIDER_NAME}')." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.distanceKm > 0 and .durationMinutes > 0 and (.segments | length) == 1' >/dev/null <<<"${LAST_BODY}"; then
  echo "Route estimate must return distanceKm>0, durationMinutes>0, and exactly 1 segment." >&2
  echo "distanceKm=${ROUTE_DISTANCE} durationMinutes=${ROUTE_DURATION} segments=${ROUTE_SEGMENTS}" >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking route estimate validation rejects a single stop..."
request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" '{"mode":"walking","stops":[{"name":"Colosseum","latitude":41.8902,"longitude":12.4922}]}'
assert_status "Route estimate rejects fewer than 2 stops" "400"

echo "Checking deterministic multi-modal route estimates..."
for MODE in train bus flight ferry; do
  MODE_ROUTE_PAYLOAD="$(
    jq -nc --arg mode "${MODE}" '{
      from:{name:"Vienna",lat:48.2082,lng:16.3738},
      to:{name:"Salzburg",lat:47.8095,lng:13.0550},
      mode:$mode,
      date:"2026-09-12",
      currency:"EUR"
    }'
  )"
  request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" "${MODE_ROUTE_PAYLOAD}"
  assert_2xx "Route estimate ${MODE}"
  if ! jq -e --arg mode "${MODE}" '
    .mode == $mode
    and .estimatedDistanceKm > 0
    and .estimatedDurationMinutes > 0
    and .estimatedCost.category == "transport"
    and .estimatedCost.currency == "EUR"
    and (.warnings | length) >= 1
  ' >/dev/null <<<"${LAST_BODY}"; then
    echo "Route estimate for ${MODE} did not include the expected deterministic estimate shape." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
done

echo "Checking internal transport search endpoint..."
TRANSPORT_SEARCH_PAYLOAD="$(
  jq -nc '{
    origin:{name:"Bratislava",lat:48.1486,lng:17.1077,country:"Slovakia"},
    destination:{name:"Vienna",lat:48.2082,lng:16.3738,country:"Austria"},
    date:"2026-09-10",
    time:"09:00",
    timePreference:"depart_after",
    travelers:2,
    modes:["train","bus","car"],
    currency:"EUR",
    locale:"en",
    constraints:{maxDurationMinutes:240,maxPriceAmount:100}
  }'
)"
request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/transport/search" "${TRANSPORT_SEARCH_PAYLOAD}"
assert_status "Transport search requires internal token" "401"
request_with_internal_token POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/transport/search" "${INTERNAL_SERVICE_TOKEN_FOR_SMOKE}" "${TRANSPORT_SEARCH_PAYLOAD}"
assert_2xx "Transport search"
if ! jq -e '
  (.options | length) >= 1
  and .summary.origin == "Bratislava"
  and .summary.destination == "Vienna"
  and (.summary.searchedModes | index("train"))
  and (.options[] | select(.mode == "train") | .durationMinutes > 0 and .estimatedPrice.currency == "EUR")
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Transport search did not return the expected option shape." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking mock weather forecast..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/weather/forecast?destination=Rome&startDate=2026-08-10&days=3"
assert_2xx "Weather forecast"

WEATHER_PROVIDER_NAME="$(jq -r '.provider // empty' <<<"${LAST_BODY}")"
WEATHER_DAY_COUNT="$(jq '.days | length' <<<"${LAST_BODY}")"
WEATHER_FIRST_DATE="$(jq -r '.days[0].date // empty' <<<"${LAST_BODY}")"
WEATHER_FIRST_CONDITION="$(jq -r '.days[0].condition // empty' <<<"${LAST_BODY}")"
if [[ "${WEATHER_PROVIDER_NAME}" != "mock" || "${WEATHER_DAY_COUNT}" -ne 3 || -z "${WEATHER_FIRST_DATE}" || -z "${WEATHER_FIRST_CONDITION}" ]]; then
  echo "Weather forecast did not include expected mock provider/day data." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking mock exchange-rate latest and conversion endpoints..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/exchange-rates/latest?base=EUR"
assert_2xx "Exchange-rate latest"
if ! jq -e '.provider == "mock" and .base == "EUR" and (.rates.JPY > 100) and (.rates.USD > 1)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Exchange-rate latest did not include expected mock EUR rates." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/exchange-rates/convert?amount=10000&from=JPY&to=EUR"
assert_2xx "Exchange-rate conversion"
if ! jq -e '.provider == "mock" and .from == "JPY" and .to == "EUR" and .amount == 10000 and (.convertedAmount > 58 and .convertedAmount < 59)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Exchange-rate conversion did not return expected mock JPY->EUR result." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking unsupported exchange-rate currency is rejected..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/exchange-rates/convert?amount=10&from=XXX&to=EUR"
assert_status "Unsupported exchange-rate currency" "400"

echo "Checking internal attraction price estimate endpoint..."
PRICE_ESTIMATE_PAYLOAD='{
  "destination": "Rome",
  "currency": "EUR",
  "date": "2026-08-10",
  "place": {
    "provider": "mock",
    "providerPlaceId": "mock-colosseum",
    "name": "Colosseum",
    "category": "landmark",
    "lat": 41.8902,
    "lng": 12.4922
  },
  "itemContext": {
    "name": "Visit the Colosseum",
    "type": "attraction"
  }
}'
request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/prices/estimate" "${PRICE_ESTIMATE_PAYLOAD}"
assert_status "Price estimate requires internal token" "401"
request_with_internal_token POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/prices/estimate" "${INTERNAL_SERVICE_TOKEN_FOR_SMOKE}" "${PRICE_ESTIMATE_PAYLOAD}"
assert_2xx "Price estimate"
if ! jq -e '.matched == true and .estimatedCost.source == "provider" and .estimatedCost.currency == "EUR" and (.estimatedCost.amount > 0) and (.matchConfidence >= 0.55)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Price estimate did not return an expected provider cost." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

PRICE_NO_MATCH_PAYLOAD='{
  "destination": "Paris",
  "currency": "EUR",
  "place": {
    "name": "Luxembourg Gardens",
    "category": "park"
  },
  "itemContext": {
    "name": "Walk through Luxembourg Gardens",
    "type": "walk"
  }
}'
request_with_internal_token POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/prices/estimate" "${INTERNAL_SERVICE_TOKEN_FOR_SMOKE}" "${PRICE_NO_MATCH_PAYLOAD}"
assert_2xx "Price estimate no-match"
if ! jq -e '.matched == false and .estimatedCost == null and (.matchConfidence >= 0)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Price estimate no-match response was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ "${SMOKE_EXPECT_OBSERVABILITY:-true}" == "true" ]]; then
  echo "Checking external provider metrics after provider calls..."
  assert_metrics_contains "External provider metrics" "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/metrics" "external_provider_requests_total"
  assert_metrics_contains "External provider cache metrics" "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/metrics" "external_provider_cache_misses_total"
fi

# Optional real-provider checks. These only run when the operator has opted into
# a real provider AND supplied an API key in the shell environment. Missing keys
# must never fail the default mock smoke test.
if [[ "${ROUTE_PROVIDER:-mock}" == "ors" && -n "${ORS_API_KEY:-}" ]]; then
  echo "Checking real ORS route provider (real result or mock fallback)..."
  request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" "${ROUTE_PAYLOAD}"
  assert_2xx "ORS route estimate"
  if ! jq -e '(.provider == "ors") or (.provider == "mock" and .fallbackUsed == true)' >/dev/null <<<"${LAST_BODY}"; then
    echo "ORS route estimate did not return a real or fallback provider result." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  echo "ORS route estimate reported provider '$(jq -r '.provider' <<<"${LAST_BODY}")'."
fi

if [[ "${WEATHER_PROVIDER:-mock}" == "openweathermap" && -n "${OPENWEATHER_API_KEY:-}" ]]; then
  echo "Checking real OpenWeather provider (real result or mock fallback)..."
  request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/weather/forecast?destination=Rome&startDate=2026-08-10&days=3"
  assert_2xx "OpenWeather forecast"
  if ! jq -e '(.provider == "openweathermap") or (.provider == "mock" and .fallbackUsed == true)' >/dev/null <<<"${LAST_BODY}"; then
    echo "OpenWeather forecast did not return a real or fallback provider result." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  echo "OpenWeather forecast reported provider '$(jq -r '.provider' <<<"${LAST_BODY}")'."
fi

echo "Web App URL: ${WEB_APP_URL}"
if request GET "${WEB_APP_URL}"; then
  if [[ "${LAST_STATUS}" =~ ^2 ]]; then
    echo "Web App responded successfully."
  else
    echo "Web App returned HTTP ${LAST_STATUS}; continuing with API smoke test."
  fi
else
  echo "Web App is not reachable; continuing with API smoke test."
fi

echo "Registering and logging in smoke test user..."
AUTH_EMAIL="${SMOKE_AUTH_EMAIL:-smoke+$(date +%s)-$$@example.com}"
AUTH_PASSWORD="StrongPassword123!"
AUTH_PAYLOAD="$(jq -nc --arg email "${AUTH_EMAIL}" --arg password "${AUTH_PASSWORD}" '{email:$email,password:$password}')"

request POST "${AUTH_SERVICE_URL}/auth/register" "${AUTH_PAYLOAD}"
assert_2xx "Auth register"

request POST "${AUTH_SERVICE_URL}/auth/login" "${AUTH_PAYLOAD}"
assert_2xx "Auth login"

ACCESS_TOKEN="$(jq -r '.accessToken // empty' <<<"${LAST_BODY}")"
REFRESH_TOKEN="$(jq -r '.refreshToken // empty' <<<"${LAST_BODY}")"
OWNER_USER_ID="$(jq -r '.user.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ACCESS_TOKEN}" || -z "${REFRESH_TOKEN}" ]]; then
  echo "Auth login response did not include both tokens." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

COLLAB_EMAIL="smoke-collab+$(date +%s)-$$@example.com"
COLLAB_PAYLOAD="$(jq -nc --arg email "${COLLAB_EMAIL}" --arg password "${AUTH_PASSWORD}" '{email:$email,password:$password}')"
echo "Registering collaborator smoke test user..."
request POST "${AUTH_SERVICE_URL}/auth/register" "${COLLAB_PAYLOAD}"
assert_2xx "Collaborator auth register"

request POST "${AUTH_SERVICE_URL}/auth/login" "${COLLAB_PAYLOAD}"
assert_2xx "Collaborator auth login"

COLLAB_ACCESS_TOKEN="$(jq -r '.accessToken // empty' <<<"${LAST_BODY}")"
COLLAB_USER_ID="$(jq -r '.user.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${COLLAB_ACCESS_TOKEN}" || -z "${COLLAB_USER_ID}" ]]; then
  echo "Collaborator login response did not include token and user id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking /auth/me..."
request_with_bearer GET "${AUTH_SERVICE_URL}/auth/me" "${ACCESS_TOKEN}"
assert_2xx "Auth me"

AUTH_ME_EMAIL="$(jq -r '.email // empty' <<<"${LAST_BODY}")"
AUTH_ME_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ "${AUTH_ME_EMAIL}" != "${AUTH_EMAIL}" ]]; then
  echo "Expected /auth/me email ${AUTH_EMAIL}, got ${AUTH_ME_EMAIL}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ -z "${AUTH_ME_ID}" ]]; then
  echo "/auth/me did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking availability search requires auth..."
AVAILABILITY_ITEM_NAME="Visit Colosseum availability smoke ${AUTH_ME_ID}"
AVAILABILITY_PAYLOAD="$(
  jq -nc --arg itemName "${AVAILABILITY_ITEM_NAME}" '{
    destination: "Rome",
    date: "2026-08-10",
    currency: "EUR",
    travelers: {adults: 2, children: 0},
    item: {
      name: $itemName,
      type: "attraction",
      startTime: "10:00",
      place: {
        name: "Colosseum",
        address: "Piazza del Colosseo, Rome",
        lat: 41.8902,
        lng: 12.4922,
        provider: "mock",
        providerPlaceId: "mock-colosseum"
      },
      estimatedCost: {
        amount: 18,
        currency: "EUR",
        category: "ticket",
        source: "estimated",
        confidence: "medium"
      }
    }
  }'
)"
request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/availability/search" "${AVAILABILITY_PAYLOAD}"
assert_status "Availability search requires auth" "401"

echo "Checking availability search with authenticated user..."
request_with_bearer POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/availability/search" "${ACCESS_TOKEN}" "${AVAILABILITY_PAYLOAD}"
assert_2xx "Availability search"
if ! jq -e '
  (.status == "available" or .status == "limited" or .status == "unavailable" or .status == "unknown")
  and ((.provider // "") | length > 0)
  and ((.providerDisplayName // "") | length > 0)
  and ((.checkedAt // "") | length > 0)
  and (.match.confidence >= 0)
  and (.options | type == "array")
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Availability search response did not include the normalized shape." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if jq -e '(.status == "available" or .status == "limited") and (.options | length > 0)' >/dev/null <<<"${LAST_BODY}"; then
  if ! jq -e '
    (.options[0].availability == "available" or .options[0].availability == "limited")
    and (.options[0].price.amount > 0)
    and (.options[0].price.currency == "EUR")
    and (.options[0].priceType != "")
    and (.options[0].bookingUrl | startswith("https://"))
  ' >/dev/null <<<"${LAST_BODY}"; then
    echo "Availability search did not include a safe bookable option." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi

echo "Checking availability cache behavior..."
request_with_bearer POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/availability/search" "${ACCESS_TOKEN}" "${AVAILABILITY_PAYLOAD}"
assert_2xx "Availability search cache repeat"
if [[ "${AVAILABILITY_CACHE_ENABLED:-true}" != "false" ]]; then
  if ! jq -e '.cached == true and ((.cacheExpiresAt // "") | length > 0)' >/dev/null <<<"${LAST_BODY}"; then
    echo "Expected repeated availability search to be served from cache." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
else
  echo "Availability cache disabled in this environment; skipping cache-hit assertion."
fi

echo "Checking availability for an unsupported item type returns unknown without a bookable option..."
AVAILABILITY_UNSUPPORTED_PAYLOAD="$(
  jq -nc '{
    destination: "Rome",
    date: "2026-08-10",
    currency: "EUR",
    travelers: {adults: 2, children: 0},
    item: {name: "Lunch break near the forum", type: "rest", startTime: "12:30"}
  }'
)"
request_with_bearer POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/availability/search" "${ACCESS_TOKEN}" "${AVAILABILITY_UNSUPPORTED_PAYLOAD}"
assert_2xx "Availability search unsupported item"
if ! jq -e '.status == "unknown" and (.options | length) == 0' >/dev/null <<<"${LAST_BODY}"; then
  echo "Expected unsupported item type to return unknown status with no options." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

# Optional real-provider smoke. Provider data changes constantly, so assert only
# on the normalized shape and provider label — never specific event names/prices.
if [[ "${AVAILABILITY_PROVIDER:-mock}" == "ticketmaster" && -n "${TICKETMASTER_API_KEY:-}" ]]; then
  echo "Checking Ticketmaster availability provider (shape-only assertions)..."
  TICKETMASTER_PAYLOAD="$(
    jq -nc '{
      destination: "London",
      date: "2026-09-10",
      currency: "GBP",
      travelers: {adults: 2, children: 0},
      item: {name: "Live concert", type: "concert", startTime: "19:30"}
    }'
  )"
  request_with_bearer POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/availability/search" "${ACCESS_TOKEN}" "${TICKETMASTER_PAYLOAD}"
  assert_2xx "Ticketmaster availability search"
  if ! jq -e '
    ((.provider == "ticketmaster") or (.fallbackUsed == true))
    and (.status == "available" or .status == "limited" or .status == "unavailable" or .status == "unknown")
    and (.options | type == "array")
  ' >/dev/null <<<"${LAST_BODY}"; then
    echo "Ticketmaster availability search did not return the expected normalized shape." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
else
  echo "TICKETMASTER_API_KEY not set / provider not ticketmaster; skipping real-provider availability smoke."
fi

if [[ "${SMOKE_EXPECT_OBSERVABILITY:-true}" == "true" ]]; then
  echo "Checking availability metrics after availability calls..."
  assert_metrics_contains "Availability search metrics" "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/metrics" "availability_search_requests_total"
fi

if [[ "${SMOKE_EXPECT_OPS_DASHBOARD:-false}" == "true" ]]; then
  echo "Checking Ops Dashboard endpoints..."
  request_with_bearer GET "${TRIP_SERVICE_URL}/ops/jobs/summary" "${ACCESS_TOKEN}"
  assert_2xx "Trip Service ops job summary"
  request_with_bearer GET "${TRIP_SERVICE_URL}/ops/jobs?limit=5" "${ACCESS_TOKEN}"
  assert_2xx "Trip Service ops job list"
  if [[ "${SMOKE_EXPECT_WORKER_SERVICE:-true}" == "true" ]]; then
    request_with_bearer GET "${WORKER_SERVICE_URL}/ops/worker/status" "${ACCESS_TOKEN}"
    assert_2xx "Worker Service ops status"
    request_with_bearer GET "${WORKER_SERVICE_URL}/ops/queues/status" "${ACCESS_TOKEN}"
    assert_2xx "Worker Service ops queue status"
  fi
  request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/ops/providers/status" "${ACCESS_TOKEN}"
  assert_2xx "External Integrations ops provider status"

  echo "Checking provider quota endpoints..."
  request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/ops/providers/quotas" "${ACCESS_TOKEN}"
  assert_2xx "External Integrations ops provider quotas"
  if ! jq -e '.providers | type == "array"' <<<"${LAST_BODY}" >/dev/null; then
    echo "Provider quotas response is missing a providers array." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  PROVIDER_LIMITS_ENABLED_RESP="$(jq -r '.enabled' <<<"${LAST_BODY}")"
  if [[ "${PROVIDER_LIMITS_ENABLED_RESP}" == "true" ]]; then
    # A route estimate and weather forecast were already exercised above, so with
    # enforcement enabled the guard must have recorded at least one unit of usage.
    ROUTES_USED="$(jq -r '[.providers[] | select(.category=="routes") | .usedToday] | add // 0' <<<"${LAST_BODY}")"
    if [[ "${ROUTES_USED}" -lt 1 ]]; then
      echo "Expected routes usedToday >= 1 with provider limits enabled, got ${ROUTES_USED}." >&2
      echo "${LAST_BODY}" >&2
      exit 1
    fi
    AVAILABILITY_USED="$(jq -r '[.providers[] | select(.category=="availability") | .usedToday] | add // 0' <<<"${LAST_BODY}")"
    if [[ "${AVAILABILITY_USED}" -lt 1 ]]; then
      echo "Expected availability usedToday >= 1 with provider limits enabled, got ${AVAILABILITY_USED}." >&2
      echo "${LAST_BODY}" >&2
      exit 1
    fi
    echo "Provider quota usage recorded for routes: ${ROUTES_USED}"
    echo "Provider quota usage recorded for availability: ${AVAILABILITY_USED}"
  else
    echo "Provider limits disabled in this environment; skipping usage-increase assertion."
  fi

  # Provider detail endpoint (operation breakdown + 7-day history).
  request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/ops/providers/quotas/mock" "${ACCESS_TOKEN}"
  assert_2xx "External Integrations ops provider quota detail"

  # Optional: verify controlled block/fallback behavior when the operator has set
  # a very low quota. A zero-cost setup is ROUTE_PROVIDER=ors, ORS_API_KEY=dummy,
  # ORS_BASE_URL=http://127.0.0.1:9, ROUTE_PROVIDER_FALLBACK_TO_MOCK=true,
  # ORS_DAILY_QUOTA=1, PROVIDER_LIMITS_ENABLED=true: the first call consumes the
  # quota (and falls back to mock because ORS is unreachable), the second is
  # quota-exceeded and served by mock fallback.
  if [[ "${SMOKE_EXPECT_PROVIDER_QUOTA_BLOCK:-false}" == "true" ]]; then
    request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" "${ROUTE_PAYLOAD}"
    request POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/routes/estimate" "${ROUTE_PAYLOAD}"
    request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/ops/providers/quotas" "${ACCESS_TOKEN}"
    assert_2xx "External Integrations ops provider quotas after block"
    BLOCK_FALLBACK="$(jq -r '[.providers[] | select(.category=="routes") | (.blockedToday + .fallbackToday)] | add // 0' <<<"${LAST_BODY}")"
    if [[ "${BLOCK_FALLBACK}" -lt 1 ]]; then
      echo "Expected routes blocked/fallback count >= 1 after exceeding the quota, got ${BLOCK_FALLBACK}." >&2
      echo "${LAST_BODY}" >&2
      exit 1
    fi
    echo "Provider quota block/fallback recorded for routes: ${BLOCK_FALLBACK}"
  fi
fi

echo "Checking default user profile..."
request_with_bearer GET "${USER_SERVICE_URL}/users/me/profile" "${ACCESS_TOKEN}"
assert_2xx "Get default profile"

PROFILE_USER_ID="$(jq -r '.userId // empty' <<<"${LAST_BODY}")"
PROFILE_CURRENCY="$(jq -r '.preferredCurrency // empty' <<<"${LAST_BODY}")"
PROFILE_LANGUAGE="$(jq -r '.preferredLanguage // empty' <<<"${LAST_BODY}")"
if [[ "${PROFILE_USER_ID}" != "${AUTH_ME_ID}" || "${PROFILE_CURRENCY}" != "EUR" || "${PROFILE_LANGUAGE}" != "en" ]]; then
  echo "Default profile did not match expected user/default values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Updating user profile..."
UPDATE_PROFILE_PAYLOAD='{
  "displayName": "Test Traveler",
  "homeCity": "Bratislava",
  "homeCountry": "Slovakia",
  "preferredCurrency": "EUR",
  "preferredLanguage": "en"
}'
request_with_bearer PUT "${USER_SERVICE_URL}/users/me/profile" "${ACCESS_TOKEN}" "${UPDATE_PROFILE_PAYLOAD}"
assert_2xx "Update profile"

UPDATED_DISPLAY_NAME="$(jq -r '.displayName // empty' <<<"${LAST_BODY}")"
UPDATED_HOME_CITY="$(jq -r '.homeCity // empty' <<<"${LAST_BODY}")"
if [[ "${UPDATED_DISPLAY_NAME}" != "Test Traveler" || "${UPDATED_HOME_CITY}" != "Bratislava" ]]; then
  echo "Updated profile did not include expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking supported and unsupported profile languages..."
UPDATE_UK_PROFILE_PAYLOAD='{
  "displayName": "Test Traveler",
  "homeCity": "Bratislava",
  "homeCountry": "Slovakia",
  "preferredCurrency": "EUR",
  "preferredLanguage": "uk"
}'
request_with_bearer PUT "${USER_SERVICE_URL}/users/me/profile" "${ACCESS_TOKEN}" "${UPDATE_UK_PROFILE_PAYLOAD}"
assert_2xx "Update preferred language to Ukrainian"
if [[ "$(jq -r '.preferredLanguage // empty' <<<"${LAST_BODY}")" != "uk" ]]; then
  echo "Profile did not persist Ukrainian preferredLanguage." >&2
  exit 1
fi

INVALID_LANGUAGE_PAYLOAD="$(jq '.preferredLanguage = "de"' <<<"${UPDATE_UK_PROFILE_PAYLOAD}")"
request_with_bearer PUT "${USER_SERVICE_URL}/users/me/profile" "${ACCESS_TOKEN}" "${INVALID_LANGUAGE_PAYLOAD}"
assert_status "Reject unsupported preferred language" "400"

# Restore English so the remaining legacy smoke assertions remain deterministic.
request_with_bearer PUT "${USER_SERVICE_URL}/users/me/profile" "${ACCESS_TOKEN}" "${UPDATE_PROFILE_PAYLOAD}"
assert_2xx "Restore English preferred language"

echo "Checking default user preferences..."
request_with_bearer GET "${USER_SERVICE_URL}/users/me/preferences" "${ACCESS_TOKEN}"
assert_2xx "Get default preferences"

DEFAULT_PACE="$(jq -r '.pace // empty' <<<"${LAST_BODY}")"
DEFAULT_TRAVEL_STYLE_COUNT="$(jq '.travelStyles | length' <<<"${LAST_BODY}")"
if [[ "${DEFAULT_PACE}" != "balanced" || "${DEFAULT_TRAVEL_STYLE_COUNT}" -ne 0 ]]; then
  echo "Default preferences did not match expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Patching user preferences..."
PATCH_PREFERENCES_PAYLOAD='{
  "travelStyles": ["budget", "food", "hidden_gems"],
  "pace": "balanced",
  "maxWalkingKmPerDay": 8,
  "foodPreferences": ["local", "cheap"],
  "avoid": ["nightclubs"],
  "preferredTransport": ["train", "public_transport"],
  "accommodationStyle": ["budget_hotel"]
}'
request_with_bearer PATCH "${USER_SERVICE_URL}/users/me/preferences" "${ACCESS_TOKEN}" "${PATCH_PREFERENCES_PAYLOAD}"
assert_2xx "Patch preferences"

PATCHED_STYLE_COUNT="$(jq '.travelStyles | length' <<<"${LAST_BODY}")"
PATCHED_WALKING="$(jq -r '.maxWalkingKmPerDay // empty' <<<"${LAST_BODY}")"
PATCHED_AVOID="$(jq -r '.avoid[0] // empty' <<<"${LAST_BODY}")"
PATCHED_ACCOMMODATION="$(jq -r '.accommodationStyle[0] // empty' <<<"${LAST_BODY}")"
if [[ "${PATCHED_STYLE_COUNT}" -ne 3 || "${PATCHED_WALKING}" != "8" || "${PATCHED_AVOID}" != "nightclubs" || "${PATCHED_ACCOMMODATION}" != "budget_hotel" ]]; then
  echo "Patched preferences did not include expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking personalization completeness and feedback..."
request_with_bearer GET "${USER_SERVICE_URL}/users/me/preferences/completeness" "${ACCESS_TOKEN}"
assert_2xx "Get preference completeness"
if ! jq -e '.score > 0 and (.missingFields | type == "array")' >/dev/null <<<"${LAST_BODY}"; then
  echo "Preference completeness response did not include a score and missing fields." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
PERSONALIZATION_FEEDBACK_PAYLOAD='{"entityType":"destination_suggestion","entityId":"smoke-vienna","feedbackType":"too_expensive","metadata":{"destination":"Vienna","source":"trip_discovery"}}'
request_with_bearer POST "${TRIP_SERVICE_URL}/personalization/feedback" "${ACCESS_TOKEN}" "${PERSONALIZATION_FEEDBACK_PAYLOAD}"
assert_status "Submit personalization feedback" "201"
request_with_bearer GET "${TRIP_SERVICE_URL}/personalization/feedback/summary" "${ACCESS_TOKEN}"
assert_2xx "Get personalization feedback summary"
if ! jq -e '.tooExpensiveCount >= 1' >/dev/null <<<"${LAST_BODY}"; then
  echo "Personalization feedback summary did not aggregate feedback." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking route alternatives workflow..."
ROUTE_ALT_SUGGEST_PAYLOAD="$(
  jq -nc '{
    origin:{name:"Bratislava",country:"Slovakia",coordinates:{lat:48.1486,lng:17.1077}},
    prompt:"A 5-day Austria trip with nature, old towns, and train travel.",
    durationDays:5,
    startDate:"2026-09-10",
    budget:{amount:700,currency:"EUR"},
    travelers:2,
    transport:{preferredModes:["train"],avoidModes:["flight"],carAvailable:false,maxTransferHoursPerDay:4},
    tripStyles:["nature","culture","train_trip"],
    outputLanguage:"en",
    suggestionCount:3
  }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/route-alternatives/suggest" "${ACCESS_TOKEN}" "${ROUTE_ALT_SUGGEST_PAYLOAD}"
assert_status "Suggest route alternatives" "201"
ROUTE_ALT_SESSION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_FIRST_ID="$(jq -r '.alternatives[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ROUTE_ALT_SESSION_ID}" || -z "${ROUTE_ALT_FIRST_ID}" ]]; then
  echo "Route alternatives response did not include a session and first alternative." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '
  (.alternatives | length) >= 2
  and (.alternatives[0].route.stops | length) >= 1
  and (.alternatives[0].route.legs | length) >= 1
  and (.alternatives[0].scores.overallFit >= 0)
  and ((.alternatives[0].difficulty // "") | length > 0)
  and ((.alternatives[0].personalizationFit.reasons // []) | length > 0)
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Route alternatives response did not include expected route, score, and difficulty data." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ROUTE_ALT_REFINE_PAYLOAD="$(jq -nc --arg id "${ROUTE_ALT_FIRST_ID}" '{instruction:"Make it cheaper and use fewer stops.",selectedAlternativeId:$id}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/route-alternatives/sessions/${ROUTE_ALT_SESSION_ID}/refine" "${ACCESS_TOKEN}" "${ROUTE_ALT_REFINE_PAYLOAD}"
assert_status "Refine route alternatives" "201"
ROUTE_ALT_CHILD_SESSION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_CHILD_PARENT_ID="$(jq -r '.parentSessionId // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_CHILD_FIRST_ID="$(jq -r '.alternatives[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ROUTE_ALT_CHILD_SESSION_ID}" || "${ROUTE_ALT_CHILD_PARENT_ID}" != "${ROUTE_ALT_SESSION_ID}" || -z "${ROUTE_ALT_CHILD_FIRST_ID}" ]]; then
  echo "Refined route alternatives response did not include expected child session metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ROUTE_ALT_CREATE_TRIP_PAYLOAD="$(jq -nc '{title:"Smoke Austria route alternative",startDate:"2026-09-10",budget:{amount:700,currency:"EUR"},travelers:2,autoGenerateItinerary:false}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/route-alternatives/sessions/${ROUTE_ALT_CHILD_SESSION_ID}/alternatives/${ROUTE_ALT_CHILD_FIRST_ID}/create-trip" "${ACCESS_TOKEN}" "${ROUTE_ALT_CREATE_TRIP_PAYLOAD}"
assert_status "Create trip from route alternative" "201"
ROUTE_ALT_TRIP_ID="$(jq -r '.trip.id // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_TRIP_REVISION="$(jq -r '.trip.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ -z "${ROUTE_ALT_TRIP_ID}" ]]; then
  echo "Create trip from route alternative did not return a trip id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.trip.tripType == "multi_destination" and (.trip.route.stops | length) >= 2 and .trip.creationMetadata.creationSource == "route_alternative"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Trip created from route alternative did not include expected multi-destination metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ROUTE_ALT_EXISTING_PAYLOAD="$(jq -nc '{prompt:"Make this route more relaxed and avoid flights.",suggestionCount:3,useCurrentRouteAsBaseline:true,outputLanguage:"en"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/route-alternatives" "${ACCESS_TOKEN}" "${ROUTE_ALT_EXISTING_PAYLOAD}"
assert_status "Suggest existing trip route alternatives" "201"
ROUTE_ALT_EXISTING_SESSION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_EXISTING_FIRST_ID="$(jq -r '.alternatives[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ROUTE_ALT_EXISTING_SESSION_ID}" || -z "${ROUTE_ALT_EXISTING_FIRST_ID}" ]]; then
  echo "Existing trip route alternatives response did not include expected ids." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ROUTE_ALT_APPLY_PAYLOAD="$(jq -nc --argjson revision "${ROUTE_ALT_TRIP_REVISION}" '{expectedItineraryRevision:$revision,regenerateItinerary:false}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/route-alternatives/${ROUTE_ALT_EXISTING_SESSION_ID}/alternatives/${ROUTE_ALT_EXISTING_FIRST_ID}/apply" "${ACCESS_TOKEN}" "${ROUTE_ALT_APPLY_PAYLOAD}"
assert_2xx "Apply route alternative"
if ! jq -e '.tripType == "multi_destination" and (.route.stops | length) >= 1' >/dev/null <<<"${LAST_BODY}"; then
  echo "Applying route alternative did not return an updated route." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/activity?limit=20" "${ACCESS_TOKEN}"
assert_2xx "Route alternative activity"
assert_activity_has "Route alternative activity" "route_alternative_applied"

ROUTE_ALT_POLL_PAYLOAD="$(jq -nc '{title:"Which smoke route should we choose?"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/route-alternatives/${ROUTE_ALT_EXISTING_SESSION_ID}/create-poll" "${ACCESS_TOKEN}" "${ROUTE_ALT_POLL_PAYLOAD}"
assert_status "Create route alternatives poll" "201"
ROUTE_ALT_POLL_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
ROUTE_ALT_POLL_OPTION_ID="$(jq -r '.options[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ROUTE_ALT_POLL_ID}" || -z "${ROUTE_ALT_POLL_OPTION_ID}" ]]; then
  echo "Route alternatives poll did not return a poll and option id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
ROUTE_ALT_POLL_VOTE_PAYLOAD="$(jq -nc --arg optionId "${ROUTE_ALT_POLL_OPTION_ID}" '{optionIds:[$optionId]}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/polls/${ROUTE_ALT_POLL_ID}/vote" "${ACCESS_TOKEN}" "${ROUTE_ALT_POLL_VOTE_PAYLOAD}"
assert_2xx "Vote route alternatives poll"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${ROUTE_ALT_TRIP_ID}/group-preferences" "${ACCESS_TOKEN}"
assert_2xx "Route alternative group preferences"
if ! jq -e --arg sessionId "${ROUTE_ALT_EXISTING_SESSION_ID}" '
  .aiConstraints.preferredRouteSessionId == $sessionId
  and ((.routeAlternativeVotes // []) | length) >= 1
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Group preferences did not include the preferred route alternative." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking default notification preferences..."
request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${ACCESS_TOKEN}"
assert_2xx "Get default notification preferences"
NOTIFICATION_PREF_COUNT="$(jq '.items | length' <<<"${LAST_BODY}")"
DEFAULT_IN_APP_COMMENTS="$(jq -r '.items[] | select(.channel == "in_app" and .category == "comments") | .enabled' <<<"${LAST_BODY}")"
DEFAULT_EMAIL_TRIP_UPDATES="$(jq -r '.items[] | select(.channel == "email" and .category == "trip_updates") | .deliveryMode' <<<"${LAST_BODY}")"
DEFAULT_PUSH_TRIP_UPDATES="$(jq -r '.items[] | select(.channel == "push" and .category == "trip_updates") | .deliveryMode' <<<"${LAST_BODY}")"
if [[ "${NOTIFICATION_PREF_COUNT}" -lt 48 || "${DEFAULT_IN_APP_COMMENTS}" != "true" || "${DEFAULT_EMAIL_TRIP_UPDATES}" != "daily_digest" || "${DEFAULT_PUSH_TRIP_UPDATES}" != "muted" ]]; then
  echo "Default notification preferences did not match expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking digest schedule, quiet hours, and trip mute controls..."
NOISE_CONTROL_PREFS='{"items":[{"channel":"email","category":"comments","deliveryMode":"daily_digest"}],"settings":{"quietHoursEnabled":true,"quietHoursStart":"22:00","quietHoursEnd":"08:00","quietHoursTimezone":"Europe/Bratislava","urgentBypassesQuietHours":true,"dailyDigestTime":"08:00","weeklyDigestDay":1,"weeklyDigestTime":"08:00"}}'
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${ACCESS_TOKEN}" "${NOISE_CONTROL_PREFS}"
assert_2xx "Save notification digest and quiet-hours settings"
if ! jq -e '.settings.quietHoursEnabled == true and (.items[] | select(.channel == "email" and .category == "comments") | .deliveryMode == "daily_digest")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Digest or quiet-hours settings were not saved." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
TRIP_COMMENT_MUTE="$(jq -nc --arg tripId "${TRIP_ID}" '{tripId:$tripId,category:"comments",mutedUntil:null}')"
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/trip-mutes" "${ACCESS_TOKEN}" "${TRIP_COMMENT_MUTE}"
assert_2xx "Mute trip comments"
TRIP_COMMENT_MUTE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/trip-mutes?tripId=${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "List trip notification mutes"
if ! jq -e '.items[] | select(.category == "comments")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Trip comment mute was not returned." >&2
  exit 1
fi
request_with_bearer DELETE "${NOTIFICATION_SERVICE_URL}/notifications/trip-mutes/${TRIP_COMMENT_MUTE_ID}" "${ACCESS_TOKEN}"
assert_2xx "Unmute trip comments"
request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/digests/pending" "${ACCESS_TOKEN}"
assert_2xx "List pending notification digests"

if [[ -n "${SMOKE_INTERNAL_SERVICE_TOKEN:-}" ]]; then
  echo "Checking grouped digest delivery, mute decisions, and urgent bypass..."
  DIGEST_KEY="trip:${TRIP_ID}:comments"
  for DIGEST_INDEX in 1 2 3; do
    DIGEST_EVENT_PAYLOAD="$(jq -nc \
      --arg userId "${OWNER_USER_ID}" --arg tripId "${TRIP_ID}" \
      --arg digestKey "${DIGEST_KEY}" --arg dedupeKey "smoke:${RUN_ID:-manual}:digest:${DIGEST_INDEX}" \
      '{notifications:[{userId:$userId,tripId:$tripId,type:"comment_created",priority:"normal",category:"comments",title:"Smoke digest comment",message:"A collaborator added a comment.",digestKey:$digestKey,dedupeKey:$dedupeKey,metadata:{tripName:"Smoke Trip"}}]}')"
    request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${DIGEST_EVENT_PAYLOAD}"
    assert_2xx "Queue grouped digest event ${DIGEST_INDEX}"
    if ! jq -e '.digested >= 1 and .email.sent == 0' <<<"${LAST_BODY}" >/dev/null; then
      echo "Normal comment unexpectedly sent an instant email or was not digested." >&2
      echo "${LAST_BODY}" >&2
      exit 1
    fi
  done
  request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/digests/pending" "${ACCESS_TOKEN}"
  assert_2xx "Read grouped pending digest"
  GROUPED_DIGEST_EVENT_COUNT="$(jq --arg digestKey "${DIGEST_KEY}" '[.items[] | select(.channel == "email") | .items[] | select(.digestKey == $digestKey) | .eventCount] | add // 0' <<<"${LAST_BODY}")"
  if [[ "${GROUPED_DIGEST_EVENT_COUNT}" -lt 3 ]]; then
    echo "Pending email digest did not group all three comment events." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/process-digests" "${SMOKE_INTERNAL_SERVICE_TOKEN}" '{"now":"2099-01-01T00:00:00Z","limit":100}'
  assert_2xx "Process pending notification digests"
  if ! jq -e '.processed >= 1 and .sent >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Digest processor did not send a grouped digest." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/trip-mutes" "${ACCESS_TOKEN}" "${TRIP_COMMENT_MUTE}"
  assert_2xx "Re-enable trip comment mute for delivery check"
  TRIP_COMMENT_MUTE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
  MUTED_COMMENT_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" --arg tripId "${TRIP_ID}" --arg dedupeKey "smoke:${RUN_ID:-manual}:muted" '{notifications:[{userId:$userId,tripId:$tripId,type:"comment_created",priority:"normal",category:"comments",title:"Muted comment",message:"A collaborator added a comment.",digestKey:("trip:"+$tripId+":comments"),dedupeKey:$dedupeKey,metadata:{}}]}')"
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${MUTED_COMMENT_PAYLOAD}"
  assert_2xx "Apply trip category mute"
  if ! jq -e '.created == 0 and .muted >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Trip comment mute did not suppress the notification." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  request_with_bearer DELETE "${NOTIFICATION_SERVICE_URL}/notifications/trip-mutes/${TRIP_COMMENT_MUTE_ID}" "${ACCESS_TOKEN}"
  assert_2xx "Remove trip comment mute after delivery check"

  URGENT_FAILURE_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" --arg tripId "${TRIP_ID}" --arg dedupeKey "smoke:${RUN_ID:-manual}:urgent" '{notifications:[{userId:$userId,tripId:$tripId,type:"generation_job_failed",priority:"urgent",category:"ai_generation",title:"Generation failed",message:"Your itinerary generation could not be completed. Open the trip to retry.",digestKey:("trip:"+$tripId+":ai_generation"),dedupeKey:$dedupeKey,metadata:{errorCode:"smoke_failure"}}]}')"
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${URGENT_FAILURE_PAYLOAD}"
  assert_2xx "Urgent generation failure bypasses digest defaults"
  if ! jq -e '.created == 1 and .email.sent >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Urgent generation failure was not delivered instantly." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  ALL_DAY_QUIET_PREFS='{"items":[{"channel":"email","category":"trip_updates","deliveryMode":"instant"}],"settings":{"quietHoursEnabled":true,"quietHoursStart":"00:00","quietHoursEnd":"00:00","quietHoursTimezone":"UTC","urgentBypassesQuietHours":true,"dailyDigestTime":"08:00","weeklyDigestDay":1,"weeklyDigestTime":"08:00"}}'
  request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${ACCESS_TOKEN}" "${ALL_DAY_QUIET_PREFS}"
  assert_2xx "Enable deterministic all-day quiet hours"
  QUIET_EVENT_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" --arg tripId "${TRIP_ID}" --arg dedupeKey "smoke:${RUN_ID:-manual}:quiet" '{notifications:[{userId:$userId,tripId:$tripId,type:"itinerary_updated",priority:"normal",category:"trip_updates",title:"Itinerary updated",message:"The itinerary was updated.",digestKey:("trip:"+$tripId+":trip_updates"),dedupeKey:$dedupeKey,metadata:{}}]}')"
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${QUIET_EVENT_PAYLOAD}"
  assert_2xx "Delay normal email during quiet hours"
  if ! jq -e '.delayed >= 1 and .email.sent == 0' <<<"${LAST_BODY}" >/dev/null; then
    echo "Quiet hours did not delay a normal email notification." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  SECURITY_EVENT_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" --arg tripId "${TRIP_ID}" --arg dedupeKey "smoke:${RUN_ID:-manual}:security" '{notifications:[{userId:$userId,tripId:$tripId,type:"share_security_changed",priority:"urgent",category:"security",title:"Sharing security changed",message:"Security settings for a shared trip changed.",digestKey:("trip:"+$tripId+":security"),dedupeKey:$dedupeKey,metadata:{}}]}')"
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${SECURITY_EVENT_PAYLOAD}"
  assert_2xx "Urgent security notification bypasses quiet hours"
  if ! jq -e '.created == 1 and .email.sent >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Urgent security notification did not bypass quiet hours." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
else
  echo "SMOKE_INTERNAL_SERVICE_TOKEN is not set; skipping digest delivery decision checks."
fi

NOISE_CONTROL_RESET='{"items":[{"channel":"email","category":"comments","deliveryMode":"daily_digest"},{"channel":"email","category":"trip_updates","deliveryMode":"daily_digest"}],"settings":{"quietHoursEnabled":false,"quietHoursStart":"22:00","quietHoursEnd":"08:00","quietHoursTimezone":"Europe/Bratislava","urgentBypassesQuietHours":true,"dailyDigestTime":"08:00","weeklyDigestDay":1,"weeklyDigestTime":"08:00"}}'
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${ACCESS_TOKEN}" "${NOISE_CONTROL_RESET}"
assert_2xx "Disable smoke-test quiet hours"

echo "Checking Web Push endpoint plumbing..."
request GET "${NOTIFICATION_SERVICE_URL}/notifications/push/public-key"
assert_2xx "Get push public key"
PUSH_ENABLED="$(jq -r '.enabled' <<<"${LAST_BODY}")"
FAKE_PUSH_ENDPOINT="https://push.example.test/smoke/${RUN_ID:-manual}"
FAKE_PUSH_SUBSCRIPTION="$(jq -nc --arg endpoint "${FAKE_PUSH_ENDPOINT}" '{subscription:{endpoint:$endpoint,keys:{p256dh:"smoke-p256dh",auth:"smoke-auth"}},userAgent:"smoke-test",browser:"Smoke",deviceLabel:"Smoke test"}')"
request_with_bearer POST "${NOTIFICATION_SERVICE_URL}/notifications/push/subscribe" "${ACCESS_TOKEN}" "${FAKE_PUSH_SUBSCRIPTION}"
assert_2xx "Subscribe push endpoint"
PUSH_SUBSCRIBED="$(jq -r '.subscribed' <<<"${LAST_BODY}")"
request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/push/status" "${ACCESS_TOKEN}"
assert_2xx "Push status"
if [[ "${PUSH_ENABLED}" == "true" && "${PUSH_SUBSCRIBED}" == "true" ]]; then
  if ! jq -e '.activeSubscriptions >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Expected at least one active push subscription after subscribe." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi
FAKE_PUSH_UNSUBSCRIBE="$(jq -nc --arg endpoint "${FAKE_PUSH_ENDPOINT}" '{endpoint:$endpoint}')"
request_with_bearer DELETE "${NOTIFICATION_SERVICE_URL}/notifications/push/unsubscribe" "${ACCESS_TOKEN}" "${FAKE_PUSH_UNSUBSCRIBE}"
assert_2xx "Unsubscribe push endpoint"

echo "Checking workspace create/invite/access flow..."
WORKSPACE_PAYLOAD="$(jq -nc '{name:"Smoke Workspace",description:"Smoke test workspace"}')"
request_with_bearer POST "${USER_SERVICE_URL}/workspaces" "${ACCESS_TOKEN}" "${WORKSPACE_PAYLOAD}"
assert_2xx "Create workspace"
WORKSPACE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_ID}" ]]; then
  echo "Create workspace response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

WORKSPACE_INVITE_PAYLOAD="$(jq -nc --arg email "${COLLAB_EMAIL}" '{email:$email,role:"member"}')"
request_with_bearer POST "${USER_SERVICE_URL}/workspaces/${WORKSPACE_ID}/members/invite" "${ACCESS_TOKEN}" "${WORKSPACE_INVITE_PAYLOAD}"
assert_2xx "Invite workspace member"
WORKSPACE_INVITATION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_INVITATION_ID}" ]]; then
  echo "Workspace invite response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Workspace invite notification" "${COLLAB_ACCESS_TOKEN}" "workspace_invited"

request_with_bearer GET "${USER_SERVICE_URL}/workspace-invitations" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List workspace invitations"
if ! jq -e --arg id "${WORKSPACE_INVITATION_ID}" '.invitations | any(.id == $id and .status == "pending")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Pending workspace invitation was not visible to invited user." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${USER_SERVICE_URL}/workspace-invitations/${WORKSPACE_INVITATION_ID}/accept" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Accept workspace invitation"
assert_notification_has "Workspace accepted notification" "${ACCESS_TOKEN}" "workspace_invitation_accepted"

WORKSPACE_TRIP_PAYLOAD="$(jq -nc --arg workspaceId "${WORKSPACE_ID}" '{
  destination:"Lisbon",
  startDate:"2026-09-01",
  days:2,
  budgetAmount:700,
  budgetCurrency:"EUR",
  travelers:2,
  interests:["food","culture"],
  pace:"balanced",
  workspaceId:$workspaceId
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips" "${ACCESS_TOKEN}" "${WORKSPACE_TRIP_PAYLOAD}"
assert_2xx "Create workspace trip"
WORKSPACE_TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
WORKSPACE_TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
WORKSPACE_TRIP_SCOPE="$(jq -r '.scope // empty' <<<"${LAST_BODY}")"
WORKSPACE_TRIP_WORKSPACE_ID="$(jq -r '.workspaceId // empty' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_TRIP_ID}" || "${WORKSPACE_TRIP_SCOPE}" != "workspace" || "${WORKSPACE_TRIP_WORKSPACE_ID}" != "${WORKSPACE_ID}" ]]; then
  echo "Workspace trip response did not include expected workspace metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips?scope=workspace&workspaceId=${WORKSPACE_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List workspace trips as member"
if ! jq -e --arg id "${WORKSPACE_TRIP_ID}" '.items | any(.id == $id and .scope == "workspace")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace member could not list the workspace trip." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/search?q=Lisbon&scope=workspace&workspaceId=${WORKSPACE_ID}&limit=5" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Search workspace trip as member"
if ! jq -e --arg id "${WORKSPACE_TRIP_ID}" '.items | any(.type == "trip" and .metadata.tripId == $id)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace search did not return the accessible workspace trip." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Fetch workspace trip as member"
WORKSPACE_MEMBER_CAN_EDIT="$(jq -r '.access.canEdit // false' <<<"${LAST_BODY}")"
if [[ "${WORKSPACE_MEMBER_CAN_EDIT}" != "true" ]]; then
  echo "Workspace member did not receive edit access." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

WORKSPACE_MEMBER_ITINERARY="$(jq -nc '{days:[{day:1,title:"Workspace smoke day",items:[{time:"10:00",type:"activity",name:"Workspace member edit",estimatedCost:{amount:80,currency:"EUR",category:"activity",confidence:"medium",source:"manual"}}]}]}')"
WORKSPACE_MEMBER_EDIT_PAYLOAD="$(jq -nc --argjson itinerary "${WORKSPACE_MEMBER_ITINERARY}" --argjson revision "${WORKSPACE_TRIP_REVISION}" '{itinerary:$itinerary,expectedItineraryRevision:$revision}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/itinerary" "${COLLAB_ACCESS_TOKEN}" "${WORKSPACE_MEMBER_EDIT_PAYLOAD}"
assert_2xx "Workspace member itinerary edit"
WORKSPACE_TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

echo "Checking workspace planning policy workflow..."
request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/policy" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Workspace member can view empty policy"
if ! jq -e '.policy == null and .defaults.schemaVersion == 1' <<<"${LAST_BODY}" >/dev/null; then
  echo "Empty workspace policy response did not include v1 defaults." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

WORKSPACE_POLICY_RULES="$(jq -nc '{
  schemaVersion:1,
  rules:{
    requireTripBudget:{enabled:false,severity:"warning"},
    maxTripBudget:{enabled:true,severity:"blocking",amount:50,currency:"EUR"},
    maxDailyBudget:{enabled:false,severity:"warning",amount:250,currency:"EUR"},
    maxItemCost:{enabled:false,severity:"warning",amount:100,currency:"EUR",categories:[]},
    maxAccommodationTotal:{enabled:false,severity:"warning",amount:800,currency:"EUR"},
    maxAccommodationPerNight:{enabled:false,severity:"warning",amount:120,currency:"EUR"},
    requireCostSplitting:{enabled:false,severity:"warning"},
    requireAvailabilityForTicketedItems:{enabled:false,severity:"warning"},
    maxWalkingKmPerDay:{enabled:false,severity:"warning",km:12},
    noLateActivitiesAfter:{enabled:false,severity:"warning",time:"22:00"},
    requiredRestTimePerDay:{enabled:false,severity:"info",minutes:60},
    preferredTransportModes:{enabled:true,severity:"info",modes:["train","public_transport"]},
    maxTransferHoursPerDay:{enabled:true,severity:"warning",hours:4},
    disallowedTransportModes:{enabled:true,severity:"blocking",modes:["flight"]},
    disallowedActivityTypes:{enabled:false,severity:"warning",types:[]}
  }
}')"
WORKSPACE_POLICY_PAYLOAD="$(jq -nc --argjson rules "${WORKSPACE_POLICY_RULES}" \
  '{name:"Smoke planning policy",description:"Smoke policy",rules:$rules}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/policy" "${COLLAB_ACCESS_TOKEN}" "${WORKSPACE_POLICY_PAYLOAD}"
assert_status "Workspace member cannot update policy" "403"
request_with_bearer PUT "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/policy" "${ACCESS_TOKEN}" "${WORKSPACE_POLICY_PAYLOAD}"
assert_2xx "Workspace owner creates policy"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/policy/evaluate" "${COLLAB_ACCESS_TOKEN}" '{}'
assert_2xx "Workspace member evaluates policy"
if ! jq -e '.status == "blocking" and .summary.blockingCount == 1' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected blocking workspace policy evaluation." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/submit" "${COLLAB_ACCESS_TOKEN}" '{"note":"Blocked smoke submission."}'
assert_status "Blocking workspace policy stops approval submission" "400"
if ! jq -e '.error == "workspace_policy_blocking_violation" and .evaluation.status == "blocking"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Blocking policy error did not include the evaluation payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

WORKSPACE_WARNING_POLICY_RULES="$(jq '.rules.maxTripBudget.severity = "warning"' <<<"${WORKSPACE_POLICY_RULES}")"
WORKSPACE_WARNING_POLICY_PAYLOAD="$(jq -nc --argjson rules "${WORKSPACE_WARNING_POLICY_RULES}" \
  '{name:"Smoke planning policy",description:"Smoke policy",rules:$rules}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/policy" "${ACCESS_TOKEN}" "${WORKSPACE_WARNING_POLICY_PAYLOAD}"
assert_2xx "Workspace owner changes policy violation to warning"

echo "Checking planning constraints preview..."
CONSTRAINTS_FLIGHT_ROUTE="$(
  jq -nc '{
    origin:{name:"Bratislava",country:"Slovakia"},
    stops:[
      {id:"stop_1",destination:"Vienna",country:"Austria",arrivalDate:"2026-09-10",departureDate:"2026-09-11"},
      {id:"stop_2",destination:"Salzburg",country:"Austria",arrivalDate:"2026-09-11",departureDate:"2026-09-12"}
    ],
    legs:[
      {
        id:"leg_1",
        fromStopId:"origin",
        toStopId:"stop_1",
        fromName:"Bratislava",
        toName:"Vienna",
        mode:"flight",
        departureDate:"2026-09-10",
        estimatedDurationMinutes:60,
        estimatedDistanceKm:80,
        estimatedCost:{amount:90,currency:"EUR",category:"transport",confidence:"medium",source:"mock"}
      }
    ],
    preferences:{
      preferredModes:["train"],
      avoidModes:["flight"],
      carAvailable:false,
      maxTransferHoursPerDay:4,
      tripStyles:["city_break","food"]
    }
  }'
)"
CONSTRAINTS_FLIGHT_PREVIEW_PAYLOAD="$(
  jq -nc --arg workspaceId "${WORKSPACE_ID}" --argjson route "${CONSTRAINTS_FLIGHT_ROUTE}" '{
    source:"trip_generation",
    workspaceId:$workspaceId,
    request:{
      tripType:"multi_destination",
      destination:"Austria rail route",
      outputLanguage:"uk",
      startDate:"2026-09-10",
      durationDays:3,
      budget:{amount:700,currency:"EUR",strictness:"target"},
      travelers:{count:2,type:"friends"},
      pace:"balanced",
      walking:{maxKmPerDay:8,allowLongHikes:false},
      transport:{preferredModes:["train"],avoidModes:["flight"],carAvailable:false,maxTransferHoursPerDay:4},
      tripStyles:["city_break","food"],
      route:$route
    }
  }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/planning-constraints/preview" "${ACCESS_TOKEN}" "${CONSTRAINTS_FLIGHT_PREVIEW_PAYLOAD}"
assert_2xx "Preview planning constraints with disallowed flight"
if ! jq -e '
  .constraints.schemaVersion == 1
  and .constraints.language == "uk"
  and (.constraints.transport.disallowedModes | index("flight"))
  and (.blockers | any(.type == "transport_mode_disallowed"))
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Planning constraints preview did not report the expected disallowed-flight blocker." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

CONSTRAINTS_TRAIN_ROUTE="$(jq '.legs[0].mode = "train"' <<<"${CONSTRAINTS_FLIGHT_ROUTE}")"
CONSTRAINTS_TRAIN_PREVIEW_PAYLOAD="$(
  jq -nc --arg workspaceId "${WORKSPACE_ID}" --argjson route "${CONSTRAINTS_TRAIN_ROUTE}" '{
    source:"trip_generation",
    workspaceId:$workspaceId,
    request:{
      tripType:"multi_destination",
      destination:"Austria rail route",
      outputLanguage:"uk",
      startDate:"2026-09-10",
      durationDays:3,
      budget:{amount:700,currency:"EUR",strictness:"target"},
      travelers:{count:2,type:"friends"},
      pace:"balanced",
      walking:{maxKmPerDay:8,allowLongHikes:false},
      transport:{preferredModes:["train"],avoidModes:["flight"],carAvailable:false,maxTransferHoursPerDay:4},
      tripStyles:["city_break","food"],
      route:$route
    }
  }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/planning-constraints/preview" "${ACCESS_TOKEN}" "${CONSTRAINTS_TRAIN_PREVIEW_PAYLOAD}"
assert_2xx "Preview planning constraints with allowed train"
if jq -e '.blockers | any(.type == "transport_mode_disallowed")' >/dev/null <<<"${LAST_BODY}"; then
  echo "Planning constraints preview still reported a disallowed transport blocker after switching to train." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking AI policy-aware trip repair workflow..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval-risk" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Workspace member reads approval risk before repair"
if ! jq -e '.status != "not_applicable" and (.factors | length >= 1)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected approval risk factors before repair." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

REPAIR_JOB_PAYLOAD="$(jq -nc --argjson revision "${WORKSPACE_TRIP_REVISION}" '{
  expectedItineraryRevision:$revision,
  repairMode:"reduce_budget_risk",
  selectedIssueTypes:["maxTripBudget"],
  selectedRiskFactorTypes:["workspace_policy_warning","trip_budget_exceeded"],
  constraints:{
    preserveConfirmedItems:true,
    minimizeChanges:true,
    preserveUserEditedItems:true,
    doNotChangeAccommodation:false,
    doNotChangeDates:true,
    maxChangedItems:10
  },
  specialInstructions:"Prefer lower-cost public options for the smoke test."
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-jobs" "${COLLAB_ACCESS_TOKEN}" "${REPAIR_JOB_PAYLOAD}"
assert_status "Create policy repair job" "202"
REPAIR_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${REPAIR_JOB_ID}" || "$(jq -r '.job.jobType // empty' <<<"${LAST_BODY}")" != "policy_repair" ]]; then
  echo "Policy repair job response did not include expected job metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

poll_generation_job "Policy repair" "${WORKSPACE_TRIP_ID}" "${REPAIR_JOB_ID}" "${COLLAB_ACCESS_TOKEN}"
if [[ "$(jq -r '.job.status // empty' <<<"${LAST_BODY}")" != "completed" ]]; then
  echo "Policy repair job did not complete successfully." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
REPAIR_PROPOSAL_ID="$(jq -r '.job.resultPayload.proposalId // empty' <<<"${LAST_BODY}")"
if [[ -z "${REPAIR_PROPOSAL_ID}" ]]; then
  echo "Policy repair job did not return a proposalId." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals?status=pending&limit=20" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List pending policy repair proposals"
if ! jq -e --arg id "${REPAIR_PROPOSAL_ID}" '.proposals | any(.id == $id and .status == "pending")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Pending repair proposal was not listed." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals/${REPAIR_PROPOSAL_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Get policy repair proposal detail"
if ! jq -e '.proposal.status == "pending" and (.proposal.proposal.repairedItinerary.days | length >= 1) and (.proposal.proposal.diff.itemsModified != null)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Repair proposal detail did not include repaired itinerary and diff." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

REPAIR_APPLY_PAYLOAD="$(jq -nc --argjson revision "${WORKSPACE_TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals/${REPAIR_PROPOSAL_ID}/apply" "${COLLAB_ACCESS_TOKEN}" "${REPAIR_APPLY_PAYLOAD}"
assert_2xx "Apply policy repair proposal"
REPAIR_APPLIED_REVISION="$(jq -r '.trip.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${REPAIR_APPLIED_REVISION}" -ne $((WORKSPACE_TRIP_REVISION + 1)) ]]; then
  echo "Expected repair apply to increment itineraryRevision to $((WORKSPACE_TRIP_REVISION + 1)), got ${REPAIR_APPLIED_REVISION}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.proposal.status == "applied"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Repair apply did not return an applied proposal." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
WORKSPACE_TRIP_REVISION="${REPAIR_APPLIED_REVISION}"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals/${REPAIR_PROPOSAL_ID}/apply" "${COLLAB_ACCESS_TOKEN}" "${REPAIR_APPLY_PAYLOAD}"
assert_status "Reapply policy repair proposal" "400"

WORKSPACE_STRICT_POLICY_RULES="$(jq '.rules.maxTripBudget.amount = 10' <<<"${WORKSPACE_WARNING_POLICY_RULES}")"
WORKSPACE_STRICT_POLICY_PAYLOAD="$(jq -nc --argjson rules "${WORKSPACE_STRICT_POLICY_RULES}" \
  '{name:"Smoke planning policy",description:"Smoke strict policy",rules:$rules}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/policy" "${ACCESS_TOKEN}" "${WORKSPACE_STRICT_POLICY_PAYLOAD}"
assert_2xx "Workspace owner tightens policy for stale repair check"

SECOND_REPAIR_PAYLOAD="$(jq -nc --argjson revision "${WORKSPACE_TRIP_REVISION}" '{
  expectedItineraryRevision:$revision,
  repairMode:"policy_compliance",
  selectedIssueTypes:["maxTripBudget"],
  constraints:{doNotChangeDates:true,maxChangedItems:10}
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-jobs" "${COLLAB_ACCESS_TOKEN}" "${SECOND_REPAIR_PAYLOAD}"
assert_status "Create second policy repair job" "202"
SECOND_REPAIR_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
poll_generation_job "Second policy repair" "${WORKSPACE_TRIP_ID}" "${SECOND_REPAIR_JOB_ID}" "${COLLAB_ACCESS_TOKEN}"
SECOND_REPAIR_PROPOSAL_ID="$(jq -r '.job.resultPayload.proposalId // empty' <<<"${LAST_BODY}")"
if [[ -z "${SECOND_REPAIR_PROPOSAL_ID}" ]]; then
  echo "Second policy repair job did not return a proposalId." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals/${SECOND_REPAIR_PROPOSAL_ID}/apply" "${COLLAB_ACCESS_TOKEN}" "${REPAIR_APPLY_PAYLOAD}"
assert_status "Stale policy repair apply" "409"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/repair-proposals/${SECOND_REPAIR_PROPOSAL_ID}/discard" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Discard second policy repair proposal"

echo "Checking workspace trip approval workflow..."
# Member sees the workspace trip as an approval draft and can submit it.
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Get workspace trip approval as member"
if ! jq -e '.status == "draft" and .canSubmit == true' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected workspace trip approval to be a submittable draft." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

# Member submits for approval; owner is notified.
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/submit" "${COLLAB_ACCESS_TOKEN}" '{"note":"Ready for review."}'
assert_2xx "Submit workspace trip for approval"
if ! jq -e '.status == "pending_approval"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Submit did not move approval to pending_approval." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Owner notified of approval submission" "${ACCESS_TOKEN}" "trip_submitted_for_approval"

# Owner sees the pending trip in the workspace approvals queue.
request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/approvals?status=pending_approval" "${ACCESS_TOKEN}"
assert_2xx "List workspace approvals as owner"
if ! jq -e --arg id "${WORKSPACE_TRIP_ID}" '.approvals | any(.tripId == $id and .approvalStatus == "pending_approval") and (.counts.pendingApproval >= 1)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Pending trip did not appear in the workspace approvals queue." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

# A member cannot approve; only owners/admins can.
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/approve" "${COLLAB_ACCESS_TOKEN}" '{}'
assert_status "Member cannot approve workspace trip" "403"

# Owner requests changes; the submitter is notified and can resubmit.
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/request-changes" "${ACCESS_TOKEN}" '{"decisionNote":"Please reduce accommodation cost and check availability."}'
assert_2xx "Owner requests changes"
if ! jq -e '.status == "changes_requested"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Request changes did not move approval to changes_requested." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Member notified of requested changes" "${COLLAB_ACCESS_TOKEN}" "trip_changes_requested"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Get workspace trip approval after changes requested"
if ! jq -e '.status == "changes_requested" and .canSubmit == true' <<<"${LAST_BODY}" >/dev/null; then
  echo "Member should be able to resubmit after changes were requested." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/submit" "${COLLAB_ACCESS_TOKEN}" '{"note":"Updated per feedback."}'
assert_2xx "Resubmit workspace trip for approval"

# Owner approves; the submitter is notified.
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/approve" "${ACCESS_TOKEN}" '{"decisionNote":"Looks good."}'
assert_2xx "Owner approves workspace trip"
if ! jq -e '.status == "approved"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Approve did not move approval to approved." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Member notified of approval" "${COLLAB_ACCESS_TOKEN}" "trip_approved"

# A material itinerary edit on an approved trip resets approval to draft.
WORKSPACE_RESET_ITINERARY="$(jq -nc '{days:[{day:1,title:"Reset smoke day",items:[{time:"09:00",type:"activity",name:"Edit after approval",estimatedCost:{amount:70,currency:"EUR",category:"activity",confidence:"medium",source:"manual"}}]}]}')"
WORKSPACE_RESET_EDIT_PAYLOAD="$(jq -nc --argjson itinerary "${WORKSPACE_RESET_ITINERARY}" --argjson revision "${WORKSPACE_TRIP_REVISION}" '{itinerary:$itinerary,expectedItineraryRevision:$revision}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/itinerary" "${COLLAB_ACCESS_TOKEN}" "${WORKSPACE_RESET_EDIT_PAYLOAD}"
assert_2xx "Edit approved workspace trip itinerary"
WORKSPACE_TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Get workspace trip approval after material edit"
if ! jq -e '.status == "draft"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Approval did not reset to draft after a material edit." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

# Approval history captures the decisions and the reset.
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/events" "${ACCESS_TOKEN}"
assert_2xx "List workspace trip approval events"
if ! jq -e '(.events | any(.eventType == "approved")) and (.events | any(.eventType == "reset_to_draft"))' <<<"${LAST_BODY}" >/dev/null; then
  echo "Approval history did not include the approved and reset_to_draft events." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

# Personal trips do not require approval and cannot be submitted.
APPROVAL_PERSONAL_PAYLOAD="$(jq -nc '{destination:"Lisbon",days:2,travelers:1,pace:"balanced"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips" "${ACCESS_TOKEN}" "${APPROVAL_PERSONAL_PAYLOAD}"
assert_status "Create personal trip for approval check" "201"
APPROVAL_PERSONAL_TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${APPROVAL_PERSONAL_TRIP_ID}/approval" "${ACCESS_TOKEN}"
assert_2xx "Get personal trip approval"
if ! jq -e '.status == "not_required" and .canSubmit == false' <<<"${LAST_BODY}" >/dev/null; then
  echo "Personal trip approval should be not_required with no actions." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${APPROVAL_PERSONAL_TRIP_ID}/approval/submit" "${ACCESS_TOKEN}" '{}'
assert_status "Personal trip cannot be submitted for approval" "400"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${APPROVAL_PERSONAL_TRIP_ID}/policy/evaluate" "${ACCESS_TOKEN}" '{}'
assert_2xx "Personal trip policy evaluation"
if ! jq -e '.status == "not_applicable" and .notApplicableReason == "personal_trip"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Personal trip policy evaluation was not marked not_applicable." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking workspace trip template flow..."
WORKSPACE_TEMPLATE_PAYLOAD="$(jq -nc --arg workspaceId "${WORKSPACE_ID}" '{
  title:"Smoke workspace template",
  description:"Reusable workspace smoke itinerary",
  visibility:"workspace",
  workspaceId:$workspaceId,
  destinationHint:"Paris",
  defaultCurrency:"EUR",
  tags:["workspace","smoke"]
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/templates" "${ACCESS_TOKEN}" "${WORKSPACE_TEMPLATE_PAYLOAD}"
assert_status "Save workspace trip template" "201"
WORKSPACE_TEMPLATE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_TEMPLATE_ID}" ]]; then
  echo "Workspace template response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/templates" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List workspace templates as member"
if ! jq -e --arg id "${WORKSPACE_TEMPLATE_ID}" '.templates | any(.id == $id and .visibility == "workspace")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace member could not list workspace template." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
CREATE_FROM_WORKSPACE_TEMPLATE_PAYLOAD="$(jq -nc --arg workspaceId "${WORKSPACE_ID}" '{
  title:"Smoke workspace trip from template",
  destination:"Paris",
  startDate:"2026-10-01",
  workspaceId:$workspaceId,
  travelers:2,
  pace:"balanced"
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${WORKSPACE_TEMPLATE_ID}/create-trip" "${COLLAB_ACCESS_TOKEN}" "${CREATE_FROM_WORKSPACE_TEMPLATE_PAYLOAD}"
assert_status "Create workspace trip from template as member" "201"
WORKSPACE_TEMPLATE_TRIP_WORKSPACE_ID="$(jq -r '.workspaceId // empty' <<<"${LAST_BODY}")"
if [[ "${WORKSPACE_TEMPLATE_TRIP_WORKSPACE_ID}" != "${WORKSPACE_ID}" ]]; then
  echo "Workspace template-created trip did not target expected workspace." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${USER_SERVICE_URL}/workspaces/${WORKSPACE_ID}/members" "${ACCESS_TOKEN}"
assert_2xx "List workspace members"
WORKSPACE_MEMBER_ID="$(jq -r --arg userId "${COLLAB_USER_ID}" '.members[] | select(.userId == $userId) | .id' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_MEMBER_ID}" ]]; then
  echo "Accepted workspace member was not present in member list." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer PATCH "${USER_SERVICE_URL}/workspaces/${WORKSPACE_ID}/members/${WORKSPACE_MEMBER_ID}" "${ACCESS_TOKEN}" '{"role":"viewer"}'
assert_2xx "Change workspace member to viewer"
assert_notification_has "Workspace role changed notification" "${COLLAB_ACCESS_TOKEN}" "workspace_role_changed"

WORKSPACE_VIEWER_EDIT_PAYLOAD="$(jq -nc --argjson itinerary "${WORKSPACE_MEMBER_ITINERARY}" --argjson revision "${WORKSPACE_TRIP_REVISION}" '{itinerary:$itinerary,expectedItineraryRevision:$revision}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/itinerary" "${COLLAB_ACCESS_TOKEN}" "${WORKSPACE_VIEWER_EDIT_PAYLOAD}"
assert_status "Workspace viewer itinerary edit" "403"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${WORKSPACE_TEMPLATE_ID}/create-trip" "${COLLAB_ACCESS_TOKEN}" "${CREATE_FROM_WORKSPACE_TEMPLATE_PAYLOAD}"
assert_status "Workspace viewer create trip from template" "403"

# A workspace viewer can read approval state but cannot submit or approve.
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Workspace viewer can view approval state"
if ! jq -e '.canSubmit == false and .canApprove == false' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace viewer should not be able to submit or approve." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}/approval/submit" "${COLLAB_ACCESS_TOKEN}" '{}'
assert_status "Workspace viewer cannot submit for approval" "403"

echo "Checking workspace shared budget flow..."
WORKSPACE_BUDGET_PAYLOAD="$(jq -nc '{
  name:"Smoke shared budget",
  description:"Smoke test workspace budget",
  amount:100,
  currency:"EUR",
  periodStart:"2026-01-01",
  periodEnd:"2026-12-31",
  isPrimary:true
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/budgets" "${ACCESS_TOKEN}" "${WORKSPACE_BUDGET_PAYLOAD}"
assert_2xx "Create workspace budget"
WORKSPACE_BUDGET_ID="$(jq -r '.budget.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${WORKSPACE_BUDGET_ID}" ]]; then
  echo "Workspace budget response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/budgets/${WORKSPACE_BUDGET_ID}/summary" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Workspace viewer budget summary"
if ! jq -e '.summary.tripCount >= 1 and .summary.estimatedTotal > 0 and (.summary.utilizationPercent != null)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace budget summary did not include expected utilization data." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer PATCH "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/budgets/${WORKSPACE_BUDGET_ID}" "${COLLAB_ACCESS_TOKEN}" '{"amount":90}'
assert_status "Workspace viewer budget patch" "403"

echo "Checking workspace viewer can read workspace cost analytics..."
request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/analytics/costs?currency=EUR&from=2026-01-01&to=2026-12-31" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Workspace viewer cost analytics"
if ! jq -e --arg id "${WORKSPACE_TRIP_ID}" '.summary.tripCount >= 1 and (.byTrip | any(.tripId == $id)) and (.byCategory | length >= 1) and (.activeBudget.id != null)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Workspace cost analytics did not include the expected workspace trip." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/budgets/${WORKSPACE_BUDGET_ID}/archive" "${ACCESS_TOKEN}" '{"reason":"Smoke test completed"}'
assert_2xx "Archive workspace budget"
request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/budgets?status=active" "${ACCESS_TOKEN}"
assert_2xx "List active workspace budgets after archive"
if jq -e --arg id "${WORKSPACE_BUDGET_ID}" '.budgets | any(.id == $id)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Archived workspace budget still appeared in active budget list." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer DELETE "${USER_SERVICE_URL}/workspaces/${WORKSPACE_ID}/members/${WORKSPACE_MEMBER_ID}" "${ACCESS_TOKEN}"
assert_2xx "Remove workspace member"
assert_notification_has "Workspace removed notification" "${COLLAB_ACCESS_TOKEN}" "workspace_member_removed"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${WORKSPACE_TRIP_ID}" "${COLLAB_ACCESS_TOKEN}"
if [[ "${LAST_STATUS}" != "403" && "${LAST_STATUS}" != "404" ]]; then
  echo "Removed workspace member still had access to workspace trip; expected 403 or 404, got HTTP ${LAST_STATUS}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/workspaces/${WORKSPACE_ID}/analytics/costs?currency=EUR" "${COLLAB_ACCESS_TOKEN}"
assert_status "Removed workspace member cost analytics" "403"

echo "Checking optional AI Planning destination context endpoint..."
if request GET "${AI_PLANNING_SERVICE_URL}/destination-context"; then
  if [[ "${LAST_STATUS}" =~ ^2 ]]; then
    echo "Destination context endpoint is available."
  elif [[ "${LAST_STATUS}" == "404" ]]; then
    echo "Destination context endpoint is not available; skipping."
  else
    echo "Destination context check returned HTTP ${LAST_STATUS}; continuing."
  fi
else
  echo "Destination context check failed; continuing."
fi

echo "Checking optional knowledge search..."
KNOWLEDGE_PAYLOAD='{"destination":"Rome","interests":["food","history"],"query":"local food and historic sights","topK":3}'
if request POST "${AI_PLANNING_SERVICE_URL}/knowledge/search" "${KNOWLEDGE_PAYLOAD}"; then
  if [[ "${LAST_STATUS}" =~ ^2 ]]; then
    RESULT_COUNT="$(jq '.items | length' <<<"${LAST_BODY}")"
    echo "Knowledge search returned ${RESULT_COUNT} item(s)."
  else
    echo "Knowledge search returned HTTP ${LAST_STATUS}; continuing."
  fi
else
  echo "Knowledge search request failed; continuing."
fi

echo "Creating a multi-destination route trip..."
MULTI_ROUTE_JSON="$(
  jq -nc '{
    origin:{
      name:"Bratislava",
      country:"Slovakia",
      coordinates:{lat:48.1486,lng:17.1077}
    },
    returnToOrigin:false,
    stops:[
      {
        id:"stop_1",
        destination:"Vienna",
        city:"Vienna",
        country:"Austria",
        arrivalDate:"2026-09-10",
        departureDate:"2026-09-12",
        nights:2,
        coordinates:{lat:48.2082,lng:16.3738},
        accommodationHint:"hotel"
      },
      {
        id:"stop_2",
        destination:"Salzburg",
        city:"Salzburg",
        country:"Austria",
        arrivalDate:"2026-09-12",
        departureDate:"2026-09-14",
        nights:2,
        coordinates:{lat:47.8095,lng:13.0550},
        accommodationHint:"guesthouse"
      }
    ],
    legs:[
      {
        id:"leg_1",
        fromStopId:"origin",
        toStopId:"stop_1",
        fromName:"Bratislava",
        toName:"Vienna",
        mode:"train",
        departureDate:"2026-09-10",
        estimatedDurationMinutes:70,
        estimatedDistanceKm:80,
        estimatedCost:{amount:18,currency:"EUR",category:"transport",confidence:"medium",source:"mock"},
        notes:"Direct regional train estimate; verify schedules before travel."
      },
      {
        id:"leg_2",
        fromStopId:"stop_1",
        toStopId:"stop_2",
        fromName:"Vienna",
        toName:"Salzburg",
        mode:"train",
        departureDate:"2026-09-12",
        estimatedDurationMinutes:150,
        estimatedDistanceKm:295,
        estimatedCost:{amount:35,currency:"EUR",category:"transport",confidence:"medium",source:"mock"},
        notes:"Intercity train estimate; verify schedules before travel."
      }
    ],
    preferences:{
      preferredModes:["train","public_transport"],
      avoidModes:["flight"],
      carAvailable:false,
      maxTransferHoursPerDay:4,
      tripStyles:["train_trip","city_break"]
    }
  }'
)"
MULTI_TRIP_PAYLOAD="$(
  jq -nc --argjson route "${MULTI_ROUTE_JSON}" '{
    tripType:"multi_destination",
    destination:"Austria route",
    startDate:"2026-09-10",
    days:5,
    budgetAmount:900,
    budgetCurrency:"EUR",
    travelers:2,
    interests:["culture","food"],
    pace:"balanced",
    route:$route
  }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips" "${ACCESS_TOKEN}" "${MULTI_TRIP_PAYLOAD}"
assert_2xx "Create multi-destination trip"
MULTI_TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
MULTI_TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ -z "${MULTI_TRIP_ID}" || "${MULTI_TRIP_REVISION}" != "0" ]]; then
  echo "Multi-destination trip response did not include id and itineraryRevision=0." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.tripType == "multi_destination" and (.route.stops | length) == 2 and .route.preferences.tripStyles[0] == "train_trip"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Multi-destination trip did not echo expected route metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}/route" "${ACCESS_TOKEN}"
assert_2xx "Get multi-destination trip route"
if ! jq -e '.route.origin.name == "Bratislava" and (.route.legs | length) == 2 and .route.legs[1].mode == "train"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Route endpoint did not return the expected stored route." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Queueing multi-destination itinerary generation job..."
MULTI_GENERATE_PAYLOAD="$(jq -nc --argjson revision "${MULTI_TRIP_REVISION}" '{jobType:"full_generation",expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}/generation-jobs" "${ACCESS_TOKEN}" "${MULTI_GENERATE_PAYLOAD}"
assert_status "Create multi-destination generation job" "202"
MULTI_GENERATION_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${MULTI_GENERATION_JOB_ID}" ]]; then
  echo "Multi-destination generation job response did not include a job id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
poll_generation_job "Multi-destination generation" "${MULTI_TRIP_ID}" "${MULTI_GENERATION_JOB_ID}" "${ACCESS_TOKEN}"
if [[ "$(jq -r '.job.status // empty' <<<"${LAST_BODY}")" != "completed" ]]; then
  echo "Multi-destination generation job did not complete." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch generated multi-destination trip"
MULTI_TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if ! jq -e '
  .status == "COMPLETED"
  and (.itinerary.days | length) == 5
  and ([.itinerary.days[]?.items[]? | select(.type == "transfer" and .transfer.mode == "train")] | length) >= 1
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Generated multi-destination trip did not include expected transfer itinerary items." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Multi-destination budget summary"
if ! jq -e '.tripBudget == 900 and (.byCategory | any(.category == "transport" and .estimatedTotal > 0 and .itemCount >= 1))' >/dev/null <<<"${LAST_BODY}"; then
  echo "Multi-destination budget summary did not include transfer transport costs." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Updating a multi-destination route leg..."
UPDATED_MULTI_ROUTE_JSON="$(
  jq -c '
    .legs[1].mode = "car"
    | .legs[1].estimatedDurationMinutes = 240
    | .legs[1].estimatedCost = {amount:53.1,currency:"EUR",category:"transport",confidence:"medium",source:"mock"}
    | .preferences.preferredModes = ["car"]
    | .preferences.carAvailable = true
    | .preferences.tripStyles = ["road_trip"]
  ' <<<"${MULTI_ROUTE_JSON}"
)"
UPDATE_MULTI_ROUTE_PAYLOAD="$(jq -nc --argjson route "${UPDATED_MULTI_ROUTE_JSON}" --argjson revision "${MULTI_TRIP_REVISION}" '{route:$route,expectedItineraryRevision:$revision}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}/route" "${ACCESS_TOKEN}" "${UPDATE_MULTI_ROUTE_PAYLOAD}"
assert_2xx "Update multi-destination trip route"
if ! jq -e '.tripType == "multi_destination" and .route.legs[1].mode == "car" and .route.preferences.carAvailable == true' >/dev/null <<<"${LAST_BODY}"; then
  echo "Route update did not persist the changed car leg." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${MULTI_TRIP_ID}/activity?limit=20" "${ACCESS_TOKEN}"
assert_2xx "Multi-destination route activity"
assert_activity_has "Multi-destination route activity" "route_updated"

echo "Creating a trip with Authorization header..."
CREATE_TRIP_PAYLOAD='{
  "destination": "Rome",
  "startDate": "2026-08-10",
  "days": 2,
  "budgetAmount": 500,
  "budgetCurrency": "EUR",
  "travelers": 2,
  "interests": ["food", "history", "hidden_gems"],
  "pace": "balanced"
}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips" "${ACCESS_TOKEN}" "${CREATE_TRIP_PAYLOAD}"
assert_2xx "Create trip"

TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${TRIP_ID}" ]]; then
  echo "Create trip response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${TRIP_REVISION}" != "0" ]]; then
  echo "Expected new trip itineraryRevision=0, got ${TRIP_REVISION}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
echo "Created trip ${TRIP_ID}."

echo "Checking accommodation endpoints and budget integration..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${ACCESS_TOKEN}"
assert_2xx "Get empty accommodation"
if ! jq -e '.accommodation == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected new trip accommodation to be null." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ACCOMMODATION_PAYLOAD="$(jq -nc '{
  accommodation: {
    name: "Hotel Roma",
    type: "hotel",
    address: "Via Roma 10",
    place: {
      provider: "mock",
      providerPlaceId: "mock-hotel-roma",
      name: "Hotel Roma",
      address: "Via Roma 10",
      latitude: 41.9028,
      longitude: 12.4964,
      category: "hotel"
    },
    checkInDate: "2026-08-10",
    checkOutDate: "2026-08-12",
    estimatedCost: {
      amount: 120,
      currency: "eur",
      category: "food",
      source: "ai",
      note: "Smoke total stay cost"
    },
    notes: "Smoke test stay"
  }
}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${ACCESS_TOKEN}" "${ACCOMMODATION_PAYLOAD}"
assert_2xx "Update accommodation"
if ! jq -e '.accommodation.name == "Hotel Roma" and .accommodation.estimatedCost.currency == "EUR" and .accommodation.estimatedCost.category == "accommodation" and .accommodation.estimatedCost.source == "manual"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Accommodation update did not normalize expected fields." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip after accommodation update"
REVISION_AFTER_ACCOMMODATION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${REVISION_AFTER_ACCOMMODATION}" != "${TRIP_REVISION}" ]]; then
  echo "Accommodation update unexpectedly changed itineraryRevision (${TRIP_REVISION} -> ${REVISION_AFTER_ACCOMMODATION})." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.accommodation.name == "Hotel Roma"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Trip detail did not include accommodation." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Budget summary with accommodation"
if ! jq -e '.accommodationTotal == 120 and (.byCategory | any(.category == "accommodation" and .estimatedTotal == 120))' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget summary did not include accommodation cost." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking owner trip presence state and snapshot endpoints..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${ACCESS_TOKEN}" '{"state":"away"}'
assert_status "Trip presence invalid state" "400"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${ACCESS_TOKEN}" '{"state":"viewing"}'
assert_2xx "Owner trip presence viewing state"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${ACCESS_TOKEN}" '{"state":"editing"}'
assert_2xx "Owner trip presence editing state"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence" "${ACCESS_TOKEN}"
assert_2xx "Owner trip presence snapshot"
PRESENCE_TRIP_ID="$(jq -r '.tripId // empty' <<<"${LAST_BODY}")"
if [[ "${PRESENCE_TRIP_ID}" != "${TRIP_ID}" ]]; then
  echo "Presence snapshot did not include expected tripId ${TRIP_ID}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Queueing full itinerary generation job..."
GENERATE_JOB_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{jobType:"full_generation",expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generation-jobs" "${ACCESS_TOKEN}" "${GENERATE_JOB_PAYLOAD}"
assert_status "Create full generation job" "202"
GENERATION_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
GENERATION_JOB_STATUS="$(jq -r '.job.status // empty' <<<"${LAST_BODY}")"
if [[ -z "${GENERATION_JOB_ID}" || "${GENERATION_JOB_STATUS}" != "queued" ]]; then
  echo "Full generation job response did not include a queued job." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
poll_generation_job "Full generation" "${TRIP_ID}" "${GENERATION_JOB_ID}" "${ACCESS_TOKEN}"
GENERATION_JOB_STATUS="$(jq -r '.job.status // empty' <<<"${LAST_BODY}")"
if [[ "${GENERATION_JOB_STATUS}" != "completed" ]]; then
  echo "Full generation job did not complete successfully." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ "${SMOKE_EXPECT_OBSERVABILITY:-true}" == "true" ]]; then
  echo "Checking job and queue metrics after full generation..."
  assert_metrics_contains "Trip generation job metrics" "${TRIP_SERVICE_URL}/metrics" "generation_jobs_created_total"
  assert_metrics_contains "Trip RabbitMQ publish metrics" "${TRIP_SERVICE_URL}/metrics" "rabbitmq_messages_published_total"
  assert_metrics_contains "AI generation trace metrics" "${TRIP_SERVICE_URL}/metrics" "ai_generation_traces_started_total"
  if [[ "${SMOKE_EXPECT_WORKER_SERVICE:-true}" == "true" ]]; then
    assert_metrics_contains "Worker job start metrics" "${WORKER_SERVICE_URL}/metrics" "worker_jobs_started_total"
    if [[ "${LAST_BODY}" != *"worker_jobs_completed_total"* && "${LAST_BODY}" != *"worker_jobs_failed_total"* ]]; then
      echo "Worker metrics missing both worker_jobs_completed_total and worker_jobs_failed_total." >&2
      exit 1
    fi
  fi
fi

echo "Fetching completed trip with Authorization header..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip"

STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
DAYS_COUNT="$(jq '.itinerary.days | length' <<<"${LAST_BODY}")"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

if [[ "${STATUS}" != "COMPLETED" ]]; then
  echo "Expected trip status COMPLETED, got ${STATUS}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking stale generation job creation conflict..."
STALE_GENERATION_JOB_PAYLOAD="$(jq -nc '{jobType:"full_generation",expectedItineraryRevision:0}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generation-jobs" "${ACCESS_TOKEN}" "${STALE_GENERATION_JOB_PAYLOAD}"
assert_status "Stale full generation job" "409"
if ! jq -e '.error == "itinerary_conflict"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Stale generation job did not return expected conflict payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ "${DAYS_COUNT}" -le 0 ]]; then
  echo "Expected itinerary.days to contain at least one day." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
COMPLETED_TRIP_BODY="${LAST_BODY}"

echo "Checking trip health and consistency summary..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/health" "${ACCESS_TOKEN}"
assert_2xx "Get trip health"
if ! jq -e --arg id "${TRIP_ID}" '
  .tripId == $id
  and (.score | type == "number")
  and .score >= 0
  and .score <= 100
  and (.level as $level | ["ready", "almost_ready", "needs_attention", "not_ready"] | index($level) != null)
  and (.summary | type == "string")
  and (.generatedAt | type == "string")
  and (.categories | type == "array")
  and (.issues | type == "array")
  and (.topFixes | type == "array")
  and (.computedFrom.itineraryRevision | type == "number")
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Trip health response was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking budget confidence summary..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-confidence" "${ACCESS_TOKEN}"
assert_2xx "Get budget confidence"
if ! jq -e --arg id "${TRIP_ID}" '
  .tripId == $id
  and (.score | type == "number")
  and .score >= 0
  and .score <= 100
  and (.level as $level | ["very_low", "low", "medium", "high", "very_high"] | index($level) != null)
  and (.riskLevel as $risk | ["low", "medium", "high", "critical"] | index($risk) != null)
  and (.coverage.overall | type == "number")
  and (.sourceQuality | type == "array")
  and (.plannedVsActual.categories | type == "array")
  and (.issues | type == "array")
  and (.recommendations | type == "array")
  and (.computedAt | type == "string")
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget confidence response was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking private Trip Copilot safe guidance..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Get trip before Copilot guidance"
COPILOT_REVISION_BEFORE="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
COPILOT_CHAT_PAYLOAD="$(jq -nc '{message:"What should I fix first?",clientContext:{currentTab:"overview"}}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${ACCESS_TOKEN}" "${COPILOT_CHAT_PAYLOAD}"
assert_2xx "Copilot next-action guidance"
if ! jq -e --arg tripHref "/trips/${TRIP_ID}" '
  (.answer | type == "string" and length > 0)
  and (.actions | type == "array")
  and (.sources | type == "array")
  and all(.actions[]?; (.href | startswith($tripHref)))
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Copilot response was not safely shaped." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
COPILOT_UNSAFE_PAYLOAD="$(jq -nc '{message:"Delete this trip"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${ACCESS_TOKEN}" "${COPILOT_UNSAFE_PAYLOAD}"
assert_2xx "Copilot unsafe request refusal"
if ! jq -e '.metadata.intent == "unsafe_mutation_request" and (.answer | test("cannot|can.t|no puedo|ne peux pas|не можу"; "i"))' <<<"${LAST_BODY}" >/dev/null; then
  echo "Copilot did not safely refuse the destructive request." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Get trip after Copilot guidance"
COPILOT_REVISION_AFTER="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${COPILOT_REVISION_BEFORE}" != "${COPILOT_REVISION_AFTER}" ]]; then
  echo "Copilot guidance unexpectedly changed the trip itinerary." >&2
  exit 1
fi

echo "Generating smart packing checklist..."
CHECKLIST_GENERATE_PAYLOAD="$(jq -nc '{mode:"full",outputLanguage:"en",instructions:"Include hiking and rainy weather preparation.",replaceAiItems:true,preserveCheckedItems:true,preserveManualItems:true}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/generate" "${ACCESS_TOKEN}" "${CHECKLIST_GENERATE_PAYLOAD}"
assert_2xx "Generate trip checklist"
CHECKLIST_ID="$(jq -r '.checklist.id // empty' <<<"${LAST_BODY}")"
CHECKLIST_ITEM_COUNT="$(jq '.checklist.items | length' <<<"${LAST_BODY}")"
CHECKLIST_SUMMARY_TOTAL="$(jq -r '.summary.totalItems // 0' <<<"${LAST_BODY}")"
if [[ -z "${CHECKLIST_ID}" || "${CHECKLIST_ITEM_COUNT}" -lt 4 || "${CHECKLIST_SUMMARY_TOTAL}" -ne "${CHECKLIST_ITEM_COUNT}" ]]; then
  echo "Generated checklist response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '([.checklist.items[] | select(.category == "documents")] | length) >= 1 and ([.checklist.items[] | select(.category == "electronics")] | length) >= 1 and ([.checklist.items[] | select(.category == "money")] | length) >= 1' <<<"${LAST_BODY}" >/dev/null; then
  echo "Generated checklist did not include expected core categories." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
CHECKLIST_AI_ITEM_ID="$(jq -r '[.checklist.items[] | select(.source != "manual")][0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${CHECKLIST_AI_ITEM_ID}" ]]; then
  echo "Generated checklist did not include an AI item to check." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking and unchecking a generated checklist item..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${CHECKLIST_AI_ITEM_ID}/check" "${ACCESS_TOKEN}"
assert_2xx "Check generated checklist item"
if ! jq -e '.checked == true and (.checkedByUserId // "") != ""' <<<"${LAST_BODY}" >/dev/null; then
  echo "Checklist item check response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${CHECKLIST_AI_ITEM_ID}/uncheck" "${ACCESS_TOKEN}"
assert_2xx "Uncheck generated checklist item"
if ! jq -e '.checked == false and .checkedAt == null and .checkedByUserId == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Checklist item uncheck response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Creating, editing, and checking a manual checklist item..."
MANUAL_CHECKLIST_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" '{title:"Smoke checklist manual charger",description:"Keep with carry-on.",category:"electronics",itemType:"packing",priority:"high",quantity:1,assignedToUserId:$userId,dueDate:"2026-08-09",reason:"Manual smoke item",metadata:{smoke:true}}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items" "${ACCESS_TOKEN}" "${MANUAL_CHECKLIST_PAYLOAD}"
assert_status "Create manual checklist item" "201"
MANUAL_CHECKLIST_ITEM_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${MANUAL_CHECKLIST_ITEM_ID}" ]] || ! jq -e --arg userId "${OWNER_USER_ID}" '.source == "manual" and .assignedToUserId == $userId and .dueDate == "2026-08-09"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Manual checklist item response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${MANUAL_CHECKLIST_ITEM_ID}" "${ACCESS_TOKEN}" '{"title":"Smoke checklist manual charger updated","priority":"critical","clearDueDate":true}'
assert_2xx "Update manual checklist item"
if ! jq -e '.title == "Smoke checklist manual charger updated" and .priority == "critical" and .dueDate == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Manual checklist item update response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${MANUAL_CHECKLIST_ITEM_ID}/check" "${ACCESS_TOKEN}"
assert_2xx "Check manual checklist item"
if ! jq -e --arg userId "${OWNER_USER_ID}" '.checked == true and .checkedByUserId == $userId' <<<"${LAST_BODY}" >/dev/null; then
  echo "Manual checklist item check response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Reordering checklist items..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist" "${ACCESS_TOKEN}"
assert_2xx "Fetch checklist before reorder"
CHECKLIST_ITEM_IDS="$(jq -c '[.checklist.items[].id]' <<<"${LAST_BODY}")"
EXPECTED_FIRST_AFTER_REORDER="$(jq -r '.[-1]' <<<"${CHECKLIST_ITEM_IDS}")"
CHECKLIST_REORDER_PAYLOAD="$(jq -nc --argjson itemIds "${CHECKLIST_ITEM_IDS}" '{itemIds:($itemIds | reverse)}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/reorder" "${ACCESS_TOKEN}" "${CHECKLIST_REORDER_PAYLOAD}"
assert_2xx "Reorder checklist items"
FIRST_AFTER_REORDER="$(jq -r '.checklist.items[0].id // empty' <<<"${LAST_BODY}")"
if [[ "${FIRST_AFTER_REORDER}" != "${EXPECTED_FIRST_AFTER_REORDER}" ]]; then
  echo "Checklist reorder did not move the expected item first." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Deleting a temporary checklist item..."
TEMP_CHECKLIST_PAYLOAD='{"title":"Smoke checklist temporary item","category":"other","itemType":"reminder","priority":"low"}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items" "${ACCESS_TOKEN}" "${TEMP_CHECKLIST_PAYLOAD}"
assert_status "Create temporary checklist item" "201"
TEMP_CHECKLIST_ITEM_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${TEMP_CHECKLIST_ITEM_ID}" "${ACCESS_TOKEN}"
assert_2xx "Delete temporary checklist item"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist" "${ACCESS_TOKEN}"
assert_2xx "Fetch checklist after delete"
if jq -e --arg id "${TEMP_CHECKLIST_ITEM_ID}" '.checklist.items | any(.id == $id)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Deleted checklist item was still returned." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Regenerating checklist while preserving manual and checked items..."
CHECKLIST_REGEN_PAYLOAD="$(jq -nc '{mode:"add_missing",outputLanguage:"en",instructions:"Preserve smoke manual item.",replaceAiItems:false,preserveCheckedItems:true,preserveManualItems:true}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/generate" "${ACCESS_TOKEN}" "${CHECKLIST_REGEN_PAYLOAD}"
assert_2xx "Regenerate trip checklist add missing"
if ! jq -e --arg id "${MANUAL_CHECKLIST_ITEM_ID}" '.checklist.items | any(.id == $id and .source == "manual" and .checked == true)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Checklist regeneration did not preserve the checked manual item." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
CHECKLIST_AFTER_REGEN_COUNT="$(jq '.checklist.items | length' <<<"${LAST_BODY}")"
if [[ "${CHECKLIST_AFTER_REGEN_COUNT}" -lt "${CHECKLIST_ITEM_COUNT}" ]]; then
  echo "Checklist regeneration unexpectedly reduced the checklist size." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Generating smart pre-trip reminders..."
REMINDER_GENERATE_PAYLOAD="$(jq -nc '{mode:"full",instructions:"Include documents, weather, tickets, and group readiness.",replaceGeneratedPendingReminders:true,preserveManualReminders:true,preserveCompletedReminders:true}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders/generate" "${ACCESS_TOKEN}" "${REMINDER_GENERATE_PAYLOAD}"
assert_2xx "Generate trip reminders"
REMINDER_COUNT="$(jq '.reminders | length' <<<"${LAST_BODY}")"
if [[ "${REMINDER_COUNT}" -lt 4 ]]; then
  echo "Generated reminder response had too few reminders." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '
  (.summary.total == (.reminders | length))
  and (.summary.pending >= 1)
  and (.reminders | any(.category == "documents"))
  and (.reminders | any(.category == "weather"))
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Generated reminders did not include expected summary/category shape." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.reminders | any(.category == "documents" and .triggerDate <= "2026-08-03")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Document reminders were not scheduled at least seven days before departure." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Creating, completing, reopening, and disabling a manual reminder..."
MANUAL_REMINDER_PAYLOAD="$(jq -nc --arg userId "${OWNER_USER_ID}" '{title:"Smoke reminder charge power bank",description:"Due reminder smoke test.",category:"before_departure",priority:"high",triggerDate:"2026-07-20",triggerTime:"09:00",timezone:"UTC",assignedToUserId:$userId,metadata:{smoke:true}}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders" "${ACCESS_TOKEN}" "${MANUAL_REMINDER_PAYLOAD}"
assert_status "Create manual reminder" "201"
MANUAL_REMINDER_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${MANUAL_REMINDER_ID}" ]] || ! jq -e --arg userId "${OWNER_USER_ID}" '.source == "manual" and .assignedToUserId == $userId and .status == "pending"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Manual reminder response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders/${MANUAL_REMINDER_ID}/complete" "${ACCESS_TOKEN}"
assert_2xx "Complete manual reminder"
if ! jq -e --arg userId "${OWNER_USER_ID}" '.status == "completed" and .completedByUserId == $userId' <<<"${LAST_BODY}" >/dev/null; then
  echo "Complete reminder response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders/${MANUAL_REMINDER_ID}/reopen" "${ACCESS_TOKEN}"
assert_2xx "Reopen manual reminder"
if ! jq -e '.status == "pending" and .completedAt == null and .completedByUserId == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Reopen reminder response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders/${MANUAL_REMINDER_ID}/disable" "${ACCESS_TOKEN}"
assert_2xx "Disable manual reminder"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders/${MANUAL_REMINDER_ID}/enable" "${ACCESS_TOKEN}"
assert_2xx "Enable manual reminder"
if ! jq -e '.status == "pending" and .disabledAt == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Enable reminder response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Processing due reminders through the internal Trip Service endpoint..."
PROCESS_REMINDERS_PAYLOAD='{"now":"2026-07-20T09:10:00Z","limit":100}'
request_with_internal_token POST "${TRIP_SERVICE_URL}/internal/reminders/process-due" "${INTERNAL_SERVICE_TOKEN_FOR_SMOKE}" "${PROCESS_REMINDERS_PAYLOAD}"
assert_2xx "Process due reminders"
if ! jq -e '.processed >= 1 and .sent >= 1 and .failed == 0' <<<"${LAST_BODY}" >/dev/null; then
  echo "Due reminder processing did not send the expected reminder." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Owner due reminder notification" "${ACCESS_TOKEN}" "pre_trip_reminder_due"
request_with_internal_token POST "${TRIP_SERVICE_URL}/internal/reminders/process-due" "${INTERNAL_SERVICE_TOKEN_FOR_SMOKE}" "${PROCESS_REMINDERS_PAYLOAD}"
assert_2xx "Process due reminders idempotently"
if ! jq -e '.processed == 0 and .sent == 0' <<<"${LAST_BODY}" >/dev/null; then
  echo "Due reminder processing was not idempotent." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/reminders?status=sent" "${ACCESS_TOKEN}"
assert_2xx "List sent reminders"
if ! jq -e --arg id "${MANUAL_REMINDER_ID}" '.reminders | any(.id == $id and .status == "sent" and .sentAt != null)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Sent manual reminder was not returned as sent." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking private trip template flow..."
PRIVATE_TEMPLATE_PAYLOAD="$(jq -nc '{
  title:"Smoke private template",
  description:"Reusable smoke itinerary",
  visibility:"private",
  destinationHint:"Rome",
  defaultCurrency:"EUR",
  tags:["smoke","city-break"]
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/templates" "${ACCESS_TOKEN}" "${PRIVATE_TEMPLATE_PAYLOAD}"
assert_status "Save private trip template" "201"
PRIVATE_TEMPLATE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${PRIVATE_TEMPLATE_ID}" ]]; then
  echo "Private template response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.templateJson.schemaVersion == 1 and (.templateJson.days[0].dayOffset == 0)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Private template did not include expected versioned dayOffset payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if jq -e '.. | objects | has("placeEnrichment") or has("priceEnrichment") or has("availability")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Template payload retained enrichment or availability metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trip-templates?visibility=private" "${ACCESS_TOKEN}"
assert_2xx "List private trip templates"
if ! jq -e --arg id "${PRIVATE_TEMPLATE_ID}" '.templates | any(.id == $id)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Private template was not visible in template list." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
CREATE_FROM_PRIVATE_TEMPLATE_PAYLOAD="$(jq -nc '{
  title:"Smoke trip from private template",
  destination:"Rome",
  startDate:"2026-09-10",
  budget:{amount:700,currency:"EUR"},
  travelers:2,
  pace:"balanced"
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${PRIVATE_TEMPLATE_ID}/create-trip" "${ACCESS_TOKEN}" "${CREATE_FROM_PRIVATE_TEMPLATE_PAYLOAD}"
assert_status "Create trip from private template" "201"
PRIVATE_TEMPLATE_TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
PRIVATE_TEMPLATE_TRIP_STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
PRIVATE_TEMPLATE_TRIP_START="$(jq -r '.startDate // empty' <<<"${LAST_BODY}")"
if [[ -z "${PRIVATE_TEMPLATE_TRIP_ID}" || "${PRIVATE_TEMPLATE_TRIP_STATUS}" != "COMPLETED" || "${PRIVATE_TEMPLATE_TRIP_START}" != "2026-09-10" ]]; then
  echo "Trip created from private template did not include expected completed trip metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${PRIVATE_TEMPLATE_TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List private template-created trip versions"
if ! jq -e '.items | any(.source == "CREATED_FROM_TEMPLATE")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Template-created trip did not record CREATED_FROM_TEMPLATE version source." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Adapting the private template with AI (mock mode)..."
ADAPTATION_JOB_PAYLOAD="$(jq -nc '{
  title:"Smoke AI adaptation to Vienna",
  destination:"Vienna",
  startDate:"2026-10-01",
  durationDays:3,
  budget:{amount:700,currency:"EUR"},
  travelers:2,
  pace:"balanced",
  interests:["museums","food"],
  avoid:["nightclubs"],
  specialInstructions:"Make it suitable for first-time visitors.",
  fallbackToDeterministic:true
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${PRIVATE_TEMPLATE_ID}/adaptation-jobs" "${ACCESS_TOKEN}" "${ADAPTATION_JOB_PAYLOAD}"
assert_status "Create template adaptation job" "202"
ADAPTATION_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
ADAPTED_TRIP_ID="$(jq -r '.job.tripId // empty' <<<"${LAST_BODY}")"
if [[ -z "${ADAPTATION_JOB_ID}" || -z "${ADAPTED_TRIP_ID}" ]]; then
  echo "Template adaptation job did not return job id and draft trip id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.job.jobType == "template_adaptation" and .job.status == "queued"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Template adaptation job did not start as a queued template_adaptation job." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
poll_generation_job "Template adaptation" "${ADAPTED_TRIP_ID}" "${ADAPTATION_JOB_ID}" "${ACCESS_TOKEN}"
if ! jq -e '.job.status == "completed"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Template adaptation job did not complete." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.job.resultPayload.targetDurationDays == 3' <<<"${LAST_BODY}" >/dev/null; then
  echo "Template adaptation job did not store an adaptation summary result payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${ADAPTED_TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch AI-adapted trip"
if ! jq -e '.destination == "Vienna" and .days == 3' <<<"${LAST_BODY}" >/dev/null; then
  echo "AI-adapted trip did not adapt destination to Vienna with 3 days." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${ADAPTED_TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List AI-adapted trip versions"
if ! jq -e '.items | any(.source == "CREATED_FROM_TEMPLATE_AI")' <<<"${LAST_BODY}" >/dev/null; then
  echo "AI-adapted trip did not record CREATED_FROM_TEMPLATE_AI version source." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${ADAPTED_TRIP_ID}/activity" "${ACCESS_TOKEN}"
assert_2xx "List AI-adapted trip activity"
assert_activity_has "AI-adapted trip activity" "trip_created_from_ai_template_adaptation"

echo "Rejecting an invalid template adaptation duration..."
INVALID_ADAPTATION_PAYLOAD="$(jq -nc '{title:"Bad duration",destination:"Vienna",startDate:"2026-10-01",durationDays:40}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${PRIVATE_TEMPLATE_ID}/adaptation-jobs" "${ACCESS_TOKEN}" "${INVALID_ADAPTATION_PAYLOAD}"
assert_status "Reject invalid adaptation duration" "400"

echo "Blocking another user from adapting a private template..."
OTHER_ADAPTATION_PAYLOAD="$(jq -nc '{title:"Not allowed",destination:"Vienna",startDate:"2026-10-01",durationDays:3}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-templates/${PRIVATE_TEMPLATE_ID}/adaptation-jobs" "${COLLAB_ACCESS_TOKEN}" "${OTHER_ADAPTATION_PAYLOAD}"
assert_status "Block other user from adapting private template" "404"

echo "Checking budget summary reflects generated itinerary costs..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Get budget summary"
BUDGET_ESTIMATED_TOTAL="$(jq -r '.estimatedTotal // empty' <<<"${LAST_BODY}")"
if [[ -z "${BUDGET_ESTIMATED_TOTAL}" ]]; then
  echo "Budget summary did not include estimatedTotal." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.tripBudget == 500' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected budget summary tripBudget=500." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
echo "Budget summary estimatedTotal=${BUDGET_ESTIMATED_TOTAL}."

echo "Updating the trip budget (must not change itineraryRevision)..."
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget" "${ACCESS_TOKEN}" '{"budget":{"amount":300,"currency":"EUR"}}'
assert_2xx "Update trip budget"
if ! jq -e '.budget.amount == 300 and .budget.currency == "EUR"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget update did not echo the new budget." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip after budget update"
REVISION_AFTER_BUDGET="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${REVISION_AFTER_BUDGET}" != "${TRIP_REVISION}" ]]; then
  echo "Budget update unexpectedly changed itineraryRevision (${TRIP_REVISION} -> ${REVISION_AFTER_BUDGET})." >&2
  exit 1
fi

echo "Rejecting an invalid budget currency..."
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget" "${ACCESS_TOKEN}" '{"budget":{"amount":100,"currency":"EU"}}'
assert_status "Invalid budget currency" "400"

echo "Confirming budget summary reflects the lower budget..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Get budget summary after budget update"
if ! jq -e '.tripBudget == 300' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget summary did not reflect updated budget 300." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Queueing a day-level budget optimization job..."
BUDGET_OPTIMIZATION_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{
  scope:"day",
  dayNumber:1,
  targetReductionAmount:20,
  currency:"EUR",
  expectedItineraryRevision:$revision,
  constraints:{
    preserveMustSeeItems:true,
    maxWalkingIncreaseKm:2,
    keepMealCount:true,
    avoidReplacingManualCosts:true
  },
  instruction:"Reduce cost while preserving the day theme."
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-jobs" "${ACCESS_TOKEN}" "${BUDGET_OPTIMIZATION_PAYLOAD}"
assert_status "Create budget optimization job" "202"
BUDGET_OPTIMIZATION_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${BUDGET_OPTIMIZATION_JOB_ID}" ]]; then
  echo "Budget optimization job response did not include a job id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
poll_generation_job "Budget optimization" "${TRIP_ID}" "${BUDGET_OPTIMIZATION_JOB_ID}" "${ACCESS_TOKEN}"
BUDGET_OPTIMIZATION_JOB_STATUS="$(jq -r '.job.status // empty' <<<"${LAST_BODY}")"
if [[ "${BUDGET_OPTIMIZATION_JOB_STATUS}" != "completed" ]]; then
  echo "Budget optimization job did not complete successfully." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Budget optimization ready notification" "${ACCESS_TOKEN}" "budget_optimization_ready"

echo "Fetching pending budget optimization proposal..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals?status=pending&limit=20" "${ACCESS_TOKEN}"
assert_2xx "List budget optimization proposals"
BUDGET_PROPOSAL_ID="$(jq -r '.proposals[0].id // empty' <<<"${LAST_BODY}")"
BUDGET_PROPOSAL_SAVINGS="$(jq -r '.proposals[0].estimatedSavingsAmount // 0' <<<"${LAST_BODY}")"
if [[ -z "${BUDGET_PROPOSAL_ID}" ]]; then
  echo "Budget optimization did not create a pending proposal." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.proposals[0].status == "pending" and .proposals[0].proposal.proposedDay.day == 1 and (.proposals[0].estimatedSavingsAmount > 0)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Budget optimization proposal was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
echo "Budget optimization proposal ${BUDGET_PROPOSAL_ID} estimates savings=${BUDGET_PROPOSAL_SAVINGS}."

echo "Applying budget optimization proposal..."
BUDGET_APPLY_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals/${BUDGET_PROPOSAL_ID}/apply" "${ACCESS_TOKEN}" "${BUDGET_APPLY_PAYLOAD}"
assert_2xx "Apply budget optimization proposal"
BUDGET_APPLIED_REVISION="$(jq -r '.trip.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${BUDGET_APPLIED_REVISION}" != "$((TRIP_REVISION + 1))" ]]; then
  echo "Expected budget optimization apply to increment itineraryRevision to $((TRIP_REVISION + 1)), got ${BUDGET_APPLIED_REVISION}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
TRIP_REVISION="${BUDGET_APPLIED_REVISION}"

echo "Confirming budget optimization proposal cannot be applied twice..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals/${BUDGET_PROPOSAL_ID}/apply" "${ACCESS_TOKEN}" "$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
assert_status "Reapply budget optimization proposal" "400"

echo "Confirming budget summary is still available after budget optimization apply..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Budget summary after budget optimization"
if ! jq -e '.tripBudget == 300 and (.byDay | length > 0)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget summary after optimization was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Creating a second budget proposal to confirm stale apply conflicts..."
SECOND_BUDGET_OPTIMIZATION_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{
  scope:"day",
  dayNumber:1,
  targetReductionAmount:10,
  currency:"EUR",
  expectedItineraryRevision:$revision
}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-jobs" "${ACCESS_TOKEN}" "${SECOND_BUDGET_OPTIMIZATION_PAYLOAD}"
assert_status "Create second budget optimization job" "202"
SECOND_BUDGET_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
poll_generation_job "Second budget optimization" "${TRIP_ID}" "${SECOND_BUDGET_JOB_ID}" "${ACCESS_TOKEN}"
if [[ "$(jq -r '.job.status // empty' <<<"${LAST_BODY}")" != "completed" ]]; then
  echo "Second budget optimization job did not complete successfully." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals?status=pending&limit=20" "${ACCESS_TOKEN}"
assert_2xx "List second pending budget optimization proposal"
SECOND_BUDGET_PROPOSAL_ID="$(jq -r '.proposals[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${SECOND_BUDGET_PROPOSAL_ID}" ]]; then
  echo "Second budget optimization did not create a pending proposal." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
STALE_BUDGET_APPLY_PAYLOAD="$(jq -nc --argjson revision "$((TRIP_REVISION - 1))" '{expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals/${SECOND_BUDGET_PROPOSAL_ID}/apply" "${ACCESS_TOKEN}" "${STALE_BUDGET_APPLY_PAYLOAD}"
assert_status "Stale budget optimization apply" "409"
if ! jq -e '.error == "itinerary_conflict"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Stale budget optimization apply did not return itinerary_conflict." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals/${SECOND_BUDGET_PROPOSAL_ID}/discard" "${ACCESS_TOKEN}"
assert_2xx "Discard second budget optimization proposal"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip after budget optimization"
COMPLETED_TRIP_BODY="${LAST_BODY}"

echo "Editing an item cost through the revision-checked itinerary update..."
UPDATED_ITINERARY="$(jq -c \
  '.itinerary | .days[0].items[0].estimatedCost = {amount:10000,currency:"JPY",category:"activity",confidence:"high",source:"manual",note:"smoke multi-currency"}' \
  <<<"${COMPLETED_TRIP_BODY}")"
ITINERARY_UPDATE_PAYLOAD="$(jq -nc --argjson itinerary "${UPDATED_ITINERARY}" --argjson revision "${TRIP_REVISION}" '{itinerary:$itinerary,expectedItineraryRevision:$revision}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${ACCESS_TOKEN}" "${ITINERARY_UPDATE_PAYLOAD}"
assert_2xx "Update itinerary item cost"
NEW_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${NEW_REVISION}" != "$((TRIP_REVISION + 1))" ]]; then
  echo "Expected itineraryRevision to increment to $((TRIP_REVISION + 1)), got ${NEW_REVISION}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
TRIP_REVISION="${NEW_REVISION}"

echo "Confirming budget summary reflects the edited item cost..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Get budget summary after item cost edit"
if ! jq -e '.estimatedTotal >= 58' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget summary did not reflect the edited item cost." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${BUDGET_CONVERSION_ENABLED:-true}" != "false" ]]; then
  if ! jq -e '
    .currency == "EUR"
    and (.convertedItemCount >= 1)
    and (.unconvertedItemCount == 0)
    and (.originalCurrencyTotals | any(.currency == "JPY" and .amount == 10000))
    and (.exchangeRateInfo.provider == "mock")
  ' >/dev/null <<<"${LAST_BODY}"; then
    echo "Budget summary did not include expected multi-currency conversion metadata." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi
echo "Budget tracking checks passed."

echo "Checking cost splitting traveler roster and allocation summary..."
COST_TRAVELER_ONE_PAYLOAD="$(jq -nc '{name:"Alex Smoke",email:"alex.split@example.com",role:"traveler"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers" "${ACCESS_TOKEN}" "${COST_TRAVELER_ONE_PAYLOAD}"
assert_status "Create first cost-split traveler" "201"
COST_TRAVELER_ONE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${COST_TRAVELER_ONE_ID}" ]]; then
  echo "First cost-split traveler response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

COST_TRAVELER_TWO_PAYLOAD="$(jq -nc '{name:"Blair Smoke",email:"blair.split@example.com",role:"traveler"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers" "${ACCESS_TOKEN}" "${COST_TRAVELER_TWO_PAYLOAD}"
assert_status "Create second cost-split traveler" "201"
COST_TRAVELER_TWO_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${COST_TRAVELER_TWO_ID}" ]]; then
  echo "Second cost-split traveler response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers/${COST_TRAVELER_TWO_ID}" "${ACCESS_TOKEN}" '{"name":"Blair Updated","role":"traveler"}'
assert_2xx "Update cost-split traveler"
if ! jq -e '.name == "Blair Updated" and .status == "active"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Updated cost-split traveler response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers" "${ACCESS_TOKEN}"
assert_2xx "List cost-split travelers"
if ! jq -e --arg one "${COST_TRAVELER_ONE_ID}" --arg two "${COST_TRAVELER_TWO_ID}" '.travelers | length == 2 and any(.id == $one and .status == "active") and any(.id == $two and .name == "Blair Updated")' >/dev/null <<<"${LAST_BODY}"; then
  echo "Traveler list did not include the expected active travelers." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/cost-splitting/summary?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Get initial cost-splitting summary"
if ! jq -e '.summary.travelerCount == 2 and .summary.estimatedTotal > 0 and .summary.allocatedTotal > 0 and .summary.defaultSplitCount >= 1 and (.travelers | length == 2)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Initial cost-splitting summary did not include expected default allocations." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

SELECTED_SPLIT_PAYLOAD="$(
  jq -nc \
    --arg travelerId "${COST_TRAVELER_ONE_ID}" \
    --argjson revision "${TRIP_REVISION}" \
    '{expectedItineraryRevision:$revision,split:{type:"selected_equal",travelerIds:[$travelerId]}}'
)"
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/days/1/items/0/cost-split" "${ACCESS_TOKEN}" "${SELECTED_SPLIT_PAYLOAD}"
assert_2xx "Update itinerary item cost split"
TRIP_REVISION="$(jq -r '.trip.itineraryRevision // -1' <<<"${LAST_BODY}")"
if ! jq -e '.trip.itinerary.days[0].items[0].estimatedCost.split.type == "selected_equal"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Updated itinerary item did not include selected_equal split." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

CUSTOM_ACCOMMODATION_SPLIT_PAYLOAD="$(
  jq -nc \
    --arg one "${COST_TRAVELER_ONE_ID}" \
    --arg two "${COST_TRAVELER_TWO_ID}" \
    '{split:{type:"custom_percentages",percentages:{($one):25,($two):75}}}'
)"
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation/cost-split" "${ACCESS_TOKEN}" "${CUSTOM_ACCOMMODATION_SPLIT_PAYLOAD}"
assert_2xx "Update accommodation cost split"
if ! jq -e '.accommodation.estimatedCost.split.type == "custom_percentages"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Accommodation response did not include custom_percentages split." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/cost-splitting/summary?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Get cost-splitting summary after split updates"
if ! jq -e --arg one "${COST_TRAVELER_ONE_ID}" --arg two "${COST_TRAVELER_TWO_ID}" '
  .summary.travelerCount == 2
  and .summary.allocatedTotal > 0
  and .summary.invalidSplitCount == 0
  and (.travelers | any(.travelerId == $one and .allocatedTotal > 0 and (.items | any(.splitType == "selected_equal"))))
  and (.travelers | any(.travelerId == $two and .allocatedTotal > 0 and (.items | any(.type == "accommodation" and .splitType == "custom_percentages"))))
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Cost-splitting summary did not reflect selected/custom split rules." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers/${COST_TRAVELER_TWO_ID}" "${ACCESS_TOKEN}"
assert_2xx "Remove cost-split traveler"
if ! jq -e '.success == true' >/dev/null <<<"${LAST_BODY}"; then
  echo "Remove traveler response did not confirm success." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/cost-splitting/summary?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Get cost-splitting summary after traveler removal"
if ! jq -e '.summary.travelerCount == 1 and .summary.invalidSplitCount >= 1 and .summary.unassignedTotal > 0 and (.unassignedCosts | length >= 1)' >/dev/null <<<"${LAST_BODY}"; then
  echo "Cost-splitting summary did not surface the removed-traveler stale split as invalid/unassigned." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking trip cost analytics reflects budget tracking data..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/analytics/costs?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Get trip cost analytics"
if ! jq -e '.summary.estimatedTotal > 0 and (.byDay | length >= 1) and (.byCategory | length >= 1) and (.expensiveItems | length >= 1)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Trip cost analytics did not include expected totals and rollups." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ "${CALENDAR_PROVIDER:-mock}" == "mock" ]]; then
  echo "Checking mock Google Calendar connection and sync..."
  request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/calendar/google/status" "${ACCESS_TOKEN}"
  assert_2xx "Initial Google Calendar status"
  if ! jq -e '.connected == false and .provider == "google"' <<<"${LAST_BODY}" >/dev/null; then
    echo "Expected initial Google Calendar status to be disconnected." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  CONNECT_PAYLOAD="$(jq -nc --arg returnUrl "${WEB_APP_URL}/trips/${TRIP_ID}" '{returnUrl:$returnUrl}')"
  request_with_bearer POST "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/calendar/google/connect" "${ACCESS_TOKEN}" "${CONNECT_PAYLOAD}"
  assert_2xx "Start Google Calendar mock connect"
  CALENDAR_AUTH_URL="$(jq -r '.authUrl // empty' <<<"${LAST_BODY}")"
  if [[ -z "${CALENDAR_AUTH_URL}" ]]; then
    echo "Calendar connect did not return authUrl." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  request GET "${CALENDAR_AUTH_URL}"
  assert_status "Mock Google Calendar callback redirect" "302"

  request_with_bearer GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/calendar/google/status" "${ACCESS_TOKEN}"
  assert_2xx "Connected Google Calendar status"
  if ! jq -e '.connected == true and .providerAccountEmail != null' <<<"${LAST_BODY}" >/dev/null; then
    echo "Expected connected Google Calendar status with account email." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  SYNC_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
  request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/calendar-sync/google/sync" "${ACCESS_TOKEN}" "${SYNC_PAYLOAD}"
  assert_2xx "Sync trip to Google Calendar"
  CALENDAR_SYNC_STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
  if [[ "${CALENDAR_SYNC_STATUS}" != "synced" && "${CALENDAR_SYNC_STATUS}" != "no_timed_items" ]]; then
    echo "Unexpected calendar sync status." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/calendar-sync/google/status" "${ACCESS_TOKEN}"
  assert_2xx "Trip Google Calendar sync status"
  if ! jq -e '.connected == true and .provider == "google"' <<<"${LAST_BODY}" >/dev/null; then
    echo "Expected Trip Service calendar sync status to include connected Google account." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi

  request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/calendar-sync/google/sync" "${ACCESS_TOKEN}" "${SYNC_PAYLOAD}"
  assert_2xx "Update trip Google Calendar sync"

  request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/calendar-sync/google" "${ACCESS_TOKEN}"
  assert_2xx "Remove trip Google Calendar sync"
fi

echo "Inviting collaborator as viewer..."
INVITE_PAYLOAD="$(jq -nc --arg email "${COLLAB_EMAIL}" '{email:$email,role:"viewer"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/collaborators" "${ACCESS_TOKEN}" "${INVITE_PAYLOAD}"
assert_2xx "Invite collaborator"

COLLABORATOR_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
INVITED_USER_ID="$(jq -r '.userId // empty' <<<"${LAST_BODY}")"
INVITED_ROLE="$(jq -r '.role // empty' <<<"${LAST_BODY}")"
INVITED_STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
if [[ -z "${COLLABORATOR_ID}" || "${INVITED_USER_ID}" != "${COLLAB_USER_ID}" || "${INVITED_ROLE}" != "viewer" || "${INVITED_STATUS}" != "pending" ]]; then
  echo "Collaborator invite response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${COLLAB_ACCESS_TOKEN}" '{"message":"What should I fix first?"}'
assert_status "Pending collaborator Copilot access" "404"

echo "Checking collaborator pending invitations..."
request_with_bearer GET "${TRIP_SERVICE_URL}/collaboration/invitations" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List collaboration invitations"

PENDING_INVITE_COUNT="$(jq --arg id "${COLLABORATOR_ID}" '[.[] | select(.collaboratorId == $id)] | length' <<<"${LAST_BODY}")"
if [[ "${PENDING_INVITE_COUNT}" -ne 1 ]]; then
  echo "Pending invitations did not include collaborator invite." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming pending collaborator cannot view private trip..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_status "Pending collaborator private trip access" "404"

echo "Confirming pending collaborator cannot list comments..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${COLLAB_ACCESS_TOKEN}"
assert_status "Pending collaborator comment access" "404"

echo "Confirming pending collaborator cannot fetch activity..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity" "${COLLAB_ACCESS_TOKEN}"
assert_status "Pending collaborator activity access" "404"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream" "${COLLAB_ACCESS_TOKEN}"
assert_status "Pending collaborator activity stream access" "404"

echo "Confirming pending collaborator cannot update presence..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${COLLAB_ACCESS_TOKEN}" '{"state":"viewing"}'
assert_status "Pending collaborator presence access" "404"

echo "Confirming notification endpoints reject unauthenticated access..."
request GET "${NOTIFICATION_SERVICE_URL}/notifications"
assert_status "Unauthenticated notifications access" "401"

echo "Confirming the invited collaborator received a collaboration_invited notification..."
assert_notification_has "Collaborator invite notification" "${COLLAB_ACCESS_TOKEN}" "collaboration_invited"

# Email notifications (v1): with EMAIL_PROVIDER=mock (the default), this invite —
# and the comment below — also trigger a mock email inside Notification Service.
# The mock provider sends nothing externally; it logs a masked recipient/subject
# line ("email send (mock)"), so there is no externally observable signal to
# assert here without adding a debug endpoint (intentionally avoided). To verify:
#   docker compose -f infra/docker-compose.yml logs notification-service | grep 'email send'
# With real SMTP configured (EMAIL_PROVIDER=smtp + SMTP_*), verify delivery in the
# recipient inbox. By default itinerary_updated is NOT allowlisted, so itinerary
# edits create in-app notifications but send no email.

echo "Confirming notifications are private to their owner..."
COLLAB_FOREIGN_NOTIFICATIONS="$(jq --arg uid "${COLLAB_USER_ID}" '[.items[] | select(.userId != $uid)] | length' <<<"${LAST_BODY}")"
if [[ "${COLLAB_FOREIGN_NOTIFICATIONS}" -ne 0 ]]; then
  echo "Collaborator notification list leaked another user's notifications." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Disabling collaborator email collaboration notifications..."
DISABLE_COLLAB_EMAIL_PREF='{"items":[{"channel":"email","category":"collaboration","enabled":false}]}'
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${COLLAB_ACCESS_TOKEN}" "${DISABLE_COLLAB_EMAIL_PREF}"
assert_2xx "Disable collaborator email collaboration notifications"
COLLAB_EMAIL_COLLAB_ENABLED="$(jq -r '.items[] | select(.channel == "email" and .category == "collaboration") | .enabled' <<<"${LAST_BODY}")"
if [[ "${COLLAB_EMAIL_COLLAB_ENABLED}" != "false" ]]; then
  echo "Collaborator email collaboration preference was not disabled." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ -n "${SMOKE_INTERNAL_SERVICE_TOKEN:-}" ]]; then
  echo "Checking email skippedByPreference through the internal batch endpoint..."
  DIRECT_ACTOR_USER_ID="${OWNER_USER_ID:-${AUTH_ME_ID}}"
  DIRECT_BATCH_PAYLOAD="$(jq -nc \
    --arg userId "${COLLAB_USER_ID}" \
    --arg actorUserId "${DIRECT_ACTOR_USER_ID}" \
    '{notifications:[{userId:$userId,actorUserId:$actorUserId,type:"collaboration_invited",title:"Smoke preference check",message:"Smoke preference check.",metadata:{destination:"Rome"}}]}')"
  request_with_internal_token POST "${NOTIFICATION_SERVICE_URL}/internal/notifications/batch" "${SMOKE_INTERNAL_SERVICE_TOKEN}" "${DIRECT_BATCH_PAYLOAD}"
  assert_2xx "Internal batch email preference check"
  DIRECT_CREATED="$(jq -r '.created // 0' <<<"${LAST_BODY}")"
  DIRECT_EMAIL_SKIPPED_BY_PREF="$(jq -r '.email.skippedByPreference // 0' <<<"${LAST_BODY}")"
  if [[ "${DIRECT_CREATED}" -lt 1 || "${DIRECT_EMAIL_SKIPPED_BY_PREF}" -lt 1 ]]; then
    echo "Internal batch did not report expected created/skippedByPreference stats." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
else
  echo "SMOKE_INTERNAL_SERVICE_TOKEN is not set; skipping direct email skippedByPreference check."
fi

echo "Confirming the trip_created activity event was recorded..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=100" "${ACCESS_TOKEN}"
assert_2xx "Owner fetch early activity"
assert_activity_has "Owner early activity" "trip_created"
assert_activity_has "Owner early activity" "itinerary_generated"

echo "Accepting collaborator invitation..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/collaborators/${COLLABORATOR_ID}/accept" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Accept collaboration invitation"

echo "Confirming the owner received a collaboration_accepted notification..."
assert_notification_has "Owner accepted notification" "${ACCESS_TOKEN}" "collaboration_accepted"
if [[ -n "${OWNER_USER_ID}" ]]; then
  OWNER_FOREIGN_NOTIFICATIONS="$(jq --arg uid "${OWNER_USER_ID}" '[.items[] | select(.userId != $uid)] | length' <<<"${LAST_BODY}")"
  if [[ "${OWNER_FOREIGN_NOTIFICATIONS}" -ne 0 ]]; then
    echo "Owner notification list leaked another user's notifications." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi

echo "Checking Shared with me list..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/shared-with-me" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "List shared trips"

SHARED_TRIP_COUNT="$(jq --arg id "${TRIP_ID}" '[.[] | select(.id == $id and .role == "viewer")] | length' <<<"${LAST_BODY}")"
if [[ "${SHARED_TRIP_COUNT}" -ne 1 ]]; then
  echo "Shared-with-me did not include accepted viewer trip." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking accepted viewer can update viewing presence..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${COLLAB_ACCESS_TOKEN}" '{"state":"viewing"}'
assert_2xx "Accepted viewer presence state"

echo "Checking accepted viewer can read the budget summary but cannot update the budget..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get budget summary"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-confidence" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get budget confidence"
VIEWER_COPILOT_PAYLOAD="$(jq -nc '{message:"What should I fix first?",clientContext:{currentTab:"overview"}}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_COPILOT_PAYLOAD}"
assert_2xx "Viewer get Copilot guidance"
if ! jq -e '(.permissionNotes | type == "array") and ([.actions[]?.type] | index("find_transport") | not) and ([.actions[]?.type] | index("add_expense") | not)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Viewer Copilot response exposed an edit action." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget" "${COLLAB_ACCESS_TOKEN}" '{"budget":{"amount":1000,"currency":"EUR"}}'
assert_status "Viewer update budget" "403"

echo "Checking group readiness summary and owner/editor nudge flow..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness?includeDetails=true" "${ACCESS_TOKEN}"
assert_2xx "Owner get group readiness"
if ! jq -e --arg id "${TRIP_ID}" --arg owner "${OWNER_USER_ID}" --arg collab "${COLLAB_USER_ID}" '
  .tripId == $id
  and (.score | type == "number")
  and .score >= 0
  and .score <= 100
  and (.level as $level | ["ready", "almost_ready", "needs_attention", "not_ready"] | index($level) != null)
  and (.summary | type == "string")
  and (.generatedAt | type == "string")
  and (.members | length) >= 2
  and (.members | any(.userId == $owner))
  and (.members | any(.userId == $collab))
  and (.categorySummary | any(.category == "availability" and .openIssueCount >= 1))
  and (.topActions | type == "array")
  and ([.members[].items[]? | select(.category == "availability" and .status == "missing")] | length) >= 1
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Group readiness response was not shaped as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness?includeDetails=false" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get group readiness"

READINESS_NUDGE_PAYLOAD="$(
  jq -nc --arg target "${COLLAB_USER_ID}" '{
    targetUserIds:[$target],
    message:"Please add availability for the smoke trip.",
    dedupeWindowHours:24
  }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness/nudge-missing-availability" "${ACCESS_TOKEN}" "${READINESS_NUDGE_PAYLOAD}"
assert_2xx "Owner nudge missing availability"
if ! jq -e --arg target "${COLLAB_USER_ID}" '
  .sentCount == 1
  and .skippedCount == 0
  and .dedupedCount == 0
  and (.targetUserIds | index($target) != null)
  and (.categories | index("availability") != null)
  and .dedupeWindowHours == 24
' <<<"${LAST_BODY}" >/dev/null; then
  echo "Readiness availability nudge response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
assert_notification_has "Collaborator availability nudge notification" "${COLLAB_ACCESS_TOKEN}" "availability_nudge"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness/nudge-missing-availability" "${ACCESS_TOKEN}" "${READINESS_NUDGE_PAYLOAD}"
assert_2xx "Owner nudge missing availability deduped"
if ! jq -e '.sentCount == 0 and .dedupedCount >= 1 and (.categories | index("availability") != null)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Readiness nudge dedupe response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=20" "${ACCESS_TOKEN}"
assert_2xx "Group readiness nudge activity"
assert_activity_has "Group readiness nudge activity" "group_readiness_nudge_sent"

VIEWER_READINESS_NUDGE_PAYLOAD="$(jq -nc --arg target "${OWNER_USER_ID}" '{targetUserIds:[$target],categories:["availability"]}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness/nudge" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_READINESS_NUDGE_PAYLOAD}"
assert_status "Viewer group readiness nudge" "403"

echo "Checking accepted viewer can read but cannot mutate cost splitting..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer list cost-split travelers"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/cost-splitting/summary?currency=EUR" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get cost-splitting summary"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/travelers" "${COLLAB_ACCESS_TOKEN}" "${COST_TRAVELER_ONE_PAYLOAD}"
assert_status "Viewer create cost-split traveler" "403"
VIEWER_SPLIT_PAYLOAD="$(
  jq -nc \
    --arg travelerId "${COST_TRAVELER_ONE_ID}" \
    --argjson revision "${TRIP_REVISION}" \
    '{expectedItineraryRevision:$revision,split:{type:"selected_equal",travelerIds:[$travelerId]}}'
)"
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/days/1/items/0/cost-split" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_SPLIT_PAYLOAD}"
assert_status "Viewer update itinerary cost split" "403"

echo "Checking shared expenses and settlement suggestions..."
EXPENSE_PAYLOAD="$(
  jq -nc \
    --arg owner "${OWNER_USER_ID}" \
    --arg collab "${COLLAB_USER_ID}" \
    '{
      title:"Smoke dinner",
      amount:{amount:42,currency:"EUR"},
      category:"food",
      expenseDate:"2026-08-10",
      paidByUserId:$owner,
      splitType:"selected_equal",
      participantUserIds:[$owner,$collab],
      notes:"Smoke actual expense"
    }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses" "${ACCESS_TOKEN}" "${EXPENSE_PAYLOAD}"
assert_2xx "Create trip expense"
EXPENSE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${EXPENSE_ID}" ]] || ! jq -e '.amount.amount == 42 and .amount.currency == "EUR" and (.participants | length) == 2' <<<"${LAST_BODY}" >/dev/null; then
  echo "Create expense response did not include expected amount and participants." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer list expenses"
if ! jq -e --arg id "${EXPENSE_ID}" '.items | any(.id == $id)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Viewer expense list did not include the created expense." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking receipt upload, mock OCR review, expense creation, attach, and delete..."
RECEIPT_FILE="$(mktemp)"
printf '\x89PNG\r\n\x1a\n' > "${RECEIPT_FILE}"
request_receipt_upload "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/upload" "${ACCESS_TOKEN}" "${RECEIPT_FILE}" "train-ticket.png"
assert_status "Upload receipt with OCR" "201"
RECEIPT_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${RECEIPT_ID}" ]] || ! jq -e '.status == "extracted" and .ocrResult.category == "transport" and .ocrResult.amount.amount == 72' <<<"${LAST_BODY}" >/dev/null; then
  echo "Receipt upload did not return expected mock OCR data." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

RECEIPT_EXPENSE_PAYLOAD="$(
  jq -nc \
    --arg owner "${OWNER_USER_ID}" \
    --arg collab "${COLLAB_USER_ID}" \
    '{
      title:"Train tickets",
      amount:{amount:72,currency:"EUR"},
      category:"transport",
      expenseDate:"2026-08-10",
      paidByUserId:$owner,
      splitType:"selected_equal",
      participantUserIds:[$owner,$collab],
      notes:"Reviewed receipt OCR"
    }'
)"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/${RECEIPT_ID}/create-expense" "${ACCESS_TOKEN}" "${RECEIPT_EXPENSE_PAYLOAD}"
assert_status "Create expense from receipt" "201"
RECEIPT_EXPENSE_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${RECEIPT_EXPENSE_ID}" ]] || ! jq -e --arg receipt_id "${RECEIPT_ID}" '.hasReceipt == true and (.receipts | any(.id == $receipt_id))' <<<"${LAST_BODY}" >/dev/null; then
  echo "Create expense from receipt did not link the receipt." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_receipt_upload "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/upload" "${ACCESS_TOKEN}" "${RECEIPT_FILE}" "parking-receipt.png" "${EXPENSE_ID}"
assert_status "Upload receipt attached to existing expense" "201"
ATTACHED_RECEIPT_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${ATTACHED_RECEIPT_ID}" ]] || ! jq -e --arg expense_id "${EXPENSE_ID}" '.expenseId == $expense_id and .status == "extracted"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Attached receipt response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/${ATTACHED_RECEIPT_ID}/file" "${ACCESS_TOKEN}"
assert_2xx "Download attached receipt file"
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/${ATTACHED_RECEIPT_ID}" "${ACCESS_TOKEN}"
assert_2xx "Delete attached receipt"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts/${ATTACHED_RECEIPT_ID}/file" "${ACCESS_TOKEN}"
assert_status "Deleted receipt file denied" "404"
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/receipts"
assert_status "Unauthenticated receipt list denied" "401"
rm -f "${RECEIPT_FILE}"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/summary?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Expense summary"
if ! jq -e '.actualTotal.amount == 114 and .settlementSummary.pendingCount >= 1 and (.balances | length) >= 2' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expense summary did not include expected totals, balances, and pending settlement." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/settlements?currency=EUR" "${ACCESS_TOKEN}"
assert_2xx "Settlement suggestions"
SETTLEMENT_ID="$(jq -r '.suggestions[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${SETTLEMENT_ID}" ]]; then
  echo "Settlement suggestions did not include a calculated settlement." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
SETTLEMENT_ID_ENCODED="$(jq -rn --arg id "${SETTLEMENT_ID}" '$id|@uri')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/settlements/${SETTLEMENT_ID_ENCODED}/mark-paid" "${ACCESS_TOKEN}" '{"notes":"Smoke settlement paid"}'
assert_2xx "Mark settlement paid"
if ! jq -e '(.paidSettlements | length) >= 1' <<<"${LAST_BODY}" >/dev/null; then
  echo "Mark settlement paid did not return a paid settlement." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses" "${COLLAB_ACCESS_TOKEN}" "${EXPENSE_PAYLOAD}"
assert_status "Viewer create expense paid by owner" "403"

echo "Checking viewer can read but cannot mutate accommodation..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get accommodation"
if ! jq -e '.accommodation.name == "Hotel Roma"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Viewer accommodation response did not include expected stay." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${COLLAB_ACCESS_TOKEN}" "${ACCOMMODATION_PAYLOAD}"
assert_status "Viewer update accommodation" "403"
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${COLLAB_ACCESS_TOKEN}"
assert_status "Viewer delete accommodation" "403"

echo "Checking accepted viewer checklist permissions..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get checklist"
VIEWER_CHECKLIST_CAN_GENERATE="$(jq -r '.canGenerate // true' <<<"${LAST_BODY}")"
VIEWER_CHECKLIST_UNASSIGNED_ID="$(jq -r '[.checklist.items[] | select(.assignedToUserId == null)][0].id // empty' <<<"${LAST_BODY}")"
if [[ "${VIEWER_CHECKLIST_CAN_GENERATE}" != "false" || -z "${VIEWER_CHECKLIST_UNASSIGNED_ID}" ]]; then
  echo "Viewer checklist read response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/health" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer get trip health"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/generate" "${COLLAB_ACCESS_TOKEN}" '{"mode":"add_missing"}'
assert_status "Viewer generate checklist" "403"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items" "${COLLAB_ACCESS_TOKEN}" "${TEMP_CHECKLIST_PAYLOAD}"
assert_status "Viewer create checklist item" "403"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${VIEWER_CHECKLIST_UNASSIGNED_ID}/check" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer check unassigned checklist item"
if ! jq -e --arg userId "${COLLAB_USER_ID}" '.checked == true and .checkedByUserId == $userId' <<<"${LAST_BODY}" >/dev/null; then
  echo "Viewer checklist check response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking viewer can view but cannot edit..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer collaborator fetch trip"
VIEWER_ACCESS_ROLE="$(jq -r '.access.role // empty' <<<"${LAST_BODY}")"
VIEWER_CAN_EDIT="$(jq -r '.access.canEdit // true' <<<"${LAST_BODY}")"
if [[ "${VIEWER_ACCESS_ROLE}" != "viewer" || "${VIEWER_CAN_EDIT}" != "false" ]]; then
  echo "Viewer trip access metadata was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking group availability and date coordination..."
OWNER_AVAILABILITY_PAYLOAD='{
  "availableRanges":[{"startDate":"2026-09-10","endDate":"2026-09-15"}],
  "preferredRanges":[{"startDate":"2026-09-12","endDate":"2026-09-13"}],
  "minTripDays":2,
  "maxTripDays":5,
  "timezone":"Europe/Bratislava",
  "notes":"Weekend is best."
}'
COLLAB_AVAILABILITY_PAYLOAD='{
  "availableRanges":[{"startDate":"2026-09-12","endDate":"2026-09-16"}],
  "unavailableRanges":[{"startDate":"2026-09-14","endDate":"2026-09-14"}],
  "preferredRanges":[{"startDate":"2026-09-12","endDate":"2026-09-13"}],
  "minTripDays":2,
  "maxTripDays":4,
  "timezone":"Europe/Bratislava",
  "notes":"Avoid Sep 14."
}'
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/me" "${ACCESS_TOKEN}" "${OWNER_AVAILABILITY_PAYLOAD}"
assert_2xx "Owner submit availability"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/me" "${COLLAB_ACCESS_TOKEN}" "${COLLAB_AVAILABILITY_PAYLOAD}"
assert_2xx "Viewer submit own availability"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability" "${ACCESS_TOKEN}"
assert_2xx "Get trip availability"
if ! jq -e '.summary.submittedCount >= 2 and (.responses | length) >= 2' <<<"${LAST_BODY}" >/dev/null; then
  echo "Trip availability summary did not include owner and collaborator responses." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${CALENDAR_PROVIDER:-mock}" == "mock" ]]; then
  echo "Checking Google Calendar free/busy availability import..."
  CALENDAR_IMPORT_PREVIEW_PAYLOAD='{
    "startDate":"2026-09-10",
    "endDate":"2026-09-30",
    "timezone":"Europe/Bratislava",
    "calendarProvider":"google",
    "calendarIds":["primary"],
    "conversion":{
      "fullyBusyThresholdHours":6,
      "markFullyBusyDaysUnavailable":true,
      "markPartiallyBusyDaysUnavailable":false,
      "includeWeekendsAsPreferredIfFree":false
    }
  }'
  request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/import-calendar/preview" "${ACCESS_TOKEN}" "${CALENDAR_IMPORT_PREVIEW_PAYLOAD}"
  assert_2xx "Preview calendar availability import"
  if ! jq -e '.preview.busyBlocksSummary.busyBlockCount >= 2 and .preview.busyBlocksSummary.fullyBusyDays >= 1 and .preview.busyBlocksSummary.partiallyBusyDays >= 1 and (.preview.suggestedUnavailableRanges | length) >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Calendar import preview did not include expected sanitized summaries." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  if jq -e '.. | objects | has("title") or has("description") or has("attendees") or has("location") or has("eventId")' <<<"${LAST_BODY}" >/dev/null; then
    echo "Calendar import preview exposed event details." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  CALENDAR_IMPORT_APPLY_PAYLOAD='{
    "startDate":"2026-09-10",
    "endDate":"2026-09-30",
    "timezone":"Europe/Bratislava",
    "calendarProvider":"google",
    "calendarIds":["primary"],
    "mode":"merge",
    "conversion":{
      "fullyBusyThresholdHours":6,
      "markFullyBusyDaysUnavailable":true,
      "markPartiallyBusyDaysUnavailable":false,
      "includeWeekendsAsPreferredIfFree":false
    },
    "availabilitySettings":{
      "minTripDays":2,
      "maxTripDays":5,
      "timezone":"Europe/Bratislava",
      "notes":"Imported from Google Calendar."
    }
  }'
  request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/import-calendar/apply" "${ACCESS_TOKEN}" "${CALENDAR_IMPORT_APPLY_PAYLOAD}"
  assert_2xx "Apply calendar availability import"
  if ! jq -e '(.availability.unavailableRanges | length) >= 1 and (.dateOptions.options | type == "array")' <<<"${LAST_BODY}" >/dev/null; then
    echo "Calendar import apply did not update availability/date options." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  if jq -e '.. | strings | test("Google Calendar conflict|calendar event|event title"; "i")' <<<"${LAST_BODY}" >/dev/null; then
    echo "Calendar import apply exposed calendar-specific conflict details." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
  request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability" "${ACCESS_TOKEN}"
  assert_2xx "Get availability after calendar import"
  if ! jq -e --arg userId "${AUTH_ME_ID}" '.responses[] | select(.userId == $userId) | (.unavailableRanges | length) >= 1' <<<"${LAST_BODY}" >/dev/null; then
    echo "Imported unavailable ranges were not persisted as normal availability." >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/date-options?minDays=2&maxDays=4&preferWeekends=true&limit=5" "${ACCESS_TOKEN}"
assert_2xx "Get trip date options"
DATE_OPTION_ID="$(jq -r '.options[0].id // empty' <<<"${LAST_BODY}")"
DATE_OPTION_SECOND_ID="$(jq -r '.options[1].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DATE_OPTION_ID}" ]]; then
  echo "Date options response did not include any options." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if ! jq -e '.summary.responseCount >= 2 and .options[0].score >= 0 and (.options[0].pros | type == "array") and (.options[0].cons | type == "array")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Date option response did not include expected scoring fields." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ -n "${DATE_OPTION_SECOND_ID}" ]]; then
  DATE_POLL_PAYLOAD="$(jq -nc --arg first "${DATE_OPTION_ID}" --arg second "${DATE_OPTION_SECOND_ID}" '{title:"Which dates work best?",optionIds:[$first,$second]}')"
else
  DATE_POLL_PAYLOAD="$(jq -nc --arg first "${DATE_OPTION_ID}" '{title:"Which dates work best?",optionIds:[$first]}')"
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/date-options/create-poll" "${ACCESS_TOKEN}" "${DATE_POLL_PAYLOAD}"
assert_status "Create date options poll" "201"
DATE_POLL_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
DATE_POLL_OPTION_ID="$(jq -r '.options[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DATE_POLL_ID}" || -z "${DATE_POLL_OPTION_ID}" ]]; then
  echo "Date poll response did not include poll and option ids." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
DATE_POLL_VOTE_PAYLOAD="$(jq -nc --arg optionId "${DATE_POLL_OPTION_ID}" '{optionIds:[$optionId]}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls/${DATE_POLL_ID}/vote" "${COLLAB_ACCESS_TOKEN}" "${DATE_POLL_VOTE_PAYLOAD}"
assert_2xx "Viewer vote date poll"
DATE_APPLY_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision,regenerateItinerary:false}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/date-options/${DATE_OPTION_ID}/apply" "${COLLAB_ACCESS_TOKEN}" "${DATE_APPLY_PAYLOAD}"
assert_status "Viewer apply date option" "403"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/date-options/${DATE_OPTION_ID}/apply" "${ACCESS_TOKEN}" "${DATE_APPLY_PAYLOAD}"
assert_2xx "Owner apply date option"
if ! jq -e '.trip.startDate != null and .trip.days >= 2 and .appliedOption.id != null and (.warnings | type == "array")' <<<"${LAST_BODY}" >/dev/null; then
  echo "Apply date option response did not include updated trip and applied option." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
DATE_PREVIEW_PAYLOAD="$(jq -nc --arg tripId "${TRIP_ID}" '{source:"trip_generation",tripId:$tripId,includeTripState:true,request:{}}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/planning-constraints/preview" "${ACCESS_TOKEN}" "${DATE_PREVIEW_PAYLOAD}"
assert_2xx "Preview planning constraints after date apply"
if ! jq -e '.constraints.groupAvailability.selectedDateOption.startDate != null and .constraints.dates.flexibility == "fixed"' <<<"${LAST_BODY}" >/dev/null; then
  echo "Planning constraints did not include selected fixed group dates." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking collaborative trip decisions and reactions..."
VIEWER_POLL_CREATE_PAYLOAD='{"title":"Viewer should not create","pollType":"single_choice","options":[{"optionKey":"yes","label":"Yes"},{"optionKey":"no","label":"No"}]}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_POLL_CREATE_PAYLOAD}"
assert_status "Viewer create poll" "403"

DECISION_POLL_PAYLOAD='{"title":"Which destination?","description":"Smoke decision poll.","pollType":"single_choice","options":[{"optionKey":"rome","label":"Rome"},{"optionKey":"florence","label":"Florence"},{"optionKey":"naples","label":"Naples"}]}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls" "${ACCESS_TOKEN}" "${DECISION_POLL_PAYLOAD}"
assert_status "Owner create decision poll" "201"
DECISION_POLL_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
DECISION_OPTION_OWNER="$(jq -r '.options[0].id // empty' <<<"${LAST_BODY}")"
DECISION_OPTION_COLLAB="$(jq -r '.options[1].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DECISION_POLL_ID}" || -z "${DECISION_OPTION_OWNER}" || -z "${DECISION_OPTION_COLLAB}" ]]; then
  echo "Decision poll response did not include expected ids." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

COLLAB_POLL_VOTE_PAYLOAD="$(jq -nc --arg optionId "${DECISION_OPTION_COLLAB}" '{optionIds:[$optionId]}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls/${DECISION_POLL_ID}/vote" "${COLLAB_ACCESS_TOKEN}" "${COLLAB_POLL_VOTE_PAYLOAD}"
assert_2xx "Viewer collaborator vote poll"
OWNER_POLL_VOTE_PAYLOAD="$(jq -nc --arg optionId "${DECISION_OPTION_OWNER}" '{optionIds:[$optionId]}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls/${DECISION_POLL_ID}/vote" "${ACCESS_TOKEN}" "${OWNER_POLL_VOTE_PAYLOAD}"
assert_2xx "Owner vote poll"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls" "${ACCESS_TOKEN}"
assert_2xx "List decision polls"
if ! jq -e --arg id "${DECISION_POLL_ID}" '
  .items
  | any(.id == $id and .results.totalVoters == 2 and .results.totalVotes == 2 and (.results.winningOptionIds | length) == 2)
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Decision poll results did not include expected tied owner/collaborator votes." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/reactions" "${COLLAB_ACCESS_TOKEN}" '{"dayNumber":1,"itemIndex":0,"reaction":"must_have"}'
assert_2xx "Viewer collaborator set itinerary reaction"
if ! jq -e '.counts.must_have >= 1 and .currentUserReaction == "must_have"' >/dev/null <<<"${LAST_BODY}"; then
  echo "Itinerary reaction summary did not include the collaborator must-have reaction." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-preferences" "${ACCESS_TOKEN}"
assert_2xx "Group preferences"
if ! jq -e '
  .summary.pollCount >= 1
  and .summary.reactionCount >= 1
  and .summary.mustHaveItemCount >= 1
  and (.topPollChoices | length) >= 1
  and (.aiConstraintSummary | length) > 0
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Group preferences did not include expected poll and reaction signals." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

DECISION_CONSTRAINTS_PREVIEW_PAYLOAD="$(jq -nc --arg tripId "${TRIP_ID}" '{source:"trip_generation",tripId:$tripId,includePreviousTripSignals:false}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/planning-constraints/preview" "${ACCESS_TOKEN}" "${DECISION_CONSTRAINTS_PREVIEW_PAYLOAD}"
assert_2xx "Planning constraints include group preferences"
if ! jq -e '
  .constraints.groupPreferences != null
  and (.constraints.groupPreferences.mustHaveItems | length) >= 1
  and (.constraints.groupPreferences.summary | length) > 0
' >/dev/null <<<"${LAST_BODY}"; then
  echo "Planning constraints preview did not include group preferences." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls/${DECISION_POLL_ID}/close" "${ACCESS_TOKEN}"
assert_2xx "Close decision poll"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls/${DECISION_POLL_ID}/vote" "${COLLAB_ACCESS_TOKEN}" "${COLLAB_POLL_VOTE_PAYLOAD}"
assert_status "Vote closed decision poll" "409"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=100" "${ACCESS_TOKEN}"
assert_2xx "Decision activity"
assert_activity_has "Decision activity" "trip_poll_created"
assert_activity_has "Decision activity" "trip_poll_closed"

VIEWER_EDIT_PAYLOAD='{"itinerary":{"days":[{"day":1,"title":"Viewer blocked day","items":[{"time":"10:00","type":"activity","name":"Should fail"}]}]}}'
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_EDIT_PAYLOAD}"
assert_status "Viewer itinerary edit" "403"
VIEWER_GENERATION_JOB_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{jobType:"day_regeneration",dayNumber:1,expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generation-jobs" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_GENERATION_JOB_PAYLOAD}"
assert_status "Viewer generation job create" "403"
VIEWER_CALENDAR_SYNC_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/calendar-sync/google/sync" "${COLLAB_ACCESS_TOKEN}" "${VIEWER_CALENDAR_SYNC_PAYLOAD}"
assert_status "Viewer calendar sync" "403"

echo "Checking viewer can read but cannot acquire edit lock..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer read edit lock"
VIEWER_LOCKED="$(jq -r '.locked // true' <<<"${LAST_BODY}")"
if [[ "${VIEWER_LOCKED}" != "false" ]]; then
  echo "Expected no active edit lock before owner acquires one." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_status "Viewer acquire edit lock" "403"

echo "Creating an owner comment on the first itinerary item..."
OWNER_COMMENT_PAYLOAD='{"dayNumber":1,"itemIndex":0,"body":"Owner: can we start this earlier?"}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${ACCESS_TOKEN}" "${OWNER_COMMENT_PAYLOAD}"
assert_2xx "Owner create comment"
OWNER_COMMENT_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
OWNER_COMMENT_IS_AUTHOR="$(jq -r '.isAuthor // false' <<<"${LAST_BODY}")"
if [[ -z "${OWNER_COMMENT_ID}" || "${OWNER_COMMENT_IS_AUTHOR}" != "true" ]]; then
  echo "Owner comment response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming the viewer collaborator can read the owner comment..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments?dayNumber=1&itemIndex=0" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Viewer list item comments"
VIEWER_SEES_OWNER_COMMENT="$(jq --arg id "${OWNER_COMMENT_ID}" '[.items[] | select(.id == $id)] | length' <<<"${LAST_BODY}")"
if [[ "${VIEWER_SEES_OWNER_COMMENT}" != "1" ]]; then
  echo "Viewer did not see the owner comment." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming an accepted viewer collaborator can fetch activity..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Accepted collaborator fetch activity"
echo "Confirming an accepted viewer collaborator can open the activity stream..."
assert_activity_stream_opens "Accepted collaborator activity stream" "${COLLAB_ACCESS_TOKEN}"

echo "Disabling collaborator in-app comment notifications..."
DISABLE_COLLAB_COMMENT_PREF='{"items":[{"channel":"in_app","category":"comments","enabled":false}]}'
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${COLLAB_ACCESS_TOKEN}" "${DISABLE_COLLAB_COMMENT_PREF}"
assert_2xx "Disable collaborator in-app comment notifications"
COLLAB_COMMENTS_IN_APP_ENABLED="$(jq -r '.items[] | select(.channel == "in_app" and .category == "comments") | .enabled' <<<"${LAST_BODY}")"
if [[ "${COLLAB_COMMENTS_IN_APP_ENABLED}" != "false" ]]; then
  echo "Collaborator in-app comment preference was not disabled." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming disabled in-app comments suppress future collaborator notifications..."
OWNER_COMMENT_PREF_PAYLOAD='{"dayNumber":1,"itemIndex":0,"body":"Owner: preference smoke check."}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${ACCESS_TOKEN}" "${OWNER_COMMENT_PREF_PAYLOAD}"
assert_2xx "Owner create preference-check comment"
assert_notification_absent "Collaborator disabled comment preference" "${COLLAB_ACCESS_TOKEN}" "comment_created"

echo "Re-enabling collaborator in-app comment notifications..."
ENABLE_COLLAB_COMMENT_PREF='{"items":[{"channel":"in_app","category":"comments","enabled":true}]}'
request_with_bearer PUT "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${COLLAB_ACCESS_TOKEN}" "${ENABLE_COLLAB_COMMENT_PREF}"
assert_2xx "Re-enable collaborator in-app comment notifications"

echo "Confirming re-enabled in-app comments create future collaborator notifications..."
OWNER_COMMENT_PREF_PAYLOAD_2='{"dayNumber":1,"itemIndex":0,"body":"Owner: notification should return."}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${ACCESS_TOKEN}" "${OWNER_COMMENT_PREF_PAYLOAD_2}"
assert_2xx "Owner create re-enabled notification comment"
assert_notification_has "Collaborator re-enabled comment notification" "${COLLAB_ACCESS_TOKEN}" "comment_created"

echo "Adding a viewer collaborator comment..."
COLLAB_COMMENT_PAYLOAD='{"dayNumber":1,"itemIndex":0,"body":"Viewer: sounds good to me."}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${COLLAB_ACCESS_TOKEN}" "${COLLAB_COMMENT_PAYLOAD}"
assert_2xx "Viewer create comment"
COLLAB_COMMENT_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${COLLAB_COMMENT_ID}" ]]; then
  echo "Viewer comment response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming the owner received a comment_created notification..."
assert_notification_has "Owner comment notification" "${ACCESS_TOKEN}" "comment_created"

echo "Confirming the owner has unread notifications..."
OWNER_UNREAD_BEFORE="$(unread_count "${ACCESS_TOKEN}")"
if [[ "${OWNER_UNREAD_BEFORE}" -lt 1 ]]; then
  echo "Owner unread notification count should be greater than zero, got ${OWNER_UNREAD_BEFORE}." >&2
  exit 1
fi

echo "Marking one owner notification read and confirming the unread count decreases..."
fetch_notifications "${ACCESS_TOKEN}"
FIRST_UNREAD_ID="$(jq -r '[.items[] | select(.readAt == null)][0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${FIRST_UNREAD_ID}" ]]; then
  echo "Owner has no unread notification to mark read." >&2
  exit 1
fi
request_with_bearer PATCH "${NOTIFICATION_SERVICE_URL}/notifications/${FIRST_UNREAD_ID}/read" "${ACCESS_TOKEN}"
assert_2xx "Mark notification read"
OWNER_UNREAD_AFTER_ONE="$(unread_count "${ACCESS_TOKEN}")"
if [[ "${OWNER_UNREAD_AFTER_ONE}" -ge "${OWNER_UNREAD_BEFORE}" ]]; then
  echo "Unread count did not decrease after marking one read (${OWNER_UNREAD_BEFORE} -> ${OWNER_UNREAD_AFTER_ONE})." >&2
  exit 1
fi

echo "Confirming mark-read is idempotent..."
request_with_bearer PATCH "${NOTIFICATION_SERVICE_URL}/notifications/${FIRST_UNREAD_ID}/read" "${ACCESS_TOKEN}"
assert_2xx "Mark notification read (idempotent)"

echo "Confirming a user cannot mark another user's notification read..."
request_with_bearer PATCH "${NOTIFICATION_SERVICE_URL}/notifications/${FIRST_UNREAD_ID}/read" "${COLLAB_ACCESS_TOKEN}"
assert_status "Cross-user mark read blocked" "404"

echo "Marking all owner notifications read and confirming the unread count is zero..."
request_with_bearer PATCH "${NOTIFICATION_SERVICE_URL}/notifications/read-all" "${ACCESS_TOKEN}"
assert_2xx "Mark all notifications read"
OWNER_UNREAD_FINAL="$(unread_count "${ACCESS_TOKEN}")"
if [[ "${OWNER_UNREAD_FINAL}" -ne 0 ]]; then
  echo "Unread count should be zero after mark-all-read, got ${OWNER_UNREAD_FINAL}." >&2
  exit 1
fi

echo "Updating the viewer's own comment..."
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments/${COLLAB_COMMENT_ID}" "${COLLAB_ACCESS_TOKEN}" '{"body":"Viewer: edited - lets confirm timing."}'
assert_2xx "Viewer update own comment"
UPDATED_COMMENT_BODY="$(jq -r '.body // empty' <<<"${LAST_BODY}")"
if [[ "${UPDATED_COMMENT_BODY}" != "Viewer: edited - lets confirm timing." ]]; then
  echo "Viewer comment was not updated." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Rejecting an empty comment body..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${COLLAB_ACCESS_TOKEN}" '{"dayNumber":1,"itemIndex":0,"body":"   "}'
assert_status "Whitespace comment rejected" "400"

echo "Rejecting a comment on a non-existent itinerary item..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${ACCESS_TOKEN}" '{"dayNumber":99,"itemIndex":0,"body":"Nowhere"}'
assert_status "Comment on missing item rejected" "400"

echo "Confirming comment counts include all active comments..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments/counts" "${ACCESS_TOKEN}"
assert_2xx "List comment counts"
DAY1_ITEM0_COUNT="$(jq -r '[.items[] | select(.dayNumber == 1 and .itemIndex == 0) | .count] | first // 0' <<<"${LAST_BODY}")"
if [[ "${DAY1_ITEM0_COUNT}" != "4" ]]; then
  echo "Comment counts did not report 4 active comments on day 1 item 0." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming a collaborator cannot delete the owner's comment..."
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments/${OWNER_COMMENT_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_status "Collaborator delete owner comment" "403"

echo "Confirming the owner can delete the collaborator's comment..."
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments/${COLLAB_COMMENT_ID}" "${ACCESS_TOKEN}"
assert_2xx "Owner delete collaborator comment"

echo "Confirming the soft-deleted comment is no longer returned..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${ACCESS_TOKEN}"
assert_2xx "List trip comments after delete"
DELETED_STILL_PRESENT="$(jq --arg id "${COLLAB_COMMENT_ID}" '[.items[] | select(.id == $id)] | length' <<<"${LAST_BODY}")"
if [[ "${DELETED_STILL_PRESENT}" != "0" ]]; then
  echo "Soft-deleted comment was still returned." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming comments require authentication..."
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments"
assert_status "Unauthenticated comment list" "401"

echo "Changing collaborator role to editor..."
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/collaborators/${COLLABORATOR_ID}" "${ACCESS_TOKEN}" '{"role":"editor"}'
assert_2xx "Update collaborator role to editor"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${COLLAB_ACCESS_TOKEN}" '{"message":"Is my route ready?"}'
assert_2xx "Editor get Copilot guidance"
if ! jq -e '[.actions[]?.type] | index("find_transport") != null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Editor Copilot response did not retain the permitted transport action." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking advisory itinerary edit locks..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_2xx "Owner acquire edit lock"
OWNER_LOCK_ACQUIRED="$(jq -r '.acquired // false' <<<"${LAST_BODY}")"
OWNER_LOCK_USER_ID="$(jq -r '.lock.lockedByUserId // empty' <<<"${LAST_BODY}")"
OWNER_LOCK_CURRENT="$(jq -r '.lock.lockedByCurrentUser // false' <<<"${LAST_BODY}")"
if [[ "${OWNER_LOCK_ACQUIRED}" != "true" || "${OWNER_LOCK_USER_ID}" != "${OWNER_USER_ID}" || "${OWNER_LOCK_CURRENT}" != "true" ]]; then
  echo "Owner edit lock acquire response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_2xx "Owner renew edit lock"
OWNER_LOCK_RENEWED="$(jq -r '.renewed // false' <<<"${LAST_BODY}")"
if [[ "${OWNER_LOCK_RENEWED}" != "true" ]]; then
  echo "Owner edit lock renew response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_status "Editor edit lock conflict" "409"
EDIT_LOCK_ERROR="$(jq -r '.error // empty' <<<"${LAST_BODY}")"
EDIT_LOCK_REASON="$(jq -r '.reason // empty' <<<"${LAST_BODY}")"
EDIT_LOCK_OWNER="$(jq -r '.lock.lockedByUserId // empty' <<<"${LAST_BODY}")"
if [[ "${EDIT_LOCK_ERROR}" != "edit_lock_conflict" || "${EDIT_LOCK_REASON}" != "locked_by_other_user" || "${EDIT_LOCK_OWNER}" != "${OWNER_USER_ID}" ]]; then
  echo "Editor edit lock conflict response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}"
assert_2xx "Editor read owner edit lock"
EDITOR_SEES_OWNER_LOCK="$(jq -r '.lockedByUserId // empty' <<<"${LAST_BODY}")"
EDITOR_LOCK_CURRENT="$(jq -r '.lockedByCurrentUser // true' <<<"${LAST_BODY}")"
if [[ "${EDITOR_SEES_OWNER_LOCK}" != "${OWNER_USER_ID}" || "${EDITOR_LOCK_CURRENT}" != "false" ]]; then
  echo "Editor did not see the owner edit lock as expected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_2xx "Owner release edit lock"
OWNER_LOCK_RELEASED="$(jq -r '.released // false' <<<"${LAST_BODY}")"
if [[ "${OWNER_LOCK_RELEASED}" != "true" ]]; then
  echo "Owner edit lock release response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_2xx "Editor acquire edit lock after owner release"
EDITOR_LOCK_ACQUIRED="$(jq -r '.acquired // false' <<<"${LAST_BODY}")"
EDITOR_LOCK_USER_ID="$(jq -r '.lock.lockedByUserId // empty' <<<"${LAST_BODY}")"
if [[ "${EDITOR_LOCK_ACQUIRED}" != "true" || "${EDITOR_LOCK_USER_ID}" != "${COLLAB_USER_ID}" ]]; then
  echo "Editor edit lock acquire response was unexpected." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${COLLAB_ACCESS_TOKEN}" '{"scope":"itinerary"}'
assert_2xx "Editor release edit lock"

EDITOR_EDIT_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{
  expectedItineraryRevision: $revision,
  itinerary: {
    days: [
      {
        day: 1,
        title: "Editor Smoke Test Day",
        items: [{time:"10:00",type:"activity",name:"Editor Smoke Test Activity"}]
      }
    ]
  }
}')"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${COLLAB_ACCESS_TOKEN}" "${EDITOR_EDIT_PAYLOAD}"
assert_2xx "Editor itinerary edit"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

echo "Checking editor cannot manage public share settings..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/share" "${COLLAB_ACCESS_TOKEN}"
assert_status "Editor public share create" "403"

echo "Checking editor version attribution..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List versions after editor edit"

LATEST_VERSION_ACTOR="$(jq -r '.items[0].createdByUserId // empty' <<<"${LAST_BODY}")"
if [[ "${LATEST_VERSION_ACTOR}" != "${COLLAB_USER_ID}" ]]; then
  echo "Latest itinerary version did not record collaborator actor." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Removing collaborator and confirming access is revoked..."
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/collaborators/${COLLABORATOR_ID}" "${ACCESS_TOKEN}"
assert_2xx "Remove collaborator"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${COLLAB_ACCESS_TOKEN}"
assert_status "Removed collaborator private trip access" "404"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${COLLAB_ACCESS_TOKEN}" '{"message":"What should I fix first?"}'
assert_status "Removed collaborator Copilot access" "404"

echo "Confirming removed collaborator cannot list comments..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${COLLAB_ACCESS_TOKEN}"
assert_status "Removed collaborator comment access" "404"

echo "Confirming removed collaborator cannot fetch activity..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity" "${COLLAB_ACCESS_TOKEN}"
assert_status "Removed collaborator activity access" "404"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream" "${COLLAB_ACCESS_TOKEN}"
assert_status "Removed collaborator activity stream access" "404"

echo "Confirming removed collaborator cannot update presence..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${COLLAB_ACCESS_TOKEN}" '{"state":"viewing"}'
assert_status "Removed collaborator presence access" "404"

echo "Creating password-protected public share link..."
FUTURE_EXPIRES_AT="$(python3 -c 'from datetime import datetime, timezone, timedelta; print((datetime.now(timezone.utc) + timedelta(days=7)).isoformat().replace("+00:00", "Z"))')"
SHARE_PASSWORD="ShareSecret123"
CREATE_SHARE_PAYLOAD="$(jq -nc --arg expiresAt "${FUTURE_EXPIRES_AT}" --arg password "${SHARE_PASSWORD}" '{expiresAt:$expiresAt,password:$password}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/share" "${ACCESS_TOKEN}" "${CREATE_SHARE_PAYLOAD}"
assert_2xx "Create trip share"

SHARE_TOKEN="$(jq -r '.shareToken // empty' <<<"${LAST_BODY}")"
SHARE_URL="$(jq -r '.shareUrl // empty' <<<"${LAST_BODY}")"
SHARE_ENABLED="$(jq -r '.enabled // false' <<<"${LAST_BODY}")"
SHARE_PASSWORD_REQUIRED="$(jq -r '.passwordRequired // false' <<<"${LAST_BODY}")"
SHARE_EXPIRES_AT="$(jq -r '.expiresAt // empty' <<<"${LAST_BODY}")"
if [[ -z "${SHARE_TOKEN}" || "${#SHARE_TOKEN}" -lt 43 || "${SHARE_ENABLED}" != "true" || "${SHARE_PASSWORD_REQUIRED}" != "true" || -z "${SHARE_EXPIRES_AT}" ]]; then
  echo "Share response did not include an enabled protected token with expiration." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ -z "${SHARE_URL}" ]]; then
  echo "Share response did not include shareUrl." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if jq -e 'has("passwordHash")' >/dev/null <<<"${LAST_BODY}"; then
  echo "Share response exposed passwordHash." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking public share status requires a password..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/status"
assert_2xx "Public share status"

STATUS_PASSWORD_REQUIRED="$(jq -r '.passwordRequired // false' <<<"${LAST_BODY}")"
if [[ "${STATUS_PASSWORD_REQUIRED}" != "true" ]]; then
  echo "Public share status did not report passwordRequired=true." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming protected public share requires unlock..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}"
assert_status "Protected public shared trip without token" "401"

echo "Checking wrong public share password is rejected..."
request POST "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/unlock" '{"password":"wrong-password"}'
assert_status "Wrong public share password" "401"

echo "Unlocking public share with correct password..."
UNLOCK_PAYLOAD="$(jq -nc --arg password "${SHARE_PASSWORD}" '{password:$password}')"
request POST "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/unlock" "${UNLOCK_PAYLOAD}"
assert_2xx "Unlock public share"

PUBLIC_SHARE_ACCESS_TOKEN="$(jq -r '.accessToken // empty' <<<"${LAST_BODY}")"
PUBLIC_SHARE_ACCESS_EXPIRES_AT="$(jq -r '.expiresAt // empty' <<<"${LAST_BODY}")"
if [[ -z "${PUBLIC_SHARE_ACCESS_TOKEN}" || -z "${PUBLIC_SHARE_ACCESS_EXPIRES_AT}" ]]; then
  echo "Unlock response did not include public share access token and expiry." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching unlocked public shared trip..."
request_with_bearer GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_2xx "Fetch public shared trip"
PUBLIC_TRIP_BODY="${LAST_BODY}"

echo "Confirming public share token cannot access edit locks..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/edit-lock" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share edit lock access" "401"

echo "Confirming public share token cannot open the private activity stream..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share activity stream access" "401"

echo "Confirming public share token cannot access budget optimization proposals..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-optimization-proposals" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share budget optimization proposal access" "401"

echo "Confirming public share token cannot access private decision surfaces..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/polls" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share polls access" "401"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/reactions" "${PUBLIC_SHARE_ACCESS_TOKEN}" '{"dayNumber":1,"itemIndex":0,"reaction":"skip"}'
assert_status "Public share reaction access" "401"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/me" "${PUBLIC_SHARE_ACCESS_TOKEN}" '{"availableRanges":[{"startDate":"2026-09-12","endDate":"2026-09-13"}]}'
assert_status "Public share availability submit access" "401"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/availability/import-calendar/preview" "${PUBLIC_SHARE_ACCESS_TOKEN}" '{"startDate":"2026-09-01","endDate":"2026-09-30","timezone":"Europe/Bratislava","calendarProvider":"google"}'
assert_status "Public share calendar availability import access" "401"

echo "Confirming public share token cannot access private checklist surfaces..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share checklist access" "401"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${MANUAL_CHECKLIST_ITEM_ID}/check" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share checklist check access" "401"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/health" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share trip health access" "401"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share group readiness access" "401"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-confidence" "${PUBLIC_SHARE_ACCESS_TOKEN}"
assert_status "Public share budget confidence access" "401"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/copilot/chat" "${PUBLIC_SHARE_ACCESS_TOKEN}" '{"message":"What should I fix first?"}'
assert_status "Public share Copilot access" "401"

PUBLIC_DESTINATION="$(jq -r '.destination // empty' <<<"${PUBLIC_TRIP_BODY}")"
PUBLIC_ITINERARY_DAYS="$(jq '.itinerary.days | length' <<<"${PUBLIC_TRIP_BODY}")"
if [[ "${PUBLIC_DESTINATION}" != "Rome" || "${PUBLIC_ITINERARY_DAYS}" -le 0 ]]; then
  echo "Public shared trip did not include expected destination and itinerary." >&2
  echo "${PUBLIC_TRIP_BODY}" >&2
  exit 1
fi
if jq -e 'has("userId") or has("email") or has("versionHistory") or has("comments") or has("checklist") or has("accommodation") or has("budget") or has("budgetAmount") or has("budgetCurrency")' >/dev/null <<<"${PUBLIC_TRIP_BODY}"; then
  echo "Public shared trip exposed private fields." >&2
  echo "${PUBLIC_TRIP_BODY}" >&2
  exit 1
fi
if jq -e '[.. | objects | select(has("priceEnrichment"))] | length > 0' >/dev/null <<<"${PUBLIC_TRIP_BODY}"; then
  echo "Public shared trip exposed price enrichment metadata." >&2
  echo "${PUBLIC_TRIP_BODY}" >&2
  exit 1
fi

echo "Deleting accommodation and confirming it clears budget summary."
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${ACCESS_TOKEN}"
assert_2xx "Delete accommodation"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/accommodation" "${ACCESS_TOKEN}"
assert_2xx "Get accommodation after delete"
if ! jq -e '.accommodation == null' <<<"${LAST_BODY}" >/dev/null; then
  echo "Expected accommodation to be null after delete." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-summary" "${ACCESS_TOKEN}"
assert_2xx "Budget summary after accommodation delete"
if ! jq -e '(.accommodationTotal == null) or (.accommodationTotal == 0)' <<<"${LAST_BODY}" >/dev/null; then
  echo "Budget summary still included accommodationTotal after delete." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Confirming the public share has no comments endpoint..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/comments"
assert_status "Public share comments endpoint absent" "404"

echo "Clearing public share password and expiration..."
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/share" "${ACCESS_TOKEN}" '{"clearPassword":true,"clearExpiration":true}'
assert_2xx "Clear public share password"

CLEARED_PASSWORD_REQUIRED="$(jq -r '.passwordRequired // true' <<<"${LAST_BODY}")"
CLEARED_EXPIRES_AT="$(jq -r '.expiresAt // empty' <<<"${LAST_BODY}")"
if [[ "${CLEARED_PASSWORD_REQUIRED}" != "false" || -n "${CLEARED_EXPIRES_AT}" ]]; then
  echo "Share settings were not cleared." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching public shared trip after password removal..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}"
assert_2xx "Fetch public shared trip after password removal"

echo "Confirming past share expiration is rejected..."
PAST_EXPIRES_AT="$(python3 -c 'from datetime import datetime, timezone, timedelta; print((datetime.now(timezone.utc) - timedelta(minutes=5)).isoformat().replace("+00:00", "Z"))')"
PAST_EXPIRATION_PAYLOAD="$(jq -nc --arg expiresAt "${PAST_EXPIRES_AT}" '{expiresAt:$expiresAt}')"
request_with_bearer PATCH "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/share" "${ACCESS_TOKEN}" "${PAST_EXPIRATION_PAYLOAD}"
assert_status "Past public share expiration" "400"

echo "Disabling public share link..."
request_with_bearer DELETE "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/share" "${ACCESS_TOKEN}"
assert_2xx "Disable trip share"

echo "Confirming disabled public share status and trip return 404..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/status"
assert_status "Disabled public shared trip status" "404"

request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}"
assert_status "Disabled public shared trip" "404"

AUTO_MATCHED_GENERATED_ITEMS="$(jq '[.itinerary.days[]?.items[]? | select(.place != null and .placeEnrichment.status == "matched")] | length' <<<"${COMPLETED_TRIP_BODY}")"
if [[ "${AUTO_MATCHED_GENERATED_ITEMS}" -gt 0 ]]; then
  if ! jq -e '
    [.itinerary.days[]?.items[]? | select(.place != null and .placeEnrichment.status == "matched")]
    | all(.[]; (.place.provider // "") != "" and (.place.providerPlaceId // "") != "" and (.place.name // "") != "" and (.placeEnrichment.confidence // 0) >= 0.75)
  ' >/dev/null <<<"${COMPLETED_TRIP_BODY}"; then
    echo "Generated itinerary auto-matched place metadata is incomplete." >&2
    echo "${COMPLETED_TRIP_BODY}" >&2
    exit 1
  fi
  echo "Generated itinerary has ${AUTO_MATCHED_GENERATED_ITEMS} auto-matched place item(s)."
else
  echo "Generated itinerary has no high-confidence auto-matched places; continuing because AI wording can vary."
fi

PROVIDER_PRICED_ITEMS="$(jq '[.itinerary.days[]?.items[]? | select(.estimatedCost.source == "provider")] | length' <<<"${COMPLETED_TRIP_BODY}")"
if [[ "${PROVIDER_PRICED_ITEMS}" -gt 0 ]]; then
  if ! jq -e '
    [.itinerary.days[]?.items[]? | select(.estimatedCost.source == "provider")]
    | all(.[]; (.estimatedCost.amount // 0) > 0 and (.estimatedCost.currency // "") != "" and (.priceEnrichment.status // "") == "matched" and (.priceEnrichment.matchConfidence // 0) >= 0.55)
  ' >/dev/null <<<"${COMPLETED_TRIP_BODY}"; then
    echo "Generated itinerary provider price metadata is incomplete." >&2
    echo "${COMPLETED_TRIP_BODY}" >&2
    exit 1
  fi
  echo "Generated itinerary has ${PROVIDER_PRICED_ITEMS} provider-priced item(s)."
else
  echo "Generated itinerary has no provider-priced items; direct price endpoint checks already passed."
fi

echo "Listing itinerary versions after generation..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List itinerary versions after generation"

VERSION_COUNT_AFTER_GENERATE="$(jq '.items | length' <<<"${LAST_BODY}")"
GENERATED_VERSION_ID="$(jq -r '[.items[] | select(.source == "GENERATED")][0].id // empty' <<<"${LAST_BODY}")"
GENERATED_VERSION_SOURCE="$(jq -r '[.items[] | select(.source == "GENERATED")][0].source // empty' <<<"${LAST_BODY}")"
if [[ "${VERSION_COUNT_AFTER_GENERATE}" -lt 1 || -z "${GENERATED_VERSION_ID}" || "${GENERATED_VERSION_SOURCE}" != "GENERATED" ]]; then
  echo "Expected at least one GENERATED itinerary version after generation." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

ITINERARY_TEXT="$(jq -r '.itinerary | .. | scalars? | tostring' <<<"${COMPLETED_TRIP_BODY}" | tr '\n' ' ')"
if grep -Eiq '\bnightclubs?\b' <<<"${ITINERARY_TEXT}"; then
  echo "Generated itinerary mentioned an avoided nightlife term; continuing because AI wording can vary." >&2
fi
echo "Personalization context path exercised through Trip Service -> User Service -> AI Planning Service."

echo "Editing itinerary with Authorization header..."
EDIT_ITINERARY_PAYLOAD="$(
  jq -nc --argjson place "${PLACE_JSON}" --argjson revision "${TRIP_REVISION}" '{
    expectedItineraryRevision: $revision,
    itinerary: {
      days: [
        {
          day: 1,
          title: "Edited Smoke Test Day",
          items: [
            {
              time: "10:00",
              type: "activity",
              name: "Edited Smoke Test Activity",
              note: "Updated by smoke test",
              estimatedCost: 12,
              place: $place
            }
          ]
        }
      ]
    }
  }'
)"
STALE_EDIT_ITINERARY_PAYLOAD="$(
  jq -nc --argjson place "${PLACE_JSON}" --argjson revision "$((TRIP_REVISION - 1))" '{
    expectedItineraryRevision: $revision,
    itinerary: {
      days: [
        {
          day: 1,
          title: "Stale Smoke Test Day",
          items: [
            {
              time: "10:00",
              type: "activity",
              name: "Stale Smoke Test Activity",
              place: $place
            }
          ]
        }
      ]
    }
  }'
)"
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${ACCESS_TOKEN}" "${STALE_EDIT_ITINERARY_PAYLOAD}"
assert_status "Stale itinerary edit" "409"
CONFLICT_ERROR="$(jq -r '.error // empty' <<<"${LAST_BODY}")"
CONFLICT_REVISION="$(jq -r '.currentItineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${CONFLICT_ERROR}" != "itinerary_conflict" || "${CONFLICT_REVISION}" != "${TRIP_REVISION}" ]]; then
  echo "Stale itinerary edit did not return expected conflict payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${ACCESS_TOKEN}" "${EDIT_ITINERARY_PAYLOAD}"
assert_2xx "Edit itinerary"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

EDIT_STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
EDIT_TITLE="$(jq -r '.itinerary.days[0].title // empty' <<<"${LAST_BODY}")"
EDIT_ITEM_NAME="$(jq -r '.itinerary.days[0].items[0].name // empty' <<<"${LAST_BODY}")"
EDIT_PLACE_ID="$(jq -r '.itinerary.days[0].items[0].place.providerPlaceId // empty' <<<"${LAST_BODY}")"
EDIT_OPENING_HOURS_COUNT="$(jq '.itinerary.days[0].items[0].place.openingHours | length' <<<"${LAST_BODY}")"
if [[ "${EDIT_STATUS}" != "COMPLETED" || "${EDIT_TITLE}" != "Edited Smoke Test Day" || "${EDIT_ITEM_NAME}" != "Edited Smoke Test Activity" || "${EDIT_PLACE_ID}" != "${PLACE_ID}" ]]; then
  echo "Edited itinerary response did not include expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${PLACE_PROVIDER_NAME}" == "mock" && "${EDIT_OPENING_HOURS_COUNT}" -lt 1 ]]; then
  echo "Edited itinerary response did not preserve mock openingHours." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching edited trip to confirm place metadata persisted..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch edited trip"

FETCHED_PLACE_ID="$(jq -r '.itinerary.days[0].items[0].place.providerPlaceId // empty' <<<"${LAST_BODY}")"
FETCHED_OPENING_HOURS_COUNT="$(jq '.itinerary.days[0].items[0].place.openingHours | length' <<<"${LAST_BODY}")"
if [[ "${FETCHED_PLACE_ID}" != "${PLACE_ID}" ]]; then
  echo "Fetched trip did not preserve attached place metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${PLACE_PROVIDER_NAME}" == "mock" && "${FETCHED_OPENING_HOURS_COUNT}" -lt 1 ]]; then
  echo "Fetched trip did not preserve mock openingHours." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Listing itinerary versions after manual edit..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List itinerary versions after manual edit"

VERSION_COUNT_AFTER_EDIT="$(jq '.items | length' <<<"${LAST_BODY}")"
MANUAL_VERSION_COUNT="$(jq '[.items[] | select(.source == "MANUAL_EDIT")] | length' <<<"${LAST_BODY}")"
MANUAL_VERSION_ID="$(jq -r '[.items[] | select(.source == "MANUAL_EDIT")][0].id // empty' <<<"${LAST_BODY}")"
if [[ "${VERSION_COUNT_AFTER_EDIT}" -le "${VERSION_COUNT_AFTER_GENERATE}" || "${MANUAL_VERSION_COUNT}" -lt 1 ]]; then
  echo "Expected manual edit to add a MANUAL_EDIT itinerary version." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching manual edit itinerary version detail..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions/${MANUAL_VERSION_ID}" "${ACCESS_TOKEN}"
assert_2xx "Get manual edit itinerary version"

MANUAL_VERSION_PLACE_ID="$(jq -r '.itinerary.days[0].items[0].place.providerPlaceId // empty' <<<"${LAST_BODY}")"
MANUAL_VERSION_OPENING_HOURS_COUNT="$(jq '.itinerary.days[0].items[0].place.openingHours | length' <<<"${LAST_BODY}")"
if [[ "${MANUAL_VERSION_PLACE_ID}" != "${PLACE_ID}" ]]; then
  echo "Manual edit version did not store attached place metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "${PLACE_PROVIDER_NAME}" == "mock" && "${MANUAL_VERSION_OPENING_HOURS_COUNT}" -lt 1 ]]; then
  echo "Manual edit version did not preserve mock openingHours." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching generated itinerary version detail..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions/${GENERATED_VERSION_ID}" "${ACCESS_TOKEN}"
assert_2xx "Get generated itinerary version"

GENERATED_VERSION_TITLE="$(jq -r '.itinerary.days[0].title // empty' <<<"${LAST_BODY}")"
if [[ -z "${GENERATED_VERSION_TITLE}" ]]; then
  echo "Generated version detail did not include an itinerary day title." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Restoring generated itinerary version..."
RESTORE_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{expectedItineraryRevision:$revision}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions/${GENERATED_VERSION_ID}/restore" "${ACCESS_TOKEN}" "${RESTORE_PAYLOAD}"
assert_2xx "Restore generated itinerary version"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

RESTORED_TITLE="$(jq -r '.itinerary.days[0].title // empty' <<<"${LAST_BODY}")"
if [[ "${RESTORED_TITLE}" != "${GENERATED_VERSION_TITLE}" ]]; then
  echo "Restored itinerary did not match generated version title." >&2
  echo "Expected: ${GENERATED_VERSION_TITLE}" >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Fetching restored trip with Authorization header..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch restored trip"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

EDITED_STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
EDITED_TITLE="$(jq -r '.itinerary.days[0].title // empty' <<<"${LAST_BODY}")"
if [[ "${EDITED_STATUS}" != "COMPLETED" || "${EDITED_TITLE}" != "${GENERATED_VERSION_TITLE}" ]]; then
  echo "Restored itinerary did not persist after fetch." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking RESTORED itinerary version exists..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions" "${ACCESS_TOKEN}"
assert_2xx "List itinerary versions after restore"

VERSION_COUNT_AFTER_RESTORE="$(jq '.items | length' <<<"${LAST_BODY}")"
RESTORED_VERSION_COUNT="$(jq '[.items[] | select(.source == "RESTORED")] | length' <<<"${LAST_BODY}")"
if [[ "${VERSION_COUNT_AFTER_RESTORE}" -le "${VERSION_COUNT_AFTER_EDIT}" || "${RESTORED_VERSION_COUNT}" -lt 1 ]]; then
  echo "Expected restore to append a RESTORED itinerary version." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Checking day regeneration revision conflicts..."
STALE_DAY_REGEN_PAYLOAD="$(jq -nc --argjson revision "$((TRIP_REVISION - 1))" '{expectedItineraryRevision:$revision,instruction:"make day one slower paced"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/days/1/regenerate" "${ACCESS_TOKEN}" "${STALE_DAY_REGEN_PAYLOAD}"
assert_status "Stale day regeneration" "409"
CONFLICT_ERROR="$(jq -r '.error // empty' <<<"${LAST_BODY}")"
CONFLICT_REVISION="$(jq -r '.currentItineraryRevision // -1' <<<"${LAST_BODY}")"
if [[ "${CONFLICT_ERROR}" != "itinerary_conflict" || "${CONFLICT_REVISION}" != "${TRIP_REVISION}" ]]; then
  echo "Stale day regeneration did not return expected conflict payload." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
DAY_REGEN_JOB_PAYLOAD="$(jq -nc --argjson revision "${TRIP_REVISION}" '{jobType:"day_regeneration",dayNumber:1,expectedItineraryRevision:$revision,instruction:"make day one slower paced"}')"
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generation-jobs" "${ACCESS_TOKEN}" "${DAY_REGEN_JOB_PAYLOAD}"
assert_status "Create day regeneration job" "202"
DAY_REGEN_JOB_ID="$(jq -r '.job.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DAY_REGEN_JOB_ID}" ]]; then
  echo "Day regeneration job response did not include a job id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
poll_generation_job "Day regeneration" "${TRIP_ID}" "${DAY_REGEN_JOB_ID}" "${ACCESS_TOKEN}"
DAY_REGEN_JOB_STATUS="$(jq -r '.job.status // empty' <<<"${LAST_BODY}")"
if [[ "${DAY_REGEN_JOB_STATUS}" != "completed" ]]; then
  echo "Day regeneration job did not complete successfully." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip after day regeneration job"
TRIP_REVISION="$(jq -r '.itineraryRevision // -1' <<<"${LAST_BODY}")"

echo "Listing trips for current user..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips?limit=20&offset=0" "${ACCESS_TOKEN}"
assert_2xx "List trips"

MATCHING_TRIPS="$(jq --arg id "${TRIP_ID}" '[.items[] | select(.id == $id)] | length' <<<"${LAST_BODY}")"
if [[ "${MATCHING_TRIPS}" -ne 1 ]]; then
  echo "Expected authenticated list to include trip ${TRIP_ID} exactly once." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

echo "Exercising AI Trip Discovery..."
PRAGUE_TRIP_PAYLOAD='{
  "destination":"Prague, Czechia",
  "days":3,
  "budgetAmount":450,
  "budgetCurrency":"EUR",
  "travelers":2,
  "interests":["city","food","architecture"],
  "pace":"balanced"
}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips" "${ACCESS_TOKEN}" "${PRAGUE_TRIP_PAYLOAD}"
assert_2xx "Create previous Prague trip for discovery context"

SURPRISE_PAYLOAD='{
  "scope":"personal",
  "durationDays":3,
  "budget":{"amount":500,"currency":"EUR"},
  "travelers":1,
  "origin":"Bratislava, Slovakia",
  "outputLanguage":"en",
  "noveltyLevel":"balanced",
  "avoidPreviouslyVisited":true
}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-discovery/surprise-me" "${ACCESS_TOKEN}" "${SURPRISE_PAYLOAD}"
assert_2xx "Trip discovery surprise me"
if [[ "$(jq '.response.suggestions | length' <<<"${LAST_BODY}")" -lt 1 ]]; then
  echo "Surprise Me returned no suggestions." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "$(jq -r '.response.suggestions[0].city' <<<"${LAST_BODY}")" == "Prague" ]]; then
  echo "Surprise Me repeated Prague as the first suggestion." >&2
  exit 1
fi
if [[ "$(jq -r '.createdTripId // empty' <<<"${LAST_BODY}")" != "" ]]; then
  echo "Surprise Me created a trip without confirmation." >&2
  exit 1
fi

DISCOVERY_PROMPT_PAYLOAD='{
  "prompt":"cheap warm food weekend",
  "scope":"personal",
  "durationDays":3,
  "budget":{"amount":700,"currency":"EUR"},
  "travelers":2,
  "origin":"Bratislava, Slovakia",
  "quickChips":["warm","food","low_budget"],
  "outputLanguage":"en",
  "avoidPreviouslyVisited":true,
  "preferNovelty":true
}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-discovery/suggestions" "${ACCESS_TOKEN}" "${DISCOVERY_PROMPT_PAYLOAD}"
assert_2xx "Trip discovery prompt suggestions"
DISCOVERY_SESSION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DISCOVERY_SESSION_ID}" ]]; then
  echo "Trip discovery response did not include a session id." >&2
  exit 1
fi

request_with_bearer POST "${TRIP_SERVICE_URL}/trip-discovery/${DISCOVERY_SESSION_ID}/refine" "${ACCESS_TOKEN}" '{"instruction":"cheaper and more nature","feedbackType":"too_expensive","outputLanguage":"en"}'
assert_2xx "Refine trip discovery"
DISCOVERY_SESSION_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
DISCOVERY_SUGGESTION_ID="$(jq -r '.response.suggestions[0].id // empty' <<<"${LAST_BODY}")"
if [[ -z "${DISCOVERY_SESSION_ID}" || -z "${DISCOVERY_SUGGESTION_ID}" ]]; then
  echo "Refined discovery response was incomplete." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

DISCOVERY_CREATE_PAYLOAD='{
  "durationDays":3,
  "budget":{"amount":450,"currency":"EUR"},
  "travelers":2,
  "autoGenerateItinerary":false
}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trip-discovery/${DISCOVERY_SESSION_ID}/suggestions/${DISCOVERY_SUGGESTION_ID}/create-trip" "${ACCESS_TOKEN}" "${DISCOVERY_CREATE_PAYLOAD}"
assert_2xx "Create trip from discovery suggestion"
if [[ "$(jq -r '.trip.creationMetadata.creationSource // empty' <<<"${LAST_BODY}")" != "trip_discovery" ]]; then
  echo "Discovery-created trip did not store source metadata." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
if [[ "$(jq -r '.generationJob // empty' <<<"${LAST_BODY}")" != "" ]]; then
  echo "Discovery trip unexpectedly created a generation job." >&2
  exit 1
fi

echo "Verifying another user cannot access the first user's trip..."
echo "Checking private data export and cleanup contracts..."
TRIP_EXPORT_PAYLOAD='{"includeReceiptFiles":false,"includeRecapPdf":false,"includePrivateNotes":false}'
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/export/archive" "${ACCESS_TOKEN}" "${TRIP_EXPORT_PAYLOAD}"
assert_status "Create private trip export" "202"
TRIP_EXPORT_ID="$(jq -r '.exportId // empty' <<<"${LAST_BODY}")"
if [[ -z "${TRIP_EXPORT_ID}" ]]; then
  echo "Trip export response did not include an exportId." >&2
  exit 1
fi
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/export/${TRIP_EXPORT_ID}" "${ACCESS_TOKEN}"
assert_2xx "Read private trip export status"
if [[ "$(jq -r '.status // empty' <<<"${LAST_BODY}")" != "completed" ]]; then
  echo "Trip export did not complete in the synchronous v1 flow." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
TRIP_EXPORT_HEADERS="$(mktemp)"
TRIP_EXPORT_DOWNLOAD_STATUS="$(curl -sS -o /dev/null -D "${TRIP_EXPORT_HEADERS}" -w "%{http_code}" -H "Authorization: Bearer ${ACCESS_TOKEN}" "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/export/${TRIP_EXPORT_ID}/download")"
if [[ "${TRIP_EXPORT_DOWNLOAD_STATUS}" != "200" ]] || ! grep -qi '^Cache-Control: private, no-store' "${TRIP_EXPORT_HEADERS}"; then
  rm -f "${TRIP_EXPORT_HEADERS}"
  echo "Private trip export download was unavailable or cacheable." >&2
  exit 1
fi
rm -f "${TRIP_EXPORT_HEADERS}"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/expenses/export.csv" "${ACCESS_TOKEN}"
assert_2xx "Download private expense CSV"
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/export/${TRIP_EXPORT_ID}"
assert_status "Unauthenticated trip export status" "401"

ACCOUNT_EXPORT_PAYLOAD='{"sections":{"profile":true,"preferences":true},"includeReceiptFiles":false,"includeWorkspaceData":false}'
request_with_bearer POST "${USER_SERVICE_URL}/users/me/export" "${ACCESS_TOKEN}" "${ACCOUNT_EXPORT_PAYLOAD}"
assert_status "Create private account export" "202"
ACCOUNT_EXPORT_ID="$(jq -r '.exportId // empty' <<<"${LAST_BODY}")"
if [[ -z "${ACCOUNT_EXPORT_ID}" ]]; then
  echo "Account export response did not include an exportId." >&2
  exit 1
fi
request_with_bearer GET "${USER_SERVICE_URL}/users/me/export/${ACCOUNT_EXPORT_ID}" "${ACCESS_TOKEN}"
assert_2xx "Read private account export status"
ACCOUNT_EXPORT_HEADERS="$(mktemp)"
ACCOUNT_EXPORT_DOWNLOAD_STATUS="$(curl -sS -o /dev/null -D "${ACCOUNT_EXPORT_HEADERS}" -w "%{http_code}" -H "Authorization: Bearer ${ACCESS_TOKEN}" "${USER_SERVICE_URL}/users/me/export/${ACCOUNT_EXPORT_ID}/download")"
if [[ "${ACCOUNT_EXPORT_DOWNLOAD_STATUS}" != "200" ]] || ! grep -qi '^Cache-Control: private, no-store' "${ACCOUNT_EXPORT_HEADERS}"; then
  rm -f "${ACCOUNT_EXPORT_HEADERS}"
  echo "Private account export download was unavailable or cacheable." >&2
  exit 1
fi
rm -f "${ACCOUNT_EXPORT_HEADERS}"
request_with_bearer POST "${USER_SERVICE_URL}/users/me/account-cleanup/request-deletion" "${ACCESS_TOKEN}" '{"reason":"smoke test","exportRequestedFirst":true}'
assert_status "Record account cleanup request without deletion" "202"

request_with_bearer POST "${NOTIFICATION_SERVICE_URL}/notifications/cleanup" "${ACCESS_TOKEN}" '{"olderThanDays":0,"onlyRead":true}'
assert_2xx "Clean up read notifications while keeping unread"

OTHER_EMAIL="smoke-other+$(date +%s)-$$@example.com"
OTHER_PAYLOAD="$(jq -nc --arg email "${OTHER_EMAIL}" --arg password "${AUTH_PASSWORD}" '{email:$email,password:$password}')"

request POST "${AUTH_SERVICE_URL}/auth/register" "${OTHER_PAYLOAD}"
assert_2xx "Second user register"

OTHER_ACCESS_TOKEN="$(jq -r '.accessToken // empty' <<<"${LAST_BODY}")"
OTHER_REFRESH_TOKEN="$(jq -r '.refreshToken // empty' <<<"${LAST_BODY}")"
if [[ -z "${OTHER_ACCESS_TOKEN}" || -z "${OTHER_REFRESH_TOKEN}" ]]; then
  echo "Second user register response did not include both tokens." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's trip" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/export/${TRIP_EXPORT_ID}" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's export" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trip-discovery/sessions/${DISCOVERY_SESSION_ID}" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's discovery session" "404"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generate" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user generate first user's trip" "404"

request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary" "${OTHER_ACCESS_TOKEN}" "${EDIT_ITINERARY_PAYLOAD}"
assert_status "Second user edit first user's trip" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user list first user's itinerary versions" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions/${GENERATED_VERSION_ID}" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user get first user's itinerary version" "404"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/itinerary/versions/${GENERATED_VERSION_ID}/restore" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user restore first user's itinerary version" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user list first user's comments" "404"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/comments" "${OTHER_ACCESS_TOKEN}" '{"dayNumber":1,"itemIndex":0,"body":"Intruder"}'
assert_status "Second user create comment on first user's trip" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user list first user's checklist" "404"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/checklist/items/${MANUAL_CHECKLIST_ITEM_ID}/check" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user check first user's checklist item" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/health" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's trip health" "404"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/group-readiness" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's group readiness" "404"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget-confidence" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's budget confidence" "404"

request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user fetch first user's activity" "404"
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream" "${OTHER_ACCESS_TOKEN}"
assert_status "Second user open first user's activity stream" "404"

request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/presence/state" "${OTHER_ACCESS_TOKEN}" '{"state":"viewing"}'
assert_status "Second user update first user's presence" "404"

echo "Verifying the owner activity feed recorded the major actions..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=100" "${ACCESS_TOKEN}"
assert_2xx "Owner fetch full activity"
assert_activity_has "Owner activity feed" "trip_created"
assert_activity_has "Owner activity feed" "itinerary_generated"
assert_activity_has "Owner activity feed" "comment_created"
assert_activity_has "Owner activity feed" "accommodation_added"
assert_activity_has "Owner activity feed" "accommodation_removed"
assert_activity_has "Owner activity feed" "trip_traveler_added"
assert_activity_has "Owner activity feed" "trip_traveler_updated"
assert_activity_has "Owner activity feed" "trip_traveler_removed"
assert_activity_has "Owner activity feed" "cost_split_updated"
assert_activity_has "Owner activity feed" "accommodation_split_updated"
assert_activity_has "Owner activity feed" "collaborator_invited"
assert_activity_has "Owner activity feed" "collaborator_accepted"
assert_activity_has "Owner activity feed" "collaborator_removed"
assert_activity_has "Owner activity feed" "share_created"
assert_activity_has "Owner activity feed" "budget_optimization_proposed"
assert_activity_has "Owner activity feed" "budget_optimization_applied"
assert_activity_has "Owner activity feed" "budget_optimization_discarded"
assert_activity_has "Owner activity feed" "checklist_generated"
assert_activity_has "Owner activity feed" "checklist_regenerated"
assert_activity_has "Owner activity feed" "checklist_item_added"
assert_activity_has "Owner activity feed" "checklist_item_deleted"
assert_activity_has "Owner activity feed" "expense_created"
assert_activity_has "Owner activity feed" "settlement_marked_paid"

echo "Paging the activity feed one event at a time via the opaque cursor..."
ACTIVITY_CURSOR=""
ACTIVITY_SEEN_IDS=""
ACTIVITY_FOUND_TRIP_CREATED="false"
for _page in $(seq 1 200); do
  if [[ -n "${ACTIVITY_CURSOR}" ]]; then
    request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=1&cursor=${ACTIVITY_CURSOR}" "${ACCESS_TOKEN}"
  else
    request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity?limit=1" "${ACCESS_TOKEN}"
  fi
  assert_2xx "Activity cursor page"
  PAGE_COUNT="$(jq '.items | length' <<<"${LAST_BODY}")"
  if [[ "${PAGE_COUNT}" == "0" ]]; then
    break
  fi
  PAGE_ID="$(jq -r '.items[0].id' <<<"${LAST_BODY}")"
  PAGE_TYPE="$(jq -r '.items[0].eventType' <<<"${LAST_BODY}")"
  if [[ "${PAGE_TYPE}" == "trip_created" ]]; then
    ACTIVITY_FOUND_TRIP_CREATED="true"
  fi
  case " ${ACTIVITY_SEEN_IDS} " in
    *" ${PAGE_ID} "*)
      echo "Activity cursor pagination returned duplicate id ${PAGE_ID}" >&2
      exit 1
      ;;
  esac
  ACTIVITY_SEEN_IDS="${ACTIVITY_SEEN_IDS} ${PAGE_ID}"
  ACTIVITY_CURSOR="$(jq -r '.nextCursor // empty' <<<"${LAST_BODY}")"
  if [[ -z "${ACTIVITY_CURSOR}" ]]; then
    break
  fi
done
if [[ "${ACTIVITY_FOUND_TRIP_CREATED}" != "true" ]]; then
  echo "Activity cursor pagination never reached trip_created." >&2
  exit 1
fi
echo "Activity cursor pagination returned each event once and reached trip_created."

echo "Confirming unauthenticated activity access is rejected..."
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity"
assert_status "Unauthenticated activity access" "401"
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/activity/stream"
assert_status "Unauthenticated activity stream access" "401"

echo "Confirming owner can open the activity stream..."
assert_activity_stream_opens "Owner activity stream" "${ACCESS_TOKEN}"

echo "Confirming the public share has no activity endpoint..."
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/activity"
assert_status "Public share activity endpoint absent" "404"
request GET "${TRIP_SERVICE_URL}/public/trips/${SHARE_TOKEN}/activity/stream"
assert_status "Public share activity stream endpoint absent" "404"

echo "Logging out smoke test users..."
LOGOUT_PAYLOAD="$(jq -nc --arg refreshToken "${REFRESH_TOKEN}" '{refreshToken:$refreshToken}')"
request POST "${AUTH_SERVICE_URL}/auth/logout" "${LOGOUT_PAYLOAD}"
assert_2xx "Logout first user"

OTHER_LOGOUT_PAYLOAD="$(jq -nc --arg refreshToken "${OTHER_REFRESH_TOKEN}" '{refreshToken:$refreshToken}')"
request POST "${AUTH_SERVICE_URL}/auth/logout" "${OTHER_LOGOUT_PAYLOAD}"
assert_2xx "Logout second user"

echo "Smoke test passed: authenticated trip ${TRIP_ID} completed with ${DAYS_COUNT} itinerary day(s), budget conversion was exercised, revision conflicts were rejected, version restore worked, calendar sync was exercised when using the mock provider, the activity feed recorded major actions with access enforced, and owner isolation was enforced."
echo "Open ${WEB_APP_URL}/login to run the manual browser flow."
