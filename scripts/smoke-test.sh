#!/usr/bin/env bash
set -euo pipefail

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://localhost:8082}"
USER_SERVICE_URL="${SMOKE_USER_SERVICE_URL:-${USER_SERVICE_URL:-http://localhost:8083}}"
AI_PLANNING_SERVICE_URL="${SMOKE_AI_PLANNING_SERVICE_URL:-${AI_PLANNING_SERVICE_URL:-http://localhost:8000}}"
EXTERNAL_INTEGRATIONS_SERVICE_URL="${SMOKE_EXTERNAL_INTEGRATIONS_SERVICE_URL:-${NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL:-http://localhost:8084}}"
NOTIFICATION_SERVICE_URL="${SMOKE_NOTIFICATION_SERVICE_URL:-${NOTIFICATION_SERVICE_URL:-http://localhost:8086}}"
WEB_APP_URL="${WEB_APP_URL:-http://localhost:3000}"
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

echo "Checking User Service health..."
request GET "${USER_SERVICE_URL}/health"
assert_2xx "User Service health check"

echo "Checking AI Planning Service health..."
request GET "${AI_PLANNING_SERVICE_URL}/health"
assert_2xx "AI Planning Service health check"

echo "Checking External Integrations Service health..."
request GET "${EXTERNAL_INTEGRATIONS_SERVICE_URL}/health"
assert_2xx "External Integrations Service health check"

echo "Checking Notification Service health..."
request GET "${NOTIFICATION_SERVICE_URL}/health"
assert_2xx "Notification Service health check"

echo "Checking Notification Service readiness..."
request GET "${NOTIFICATION_SERVICE_URL}/ready"
assert_2xx "Notification Service readiness check"

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
AUTH_EMAIL="smoke+$(date +%s)-$$@example.com"
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
  "preferredTransport": ["walking", "public_transport"],
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

echo "Checking default notification preferences..."
request_with_bearer GET "${NOTIFICATION_SERVICE_URL}/notifications/preferences" "${ACCESS_TOKEN}"
assert_2xx "Get default notification preferences"
NOTIFICATION_PREF_COUNT="$(jq '.items | length' <<<"${LAST_BODY}")"
DEFAULT_IN_APP_COMMENTS="$(jq -r '.items[] | select(.channel == "in_app" and .category == "comments") | .enabled' <<<"${LAST_BODY}")"
DEFAULT_EMAIL_TRIP_UPDATES="$(jq -r '.items[] | select(.channel == "email" and .category == "trip_updates") | .enabled' <<<"${LAST_BODY}")"
if [[ "${NOTIFICATION_PREF_COUNT}" -ne 8 || "${DEFAULT_IN_APP_COMMENTS}" != "true" || "${DEFAULT_EMAIL_TRIP_UPDATES}" != "false" ]]; then
  echo "Default notification preferences did not match expected values." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

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
request_with_bearer PUT "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/budget" "${COLLAB_ACCESS_TOKEN}" '{"budget":{"amount":1000,"currency":"EUR"}}'
assert_status "Viewer update budget" "403"

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

PUBLIC_DESTINATION="$(jq -r '.destination // empty' <<<"${PUBLIC_TRIP_BODY}")"
PUBLIC_ITINERARY_DAYS="$(jq '.itinerary.days | length' <<<"${PUBLIC_TRIP_BODY}")"
if [[ "${PUBLIC_DESTINATION}" != "Rome" || "${PUBLIC_ITINERARY_DAYS}" -le 0 ]]; then
  echo "Public shared trip did not include expected destination and itinerary." >&2
  echo "${PUBLIC_TRIP_BODY}" >&2
  exit 1
fi
if jq -e 'has("userId") or has("email") or has("versionHistory") or has("comments") or has("accommodation") or has("budget") or has("budgetAmount") or has("budgetCurrency")' >/dev/null <<<"${PUBLIC_TRIP_BODY}"; then
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

echo "Verifying another user cannot access the first user's trip..."
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
assert_activity_has "Owner activity feed" "collaborator_invited"
assert_activity_has "Owner activity feed" "collaborator_accepted"
assert_activity_has "Owner activity feed" "collaborator_removed"
assert_activity_has "Owner activity feed" "share_created"
assert_activity_has "Owner activity feed" "budget_optimization_proposed"
assert_activity_has "Owner activity feed" "budget_optimization_applied"
assert_activity_has "Owner activity feed" "budget_optimization_discarded"

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
