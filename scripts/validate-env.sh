#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
TARGET=""

usage() {
  cat <<'USAGE'
Usage: scripts/validate-env.sh [local|test|staging|production|alpha] [--env-file PATH]
       scripts/validate-env.sh PATH_TO_ENV_FILE

Validates infra/.env by default and never prints secret values. Passing a
target checks that APP_ENV agrees with the intended deployment environment.
The alpha target intentionally requires APP_ENV=staging because service
configuration currently recognizes local/test/staging/production.
The legacy single environment-file argument remains supported.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    local|development|test|staging|production|alpha)
      [[ -z "${TARGET}" ]] || { echo "Only one environment target is allowed." >&2; exit 2; }
      TARGET="$1"
      shift
      ;;
    --env-file)
      [[ $# -ge 2 ]] || { echo "--env-file needs a path" >&2; exit 2; }
      ENV_FILE="$2"
      shift 2
      ;;
    --*)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      [[ $# -eq 1 && -z "${TARGET}" ]] || { echo "Unexpected argument: $1" >&2; usage >&2; exit 2; }
      ENV_FILE="$1"
      shift
      ;;
  esac
done

[[ -f "${ENV_FILE}" ]] || {
  echo "Environment file not found: ${ENV_FILE}" >&2
  echo "Create it with: cp infra/.env.example infra/.env" >&2
  exit 1
}

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      ""|\#*) continue ;;
      *=*) export "${line}" ;;
    esac
  done < "$1"
}

load_env_file "${ENV_FILE}"
APP_ENV="${APP_ENV:-}"
case "${APP_ENV}" in
  local|development|test) STRICT=false ;;
  staging|production) STRICT=true ;;
  *) echo "APP_ENV must be local, staging, or production." >&2; exit 1 ;;
esac
ALPHA_TARGET=false

if [[ "${TARGET}" == "alpha" ]]; then
  ALPHA_TARGET=true
  if [[ "${APP_ENV}" != "staging" ]]; then
    echo "Alpha validation requires APP_ENV=staging; got ${APP_ENV}." >&2
    exit 1
  fi
elif [[ -n "${TARGET}" && "${TARGET}" != "${APP_ENV}" ]]; then
  echo "APP_ENV is ${APP_ENV}, but ${TARGET} validation was requested." >&2
  exit 1
fi

declare -a ISSUES=()

issue() { ISSUES+=("$1"); }

value_of() {
  local name="$1"
  printf '%s' "${!name:-}"
}

require() {
  local name="$1"
  [[ -n "$(value_of "${name}")" ]] || issue "${name} is required"
}

is_placeholder() {
  local value
  value="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  [[ "${value}" == *change_me* || "${value}" == *change-me* || "${value}" == *replace_with* || "${value}" == *placeholder* || "${value}" == *example.com* || "${value}" == set_in_secret_manager_or_env || "${value}" == dev-* || "${value}" == guest || "${value}" == postgres || "${value}" == admin || "${value}" == password || "${value}" == secret ]]
}

