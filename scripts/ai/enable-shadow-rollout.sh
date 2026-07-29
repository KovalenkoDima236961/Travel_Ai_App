#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/ai/enable-shadow-rollout.sh --deployment-id UUID --sample-percent N --reason TEXT [--production]

Targets local/staging by default. Production requires --production and
CONFIRM_PRODUCTION_AI_ROLLOUT=1.
USAGE
}

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
DEPLOYMENT_ID=""
SAMPLE_PERCENT=""
REASON=""
PRODUCTION=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --deployment-id) DEPLOYMENT_ID="${2:-}"; shift 2 ;;
    --sample-percent) SAMPLE_PERCENT="${2:-}"; shift 2 ;;
    --reason) REASON="${2:-}"; shift 2 ;;
    --production) PRODUCTION=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "${DEPLOYMENT_ID}" || -z "${SAMPLE_PERCENT}" || -z "${REASON}" ]]; then
  usage >&2
  exit 2
fi
if [[ "${PRODUCTION}" == true && "${CONFIRM_PRODUCTION_AI_ROLLOUT:-}" != "1" ]]; then
  echo "production shadow rollout requires CONFIRM_PRODUCTION_AI_ROLLOUT=1" >&2
  exit 2
fi

args=(-fsS -H "Content-Type: application/json")
if [[ -n "${OPS_AUTH_TOKEN:-}" ]]; then
  args+=(-H "Authorization: Bearer ${OPS_AUTH_TOKEN}")
fi

reason_json=$(printf '%s' "${REASON}" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')
payload=$(printf '{"shadowSamplePercent":%s,"reason":%s}' "${SAMPLE_PERCENT}" "${reason_json}")
curl "${args[@]}" -X POST -d "${payload}" "${TRIP_SERVICE_URL%/}/ops/ai/model-deployments/${DEPLOYMENT_ID}/enable-shadow"
