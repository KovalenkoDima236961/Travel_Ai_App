#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
"${ROOT_DIR}/scripts/ai/validate-knowledge.sh"
exec python3 "${ROOT_DIR}/services/ai-planning-service/scripts/run_evals.py" "$@"
