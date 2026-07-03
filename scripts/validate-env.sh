#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/validate-env.sh <env-file>

Performs deployment-focused validation without printing secret values.
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

ENV_FILE="${1:-}"
if [ -z "$ENV_FILE" ] || [ ! -f "$ENV_FILE" ]; then
  usage >&2
  exit 1
fi

load_env_file() {
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ""|\#*) continue ;;
      *=*) export "$line" ;;
    esac
  done < "$1"
}

load_env_file "$ENV_FILE"

APP_ENV="${APP_ENV:-}"
case "$APP_ENV" in
  local|development|test) STRICT=false ;;
  staging|production) STRICT=true ;;
  *) echo "APP_ENV must be local, staging, or production" >&2; exit 1 ;;
esac

fail() {
  echo "env validation failed: $1" >&2
  exit 1
}

require() {
  name="$1"
  eval "value=\${$name:-}"
  [ -n "$value" ] || fail "$name is required"
}

require_secret() {
  name="$1"
  min_len="$2"
  eval "value=\${$name:-}"
  [ -n "$value" ] || fail "$name is required in $APP_ENV"
  [ "${#value}" -ge "$min_len" ] || fail "$name must be at least $min_len characters in $APP_ENV"
  case "$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]')" in
    secret|password|dev|changeme|change-me|guest|admin|postgres|change-me-in-development|dev-internal-service-token)
      fail "$name must not use a development default in $APP_ENV"
      ;;
  esac
}

require_url() {
  name="$1"
  eval "value=\${$name:-}"
  [ -n "$value" ] || fail "$name is required"
  case "$value" in
    http://*|https://*) ;;
    *) fail "$name must be an http/https URL" ;;
  esac
  if [ "$APP_ENV" = "production" ]; then
    case "$value" in
      https://*) ;;
      *) fail "$name must use https in production" ;;
    esac
    case "$value" in
      *localhost*|*127.0.0.1*) fail "$name must not use localhost in production" ;;
    esac
  fi
}

if [ "$STRICT" = true ]; then
  require_secret POSTGRES_PASSWORD 16
  require_secret RABBITMQ_PASSWORD 16
  require_secret JWT_ACCESS_SECRET 32
  require_secret JWT_REFRESH_SECRET 32
  require_secret INTERNAL_SERVICE_TOKEN 32
  require_secret NOTIFICATION_SERVICE_TOKEN 32
  require_secret OPS_INTERNAL_SERVICE_TOKEN 32
  require CORS_ALLOWED_ORIGINS
  [ "$CORS_ALLOWED_ORIGINS" != "*" ] || fail "CORS_ALLOWED_ORIGINS must not be wildcard in $APP_ENV"
  require_url PUBLIC_WEB_BASE_URL
  require_url NEXT_PUBLIC_AUTH_SERVICE_URL
  require_url NEXT_PUBLIC_USER_SERVICE_URL
  require_url NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL
  require_url NEXT_PUBLIC_TRIP_SERVICE_URL
  require_url NEXT_PUBLIC_NOTIFICATION_SERVICE_URL
fi

case "${RABBITMQ_URL:-}" in
  amqp://*|amqps://*) ;;
  "") fail "RABBITMQ_URL is required" ;;
  *) fail "RABBITMQ_URL must be an amqp/amqps URL" ;;
esac

if [ "${PLACE_PROVIDER:-mock}" = "foursquare" ]; then require FOURSQUARE_API_KEY; fi
if [ "${ROUTE_PROVIDER:-mock}" = "ors" ]; then require ORS_API_KEY; fi
if [ "${WEATHER_PROVIDER:-mock}" = "openweathermap" ]; then require OPENWEATHER_API_KEY; fi
case "${EXCHANGE_RATE_PROVIDER:-mock}" in
  mock) ;;
  *) require EXCHANGE_RATE_API_KEY ;;
esac
if [ "${GOOGLE_CALENDAR_ENABLED:-false}" = "true" ] && [ "${CALENDAR_PROVIDER:-mock}" = "google" ]; then
  require GOOGLE_OAUTH_CLIENT_ID
  require GOOGLE_OAUTH_CLIENT_SECRET
  require_url GOOGLE_OAUTH_REDIRECT_URL
  require CALENDAR_TOKEN_ENCRYPTION_KEY
fi
if [ "${WEB_PUSH_ENABLED:-false}" = "true" ]; then
  require WEB_PUSH_VAPID_PUBLIC_KEY
  require WEB_PUSH_VAPID_PRIVATE_KEY
  require WEB_PUSH_SUBJECT
fi
if [ "${EMAIL_PROVIDER:-mock}" = "smtp" ]; then
  require SMTP_HOST
  require SMTP_FROM_EMAIL
  if [ "$STRICT" = true ]; then require SMTP_PASSWORD; fi
fi
if [ "${OPS_DASHBOARD_ENABLED:-false}" = "true" ] && [ "$STRICT" = true ]; then
  require OPS_ADMIN_EMAILS
fi

echo "Environment file is valid for APP_ENV=${APP_ENV}"
