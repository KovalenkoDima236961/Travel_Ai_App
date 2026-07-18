#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
services=(auth-service user-service trip-service notification-service external-integrations-service worker-service)
flags=(-count=1)
export GOCACHE="${GOCACHE:-${PROJECT_ROOT}/.cache/go-build-tests}"
mkdir -p "${GOCACHE}"

if [[ "${GO_TEST_RACE:-false}" == "true" ]]; then
  flags+=(-race)
fi

for service in "${services[@]}"; do
  echo "==> go test services/${service}"
  (cd "${PROJECT_ROOT}/services/${service}" && go test ./... "${flags[@]}")
done

echo "PASS all Go service tests"
