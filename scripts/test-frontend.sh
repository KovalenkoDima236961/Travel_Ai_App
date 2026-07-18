#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$(cd -- "${SCRIPT_DIR}/../apps/web" && pwd)"

cd "${WEB_DIR}"
npm run lint
npm run typecheck
if [[ "${CI:-false}" == "true" ]]; then
  npm run test:coverage
else
  npm test
fi

echo "PASS frontend lint, typecheck, and unit/component tests"
