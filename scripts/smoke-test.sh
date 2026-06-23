#!/usr/bin/env bash
set -euo pipefail

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://localhost:8082}"
USER_SERVICE_URL="${USER_SERVICE_URL:-http://localhost:8083}"
AI_PLANNING_SERVICE_URL="${AI_PLANNING_SERVICE_URL:-http://localhost:8000}"
WEB_APP_URL="${WEB_APP_URL:-http://localhost:3000}"

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
if [[ -z "${ACCESS_TOKEN}" || -z "${REFRESH_TOKEN}" ]]; then
  echo "Auth login response did not include both tokens." >&2
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
  "preferredTransport": ["walking", "public_transport"]
}'
request_with_bearer PATCH "${USER_SERVICE_URL}/users/me/preferences" "${ACCESS_TOKEN}" "${PATCH_PREFERENCES_PAYLOAD}"
assert_2xx "Patch preferences"

PATCHED_STYLE_COUNT="$(jq '.travelStyles | length' <<<"${LAST_BODY}")"
PATCHED_WALKING="$(jq -r '.maxWalkingKmPerDay // empty' <<<"${LAST_BODY}")"
PATCHED_AVOID="$(jq -r '.avoid[0] // empty' <<<"${LAST_BODY}")"
if [[ "${PATCHED_STYLE_COUNT}" -ne 3 || "${PATCHED_WALKING}" != "8" || "${PATCHED_AVOID}" != "nightclubs" ]]; then
  echo "Patched preferences did not include expected values." >&2
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
echo "Created trip ${TRIP_ID}."

echo "Generating itinerary with Authorization header..."
request_with_bearer POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generate" "${ACCESS_TOKEN}"
assert_2xx "Generate itinerary"

echo "Fetching completed trip with Authorization header..."
request_with_bearer GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}" "${ACCESS_TOKEN}"
assert_2xx "Fetch trip"

STATUS="$(jq -r '.status // empty' <<<"${LAST_BODY}")"
DAYS_COUNT="$(jq '.itinerary.days | length' <<<"${LAST_BODY}")"

if [[ "${STATUS}" != "COMPLETED" ]]; then
  echo "Expected trip status COMPLETED, got ${STATUS}." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

if [[ "${DAYS_COUNT}" -le 0 ]]; then
  echo "Expected itinerary.days to contain at least one day." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi

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

echo "Logging out smoke test users..."
LOGOUT_PAYLOAD="$(jq -nc --arg refreshToken "${REFRESH_TOKEN}" '{refreshToken:$refreshToken}')"
request POST "${AUTH_SERVICE_URL}/auth/logout" "${LOGOUT_PAYLOAD}"
assert_2xx "Logout first user"

OTHER_LOGOUT_PAYLOAD="$(jq -nc --arg refreshToken "${OTHER_REFRESH_TOKEN}" '{refreshToken:$refreshToken}')"
request POST "${AUTH_SERVICE_URL}/auth/logout" "${OTHER_LOGOUT_PAYLOAD}"
assert_2xx "Logout second user"

echo "Smoke test passed: authenticated trip ${TRIP_ID} completed with ${DAYS_COUNT} itinerary day(s), and owner isolation was enforced."
echo "Open ${WEB_APP_URL}/login to run the manual browser flow."
