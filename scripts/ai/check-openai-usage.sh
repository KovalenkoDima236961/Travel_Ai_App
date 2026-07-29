#!/usr/bin/env bash
set -euo pipefail

AI_SERVICE_URL="${AI_SERVICE_URL:-http://127.0.0.1:8000}"

metrics="$(curl -fsS "${AI_SERVICE_URL%/}/metrics")"
printf '%s\n' "${metrics}" | grep -E '^(ai_provider_requests_total|ai_provider_input_tokens_total|ai_provider_output_tokens_total|ai_provider_errors_total|ai_provider_fallbacks_total)' || {
  echo "No AI provider usage metrics found at ${AI_SERVICE_URL%/}/metrics." >&2
  exit 1
}

