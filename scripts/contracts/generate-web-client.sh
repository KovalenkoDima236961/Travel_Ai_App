#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEB_DIR="$ROOT_DIR/apps/web"
GENERATOR="$WEB_DIR/node_modules/.bin/openapi-typescript"
OUT_DIR="$WEB_DIR/src/lib/api/generated"

if [[ ! -x "$GENERATOR" ]]; then
  echo "OpenAPI generator is missing. Run npm ci in apps/web first." >&2
  exit 1
fi

for service in auth user trips notifications external-integrations ai-planning; do
  case "$service" in
    auth) spec_name="auth-service.openapi.yaml" ;;
    user) spec_name="user-service.openapi.yaml" ;;
    trips) spec_name="trip-service.openapi.yaml" ;;
    notifications) spec_name="notification-service.openapi.yaml" ;;
    external-integrations) spec_name="external-integrations-service.openapi.yaml" ;;
    ai-planning) spec_name="ai-planning-service.openapi.yaml" ;;
  esac
  spec="$ROOT_DIR/docs/api/openapi/$spec_name"
  output="$OUT_DIR/$service/schema.ts"
  mkdir -p "$(dirname "$output")"
  "$GENERATOR" "$spec" --output "$output"
done
