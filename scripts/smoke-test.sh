#!/usr/bin/env bash
set -euo pipefail

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
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

assert_2xx() {
  local label="$1"
  if [[ ! "${LAST_STATUS}" =~ ^2 ]]; then
    echo "${label} failed with HTTP ${LAST_STATUS}" >&2
    echo "${LAST_BODY}" >&2
    exit 1
  fi
}

echo "Checking Trip Service health..."
request GET "${TRIP_SERVICE_URL}/health"
assert_2xx "Trip Service health check"

echo "Checking AI Planning Service health..."
request GET "${AI_PLANNING_SERVICE_URL}/health"
assert_2xx "AI Planning Service health check"

echo "Checking Web App..."
request GET "${WEB_APP_URL}"
assert_2xx "Web App check"

echo "Checking AI Planning Service destination context endpoint..."
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

echo "Creating a trip through Trip Service..."
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
request POST "${TRIP_SERVICE_URL}/trips" "${CREATE_TRIP_PAYLOAD}"
assert_2xx "Create trip"

TRIP_ID="$(jq -r '.id // empty' <<<"${LAST_BODY}")"
if [[ -z "${TRIP_ID}" ]]; then
  echo "Create trip response did not include an id." >&2
  echo "${LAST_BODY}" >&2
  exit 1
fi
echo "Created trip ${TRIP_ID}."

echo "Generating itinerary..."
request POST "${TRIP_SERVICE_URL}/trips/${TRIP_ID}/generate"
assert_2xx "Generate itinerary"

echo "Fetching completed trip..."
request GET "${TRIP_SERVICE_URL}/trips/${TRIP_ID}"
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

echo "Smoke test passed: trip ${TRIP_ID} completed with ${DAYS_COUNT} itinerary day(s)."
