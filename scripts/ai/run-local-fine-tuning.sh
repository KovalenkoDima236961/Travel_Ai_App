#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVICE_DIR="${ROOT_DIR}/services/ai-planning-service"

if [[ "${AI_FINE_TUNING_EXPERIMENTS_ENABLED:-false}" != "true" ]]; then
  echo "Set AI_FINE_TUNING_EXPERIMENTS_ENABLED=true to run local fine-tuning." >&2
  exit 1
fi

export PYTHONPATH="${SERVICE_DIR}${PYTHONPATH:+:${PYTHONPATH}}"
cd "${ROOT_DIR}"
exec python3 -m training.cli "$@"