require_secret() {
  local name="$1" min_len="$2" value
  value="$(value_of "${name}")"
  if [[ -z "${value}" ]]; then
    issue "${name} is required in ${APP_ENV}"
  elif [[ ${#value} -lt ${min_len} ]]; then
    issue "${name} must be at least ${min_len} characters in ${APP_ENV}"
  elif is_placeholder "${value}"; then
    issue "${name} must not use a default or placeholder in ${APP_ENV}"
  fi
}

require_http_url() {
  local name="$1" value
  value="$(value_of "${name}")"
  if [[ ! "${value}" =~ ^https?:// ]]; then
    issue "${name} must be an http/https URL"
    return
  fi
  if [[ "${APP_ENV}" == "production" ]]; then
    [[ "${value}" =~ ^https:// ]] || issue "${name} must use https in production"
    [[ "${value}" != *localhost* && "${value}" != *127.0.0.1* ]] || issue "${name} must not use localhost in production"
  fi
}

require_provider_key() {
  local mode_name="$1" expected_mode="$2" key_name="$3"
  if [[ "$(value_of "${mode_name}")" == "${expected_mode}" ]]; then
    require "${key_name}"
  fi
}

is_openai_selected() {
  [[ "${AI_MODEL_PROVIDER:-mock}" == "openai" \
    || "${AI_ITINERARY_GENERATOR_MODE:-mock}" == "openai" \
    || "${COPILOT_MODE:-mock}" == "openai" \
    || "${TRIP_RECAP_AI_MODE:-mock}" == "openai" ]]
}

validate_boolean() {
  local name="$1" value
  value="$(value_of "${name}")"
  [[ -z "${value}" || "${value}" == "true" || "${value}" == "false" ]] || issue "${name} must be true or false"
}

require POSTGRES_USER
require POSTGRES_DB
require RABBITMQ_URL
[[ "${RABBITMQ_URL}" =~ ^amqps?:// ]] || issue "RABBITMQ_URL must be an amqp/amqps URL"
require PUBLIC_WEB_BASE_URL
require_http_url PUBLIC_WEB_BASE_URL

if [[ "${STRICT}" == true ]]; then
  require_secret POSTGRES_PASSWORD 16
  require_secret RABBITMQ_PASSWORD 16
  require_secret JWT_ACCESS_SECRET 32
  require_secret JWT_REFRESH_SECRET 32
  require_secret INTERNAL_SERVICE_TOKEN 32
  require_secret NOTIFICATION_SERVICE_TOKEN 32
  require_secret OPS_INTERNAL_SERVICE_TOKEN 32
  require_secret PUBLIC_SHARE_ACCESS_SECRET 32
  require_secret CALENDAR_TOKEN_ENCRYPTION_KEY 32
  require_secret GRAFANA_ADMIN_PASSWORD 16
  [[ -n "${CORS_ALLOWED_ORIGINS:-}" && "${CORS_ALLOWED_ORIGINS}" != "*" ]] || issue "CORS_ALLOWED_ORIGINS must be set and must not be wildcard in ${APP_ENV}"
  [[ "${CORS_ALLOWED_ORIGINS:-}" != *localhost* && "${CORS_ALLOWED_ORIGINS:-}" != *127.0.0.1* ]] || issue "CORS_ALLOWED_ORIGINS must not use localhost in ${APP_ENV}"
  [[ "${LOG_LEVEL:-INFO}" != "DEBUG" ]] || issue "LOG_LEVEL must not be DEBUG in ${APP_ENV}"
  [[ "${FILE_SCANNING_FAIL_OPEN:-false}" != "true" ]] || issue "FILE_SCANNING_FAIL_OPEN must be false in ${APP_ENV}"
fi

require_provider_key PLACE_PROVIDER foursquare FOURSQUARE_API_KEY
require_provider_key ROUTE_PROVIDER ors ORS_API_KEY
require_provider_key WEATHER_PROVIDER openweathermap OPENWEATHER_API_KEY
if [[ "${EXCHANGE_RATE_PROVIDER:-mock}" != "mock" ]]; then require EXCHANGE_RATE_API_KEY; fi
if [[ "${PRICE_PROVIDER:-mock}" != "mock" ]]; then require PRICE_API_KEY; fi
if [[ "${CALENDAR_PROVIDER:-mock}" == "google" && "${GOOGLE_CALENDAR_ENABLED:-false}" == "true" ]]; then
  require GOOGLE_OAUTH_CLIENT_ID
  require GOOGLE_OAUTH_CLIENT_SECRET
  require_http_url GOOGLE_OAUTH_REDIRECT_URL
fi
if [[ "${EMAIL_PROVIDER:-mock}" == "smtp" ]]; then
  require SMTP_HOST
  require SMTP_FROM_EMAIL
  [[ "${STRICT}" == false ]] || require SMTP_PASSWORD
fi
if [[ "${WEB_PUSH_ENABLED:-false}" == "true" ]]; then
  require WEB_PUSH_VAPID_PUBLIC_KEY
  require WEB_PUSH_VAPID_PRIVATE_KEY
  require WEB_PUSH_SUBJECT
fi
case "${AI_MODEL_PROVIDER:-mock}" in
  mock|ollama|openai) ;;
  *) issue "AI_MODEL_PROVIDER must be mock, ollama, or openai" ;;
esac
case "${AI_MODEL_PROVIDER_FALLBACK:-none}" in
  none|mock|ollama) ;;
  *) issue "AI_MODEL_PROVIDER_FALLBACK must be none, mock, or ollama" ;;
esac
case "${AI_ITINERARY_GENERATOR_MODE:-mock}" in
  mock|ollama|openai) ;;
  *) issue "AI_ITINERARY_GENERATOR_MODE must be mock, ollama, or openai" ;;
esac
if [[ "${AI_ITINERARY_GENERATOR_MODE:-mock}" == "ollama" ]]; then require_http_url OLLAMA_BASE_URL; fi
if is_openai_selected; then
  [[ "${OPENAI_ENABLED:-false}" == "true" ]] || issue "OPENAI_ENABLED must be true when OpenAI is selected"
  require OPENAI_API_KEY
  [[ -n "${OPENAI_MODEL_DEFAULT:-}${OPENAI_MODEL_ITINERARY:-}" ]] || issue "OPENAI_MODEL_DEFAULT or OPENAI_MODEL_ITINERARY is required when OpenAI is selected"
  [[ "${OPENAI_STORE_RESPONSES+x}" == x ]] || issue "OPENAI_STORE_RESPONSES must be explicit when OpenAI is selected"
  if [[ -n "${OPENAI_BASE_URL:-}" ]]; then require_http_url OPENAI_BASE_URL; fi
  if [[ "${STRICT}" == true ]]; then
    require_secret OPENAI_API_KEY 20
    [[ "${OPENAI_STORE_RESPONSES:-false}" == "false" ]] || issue "OPENAI_STORE_RESPONSES should remain false unless explicitly reviewed"
  fi
fi
if [[ "${RAG_ENABLED:-false}" == "true" ]]; then require OLLAMA_EMBEDDING_MODEL; fi

for flag_var in \
  AI_MODEL_PROVIDER_FALLBACK_ENABLED OPENAI_ENABLED OPENAI_STORE_RESPONSES \
  OPENAI_USAGE_TRACKING_ENABLED OPENAI_COST_TRACKING_ENABLED OPENAI_REPAIR_ENABLED \
  OPENAI_BATCH_ENABLED \
  FEATURE_FLAGS_ENABLED FEATURE_FLAGS_FAIL_CLOSED \
  FEATURE_AI_GENERATION_ENABLED FEATURE_AI_REPAIR_ENABLED FEATURE_COPILOT_ENABLED \
  FEATURE_ROUTE_ALTERNATIVES_ENABLED FEATURE_TEMPLATE_ADAPTATION_ENABLED \
  FEATURE_PUBLIC_SHARING_ENABLED FEATURE_DATA_EXPORTS_ENABLED FEATURE_REAL_PROVIDERS_ENABLED \
  FEATURE_CALENDAR_SYNC_ENABLED FEATURE_AVAILABILITY_SEARCH_ENABLED FEATURE_TRANSPORT_SEARCH_ENABLED \
  FEATURE_RECEIPT_OCR_ENABLED FEATURE_WORKSPACE_APPROVALS_ENABLED FEATURE_POLICY_REPAIR_ENABLED \
  FEATURE_WEB_PUSH_ENABLED FEATURE_EMAIL_NOTIFICATIONS_ENABLED FEATURE_NOTIFICATION_DIGESTS_ENABLED \
  FEATURE_OFFLINE_MODE_ENABLED FEATURE_OPS_DASHBOARD_ENABLED \
  FEATURE_AI_FINE_TUNING_EXPERIMENTS_ENABLED FEATURE_AI_ADAPTER_INFERENCE_ENABLED \
  FEATURE_AI_ADAPTER_STAGING_ENABLED FEATURE_AI_MODEL_SERVING_ENABLED \
  FEATURE_AI_SHADOW_EVALUATION_ENABLED FEATURE_AI_CANDIDATE_INTERNAL_ROLLOUT_ENABLED \
  FEATURE_AI_CANDIDATE_ALLOWLIST_ROLLOUT_ENABLED FEATURE_AI_CANDIDATE_PERCENTAGE_ROLLOUT_ENABLED \
  FEATURE_AI_CANDIDATE_USER_OPT_IN_ENABLED FEATURE_AI_AUTOMATIC_GUARDRAIL_PAUSE_ENABLED; do
  validate_boolean "${flag_var}"
done

if [[ -n "${FEATURE_FLAGS_CACHE_TTL_SECONDS:-}" ]] && ! [[ "${FEATURE_FLAGS_CACHE_TTL_SECONDS}" =~ ^[1-9][0-9]*$ ]] ; then
  issue "FEATURE_FLAGS_CACHE_TTL_SECONDS must be a positive integer"
fi

if [[ "${STRICT}" == true && "${FEATURE_CALENDAR_SYNC_ENABLED:-false}" == "true" ]]; then
  [[ "${GOOGLE_CALENDAR_ENABLED:-false}" == "true" ]] || issue "GOOGLE_CALENDAR_ENABLED must be true when FEATURE_CALENDAR_SYNC_ENABLED=true"
  require GOOGLE_OAUTH_CLIENT_ID
  require GOOGLE_OAUTH_CLIENT_SECRET
  require_http_url GOOGLE_OAUTH_REDIRECT_URL
fi
if [[ "${STRICT}" == true && "${FEATURE_WEB_PUSH_ENABLED:-false}" == "true" ]]; then
  require WEB_PUSH_VAPID_PUBLIC_KEY
  require WEB_PUSH_VAPID_PRIVATE_KEY
  require WEB_PUSH_SUBJECT
fi
if [[ "${STRICT}" == true && "${FEATURE_EMAIL_NOTIFICATIONS_ENABLED:-false}" == "true" && "${EMAIL_PROVIDER:-mock}" == "smtp" ]]; then
  require SMTP_HOST
  require SMTP_FROM_EMAIL
  [[ "${STRICT}" == false ]] || require SMTP_PASSWORD
fi
if [[ "${STRICT}" == true && "${FEATURE_REAL_PROVIDERS_ENABLED:-false}" == "true" ]]; then
  if [[ -z "${FOURSQUARE_API_KEY:-}${ORS_API_KEY:-}${OPENWEATHER_API_KEY:-}${PRICE_API_KEY:-}${EXCHANGE_RATE_API_KEY:-}" ]]; then
    issue "at least one real provider key is required when FEATURE_REAL_PROVIDERS_ENABLED=true"
  fi
fi
if [[ "${STRICT}" == true && "${FEATURE_OPS_DASHBOARD_ENABLED:-false}" == "true" ]]; then
  require OPS_ADMIN_EMAILS
fi

positive_number() {
  local name="$1" value
  value="$(value_of "${name}")"
  [[ -n "${value}" ]] || { issue "${name} is required in alpha"; return; }
  [[ "${value}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { issue "${name} must be a positive number"; return; }
  awk -v n="${value}" 'BEGIN { exit !(n > 0) }' || issue "${name} must be greater than zero"
}

positive_integer() {
  local name="$1" value
  value="$(value_of "${name}")"
  [[ -n "${value}" ]] || { issue "${name} is required in alpha"; return; }
  [[ "${value}" =~ ^[1-9][0-9]*$ ]] || issue "${name} must be a positive integer"
}

if [[ "${ALPHA_TARGET}" == true ]]; then
  [[ "${ALPHA_RELEASE_PROFILE:-}" == "closed-alpha-v1" ]] || issue "ALPHA_RELEASE_PROFILE must be closed-alpha-v1"
  [[ "${NEXT_PUBLIC_APP_ENV:-}" == "staging" ]] || issue "NEXT_PUBLIC_APP_ENV must be staging for alpha"
  [[ "${NEXT_PUBLIC_ALPHA_RELEASE_LABEL:-}" == "Alpha" ]] || issue "NEXT_PUBLIC_ALPHA_RELEASE_LABEL must be Alpha"
  [[ "${AI_MODEL_PROVIDER:-}" == "openai" ]] || issue "AI_MODEL_PROVIDER must be openai in the alpha profile"
  [[ "${AI_ITINERARY_GENERATOR_MODE:-}" == "openai" ]] || issue "AI_ITINERARY_GENERATOR_MODE must be openai in the alpha profile"
  [[ "${AI_MODEL_PROVIDER_FALLBACK:-}" == "mock" || "${AI_MODEL_PROVIDER_FALLBACK:-}" == "ollama" ]] || issue "AI_MODEL_PROVIDER_FALLBACK must be mock or ollama in alpha"
  [[ "${AI_MODEL_PROVIDER_FALLBACK_ENABLED:-false}" == "true" ]] || issue "AI_MODEL_PROVIDER_FALLBACK_ENABLED must be true in alpha"
  [[ "${OPENAI_ENABLED:-false}" == "true" ]] || issue "OPENAI_ENABLED must be true in alpha"
  [[ "${OPENAI_STORE_RESPONSES:-}" == "false" ]] || issue "OPENAI_STORE_RESPONSES must be false in alpha"
  [[ "${OPENAI_USAGE_TRACKING_ENABLED:-false}" == "true" ]] || issue "OPENAI_USAGE_TRACKING_ENABLED must be true in alpha"
  [[ "${OPENAI_COST_TRACKING_ENABLED:-false}" == "true" ]] || issue "OPENAI_COST_TRACKING_ENABLED must be true in alpha"
  positive_number OPENAI_DAILY_SPEND_LIMIT_UAH
  positive_number OPENAI_MONTHLY_SPEND_LIMIT_UAH
  positive_integer OPENAI_PER_USER_DAILY_GENERATION_LIMIT
  positive_integer OPENAI_PER_TRIP_DAILY_GENERATION_LIMIT
  positive_integer OPENAI_MAX_CONCURRENT_REQUESTS
  positive_integer OPENAI_MAX_INPUT_TOKENS
  positive_integer OPENAI_MAX_CONTEXT_BYTES
  positive_integer OPENAI_MAX_GROUNDING_PLACES
  positive_integer OPENAI_MAX_GROUNDING_DOCUMENTS
  positive_integer OPENAI_MAX_RETRIES
  positive_integer OPENAI_TIMEOUT_SECONDS
  positive_integer OPENAI_REPAIR_TIMEOUT_SECONDS
  [[ "${OPENAI_REPAIR_ENABLED:-false}" == "true" ]] || issue "OPENAI_REPAIR_ENABLED must be true in alpha"
  [[ "${OPENAI_MAX_REPAIR_ATTEMPTS:-}" == "1" ]] || issue "OPENAI_MAX_REPAIR_ATTEMPTS must be 1 in alpha"
  [[ "${AI_PROMPT_LOGGING_ENABLED:-false}" == "false" ]] || issue "AI_PROMPT_LOGGING_ENABLED must be false in alpha"
  [[ "${AI_OBSERVABILITY_ENABLED:-false}" == "true" ]] || issue "AI_OBSERVABILITY_ENABLED must be true in alpha"
  [[ "${AI_OBSERVABILITY_STORE_REDACTED_PROMPTS:-false}" == "false" ]] || issue "AI_OBSERVABILITY_STORE_REDACTED_PROMPTS must be false in alpha"
  [[ "${AI_OBSERVABILITY_STORE_REDACTED_RESPONSES:-false}" == "false" ]] || issue "AI_OBSERVABILITY_STORE_REDACTED_RESPONSES must be false in alpha"
  [[ "${AI_OBSERVABILITY_REDACTION_ENABLED:-false}" == "true" ]] || issue "AI_OBSERVABILITY_REDACTION_ENABLED must be true in alpha"
  [[ "${OPS_DASHBOARD_ENABLED:-false}" == "true" ]] || issue "OPS_DASHBOARD_ENABLED must be true in alpha"
  [[ "${FEATURE_OPS_DASHBOARD_ENABLED:-false}" == "true" ]] || issue "FEATURE_OPS_DASHBOARD_ENABLED must be true in alpha"
  [[ -n "${OPS_ADMIN_EMAILS:-}" && "${OPS_ADMIN_EMAILS}" != *"*"* ]] || issue "OPS_ADMIN_EMAILS must be an explicit allowlist in alpha"
  [[ -n "${BACKUP_DIR:-}" ]] || issue "BACKUP_DIR is required in alpha"
  [[ -n "${FEATURE_FLAG_PROFILE_PATH:-}" ]] || issue "FEATURE_FLAG_PROFILE_PATH is required in alpha"
  [[ -f "${PROJECT_ROOT}/${FEATURE_FLAG_PROFILE_PATH:-}" || -f "${FEATURE_FLAG_PROFILE_PATH:-}" ]] || issue "FEATURE_FLAG_PROFILE_PATH must point to an existing profile"
  [[ "${FEATURE_FLAGS_ENABLED:-false}" == "true" ]] || issue "FEATURE_FLAGS_ENABLED must be true in alpha"
  [[ "${FEATURE_FLAGS_FAIL_CLOSED:-false}" == "true" ]] || issue "FEATURE_FLAGS_FAIL_CLOSED must be true in alpha"
  [[ "${DEBUG_PROMPT_ENDPOINT_ENABLED:-false}" != "true" ]] || issue "DEBUG_PROMPT_ENDPOINT_ENABLED must not be true in alpha"
  for blocked in \
    FEATURE_REAL_PROVIDERS_ENABLED FEATURE_CALENDAR_SYNC_ENABLED FEATURE_AVAILABILITY_SEARCH_ENABLED \
    FEATURE_TRANSPORT_SEARCH_ENABLED FEATURE_RECEIPT_OCR_ENABLED FEATURE_WORKSPACE_APPROVALS_ENABLED \
    FEATURE_POLICY_REPAIR_ENABLED FEATURE_WEB_PUSH_ENABLED FEATURE_OFFLINE_MODE_ENABLED \
    FEATURE_ROUTE_ALTERNATIVES_ENABLED FEATURE_TEMPLATE_ADAPTATION_ENABLED \
    FEATURE_AI_FINE_TUNING_EXPERIMENTS_ENABLED FEATURE_AI_ADAPTER_INFERENCE_ENABLED \
    FEATURE_AI_ADAPTER_STAGING_ENABLED FEATURE_AI_SHADOW_EVALUATION_ENABLED \
    FEATURE_AI_CANDIDATE_INTERNAL_ROLLOUT_ENABLED FEATURE_AI_CANDIDATE_ALLOWLIST_ROLLOUT_ENABLED \
    FEATURE_AI_CANDIDATE_PERCENTAGE_ROLLOUT_ENABLED FEATURE_AI_CANDIDATE_USER_OPT_IN_ENABLED \
    AI_SHADOW_ENABLED AI_CANDIDATE_INTERNAL_ROLLOUT_ENABLED AI_CANDIDATE_ALLOWLIST_ROLLOUT_ENABLED \
    AI_CANDIDATE_PERCENTAGE_ROLLOUT_ENABLED AI_CANDIDATE_USER_OPT_IN_ENABLED; do
    [[ "$(value_of "${blocked}")" != "true" ]] || issue "${blocked} must be false in alpha"
  done
  "${PROJECT_ROOT}/scripts/release/validate-alpha-scope.sh" --profile "${PROJECT_ROOT}/config/feature-flags/alpha.json" --env-file "${ENV_FILE}" >/dev/null || issue "alpha feature flag profile validation failed"
fi

if (( ${#ISSUES[@]} > 0 )); then
  echo "Environment validation failed for APP_ENV=${APP_ENV}:" >&2
  for item in "${ISSUES[@]}"; do echo "- ${item}" >&2; done
  exit 1
fi

echo "Environment is valid for APP_ENV=${APP_ENV}."
