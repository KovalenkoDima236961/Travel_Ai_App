#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

"$ROOT_DIR/scripts/contracts/generate-web-client.sh"

if ! git -C "$ROOT_DIR" diff --quiet -- apps/web/src/lib/api/generated || \
  [[ -n "$(git -C "$ROOT_DIR" ls-files --others --exclude-standard -- apps/web/src/lib/api/generated)" ]]; then
  echo "Generated API types are stale. Run ./scripts/contracts/generate-web-client.sh and commit the result." >&2
  git -C "$ROOT_DIR" diff -- apps/web/src/lib/api/generated >&2
  exit 1
fi
