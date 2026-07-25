#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVICE_DIR="${ROOT_DIR}/services/ai-planning-service"

usage() {
  echo "Usage: scripts/ai/validate-training-run.sh --config <config.json> [--dataset-path <export-dir>] [--dataset-version <version>] [--method lora|qlora]" >&2
}

if [[ $# -eq 0 ]]; then
  usage
  exit 2
fi

export PYTHONPATH="${SERVICE_DIR}${PYTHONPATH:+:${PYTHONPATH}}"
cd "${ROOT_DIR}"
exec python3 -m training.cli --dry-run "$@"
