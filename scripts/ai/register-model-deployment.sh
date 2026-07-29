#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/ai/register-model-deployment.sh --payload deployment.json [--production]

Environment:
  TRIP_SERVICE_URL  Default: http://localhost:8080
  OPS_AUTH_TOKEN    Optional bearer token. Never printed.

The payload must include deploymentKey, modelId, modelVariant, promptVersion,
and reason. Candidate deployments also require adapterId.

Production requires --production and CONFIRM_PRODUCTION_AI_ROLLOUT=1.
USAGE
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
PAYLOAD=""
PRODUCTION=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --payload) PAYLOAD="${2:-}"; shift 2 ;;
    --production) PRODUCTION=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "${PAYLOAD}" || ! -f "${PAYLOAD}" ]]; then
  echo "--payload must point to a deployment JSON file" >&2
  exit 2
fi
if [[ "${PRODUCTION}" == true && "${CONFIRM_PRODUCTION_AI_ROLLOUT:-}" != "1" ]]; then
  echo "production registration requires CONFIRM_PRODUCTION_AI_ROLLOUT=1" >&2
  exit 2
fi

args=(-fsS -H "Content-Type: application/json")
if [[ -n "${OPS_AUTH_TOKEN:-}" ]]; then
  args+=(-H "Authorization: Bearer ${OPS_AUTH_TOKEN}")
fi

cd "${ROOT_DIR}"
curl "${args[@]}" -X POST --data-binary "@${PAYLOAD}" "${TRIP_SERVICE_URL%/}/ops/ai/model-deployments"
