#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEB_DIR="$ROOT_DIR/apps/web"
SPECTRAL="$WEB_DIR/node_modules/.bin/spectral"

if [[ ! -x "$SPECTRAL" ]]; then
  echo "OpenAPI lint tooling is missing. Run npm ci in apps/web first." >&2
  exit 1
fi

shopt -s nullglob
SPECS=("$ROOT_DIR"/docs/api/openapi/*.yaml "$ROOT_DIR"/docs/api/openapi/*.yml "$ROOT_DIR"/docs/api/openapi/*.json)
if [[ ${#SPECS[@]} -eq 0 ]]; then
  echo "No OpenAPI specifications found in docs/api/openapi." >&2
  exit 1
fi

"$SPECTRAL" lint --ruleset "$ROOT_DIR/.spectral.yaml" "${SPECS[@]}"
