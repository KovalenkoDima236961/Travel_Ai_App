#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEB_DIR="$ROOT_DIR/apps/web"
VALIDATOR="$WEB_DIR/scripts/validate-openapi.mjs"

if [[ ! -f "$VALIDATOR" ]]; then
  echo "OpenAPI validation tooling is missing. Restore apps/web/scripts/validate-openapi.mjs." >&2
  exit 1
fi

shopt -s nullglob
SPECS=("$ROOT_DIR"/docs/api/openapi/*.yaml "$ROOT_DIR"/docs/api/openapi/*.yml "$ROOT_DIR"/docs/api/openapi/*.json)
if [[ ${#SPECS[@]} -eq 0 ]]; then
  echo "No OpenAPI specifications found in docs/api/openapi." >&2
  exit 1
fi

node "$VALIDATOR" "${SPECS[@]}"
