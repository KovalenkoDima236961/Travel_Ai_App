#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/ai/run-shadow-evaluation-smoke.sh --deployment-id UUID --reason TEXT

Runs a local/staging smoke by enabling 100 percent shadow sampling for a
candidate deployment. This script refuses production.
USAGE
}

DEPLOYMENT_ID=""
REASON=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --deployment-id) DEPLOYMENT_ID="${2:-}"; shift 2 ;;
    --reason) REASON="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "${DEPLOYMENT_ID}" || -z "${REASON}" ]]; then
  usage >&2
  exit 2
fi
if [[ "${APP_ENV:-local}" == "production" ]]; then
  echo "shadow smoke is disabled in production" >&2
  exit 2
fi

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/enable-shadow-rollout.sh" \
  --deployment-id "${DEPLOYMENT_ID}" \
  --sample-percent 100 \
  --reason "${REASON}"
