#!/usr/bin/env bash
set -euo pipefail

# Exercises the private recap lifecycle against a disposable owner/editor trip.
# It deliberately uses generateEarly so a seeded trip need not have ended.
TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
ACCESS_TOKEN="${RECAP_SMOKE_ACCESS_TOKEN:-}"
TRIP_ID="${RECAP_SMOKE_TRIP_ID:-}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

for command_name in curl jq; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    echo "${command_name} is required." >&2
    exit 1
  }
done
if [[ -z "${ACCESS_TOKEN}" || -z "${TRIP_ID}" ]]; then
  echo "Set RECAP_SMOKE_ACCESS_TOKEN and RECAP_SMOKE_TRIP_ID for an owner/editor disposable trip." >&2
  exit 1
fi

request() {
  local method="$1" path="$2" body="$3" output="$4" status
  if [[ -n "${body}" ]]; then
    status="$(curl -sS -o "${output}" -w '%{http_code}' -X "${method}" -H "Authorization: Bearer ${ACCESS_TOKEN}" -H 'Content-Type: application/json' --data "${body}" "${TRIP_SERVICE_URL}${path}")"
  else
    status="$(curl -sS -o "${output}" -w '%{http_code}' -X "${method}" -H "Authorization: Bearer ${ACCESS_TOKEN}" -H 'Content-Type: application/json' "${TRIP_SERVICE_URL}${path}")"
  fi
  if [[ "${status}" -lt 200 || "${status}" -ge 300 ]]; then
    echo "${method} ${path} failed with HTTP ${status}: $(cat "${output}")" >&2
    exit 1
  fi
}

echo "Checking recap status…"
request GET "/trips/${TRIP_ID}/recap/status" "" "${TMP_DIR}/status.json"

echo "Generating recap…"
request POST "/trips/${TRIP_ID}/recap/generate" '{"generateEarly":true,"language":"en"}' "${TMP_DIR}/generated.json"
jq -e '.recap.schemaVersion == "trip_recap_v1"' "${TMP_DIR}/generated.json" >/dev/null

echo "Reading and editing recap…"
request GET "/trips/${TRIP_ID}/recap" "" "${TMP_DIR}/recap.json"
PATCH_BODY="$(jq -c '.recap.recap.userEditableNotes = "Recap smoke review." | {recap: .recap.recap}' "${TMP_DIR}/recap.json")"
request PATCH "/trips/${TRIP_ID}/recap" "${PATCH_BODY}" "${TMP_DIR}/edited.json"

echo "Submitting explicit learning feedback…"
FEEDBACK_BODY='{"feedbackType":"prefer_next_time","label":"recap smoke preference","value":"balanced pace","approvedForPersonalization":false,"metadata":{"source":"smoke"}}'
request POST "/trips/${TRIP_ID}/recap/feedback" "${FEEDBACK_BODY}" "${TMP_DIR}/feedback.json"
FEEDBACK_ID="$(jq -r '.id' "${TMP_DIR}/feedback.json")"
request POST "/trips/${TRIP_ID}/recap/apply-learning" "$(jq -nc --arg id "${FEEDBACK_ID}" '{feedbackIds:[$id]}')" "${TMP_DIR}/learning.json"

echo "Creating a safe reusable template and finalizing…"
TEMPLATE_BODY="$(jq -nc --arg title "Recap smoke ${TRIP_ID}" '{title:$title,description:"Created by recap smoke",visibility:"private",tags:["smoke","recap"],useRecapLessons:true}')"
request POST "/trips/${TRIP_ID}/recap/create-template" "${TEMPLATE_BODY}" "${TMP_DIR}/template.json"
request POST "/trips/${TRIP_ID}/recap/finalize" '{}' "${TMP_DIR}/finalized.json"

if grep -Eqi '"(rawText|ocr|storageKey|commentBody|calendar|accessToken|refreshToken)"[[:space:]]*:' "${TMP_DIR}"/*.json; then
  echo "Recap smoke found a forbidden private field in a response." >&2
  exit 1
fi

if [[ "${RECAP_SMOKE_ARCHIVE:-false}" == "true" ]]; then
  echo "Archiving recap (explicitly enabled)…"
  request DELETE "/trips/${TRIP_ID}/recap" "" "${TMP_DIR}/archived.json"
fi
echo "Recap smoke passed for trip ${TRIP_ID}."
