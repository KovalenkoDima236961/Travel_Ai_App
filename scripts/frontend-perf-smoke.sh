#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="${REPO_ROOT}/apps/web"

cd "${WEB_DIR}"
npm run typecheck
npm run build

if [[ "${FRONTEND_ANALYZE:-0}" == "1" ]]; then
  npm run analyze
fi

echo "Frontend performance smoke passed."
