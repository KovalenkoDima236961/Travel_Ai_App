#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PYTHONPATH="${ROOT_DIR}/services/ai-planning-service${PYTHONPATH:+:${PYTHONPATH}}"
cd "${ROOT_DIR}"
exec python3 -m training.metrics "$@"
