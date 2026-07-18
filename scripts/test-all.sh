#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/test-frontend.sh"
"${SCRIPT_DIR}/test-go.sh"
"${SCRIPT_DIR}/test-python.sh"
"${SCRIPT_DIR}/test-stack-up.sh"

cleanup() {
  local status=$?
  trap - EXIT
  if [[ "${KEEP_TEST_STACK:-false}" != "true" ]]; then
    "${SCRIPT_DIR}/test-stack-down.sh" || true
  fi
  exit "${status}"
}
trap cleanup EXIT

TEST_STACK_MANAGED=false "${SCRIPT_DIR}/test-backend-integration.sh"
TEST_STACK_MANAGED=false "${SCRIPT_DIR}/test-frontend-e2e.sh"

echo "PASS complete test pyramid"
