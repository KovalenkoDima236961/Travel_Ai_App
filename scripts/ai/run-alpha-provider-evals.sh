#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODE="mock"
ALLOW_REAL_OPENAI="false"
MAX_REQUESTS="${OPENAI_EVAL_MAX_REQUESTS:-10}"

usage() {
  cat <<'USAGE'
Usage: scripts/ai/run-alpha-provider-evals.sh [--mock|--ollama|--openai] [--compare] [--allow-real-openai] [--max-requests N]

Mock mode is deterministic and CI-safe. OpenAI mode requires explicit
--allow-real-openai and validates configuration before any real-call workflow.
USAGE
}

COMPARE="false"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mock) MODE="mock"; shift ;;
    --ollama) MODE="ollama"; shift ;;
    --openai) MODE="openai"; shift ;;
    --compare) COMPARE="true"; shift ;;
    --allow-real-openai) ALLOW_REAL_OPENAI="true"; shift ;;
    --max-requests)
      [[ $# -ge 2 ]] || { echo "--max-requests requires a value" >&2; exit 2; }
      MAX_REQUESTS="$2"
      shift 2
      ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

case "${MAX_REQUESTS}" in
  ''|*[!0-9]*) echo "--max-requests must be a positive integer" >&2; exit 2 ;;
esac
if [[ "${MAX_REQUESTS}" -lt 1 || "${MAX_REQUESTS}" -gt 50 ]]; then
  echo "--max-requests must be between 1 and 50" >&2
  exit 2
fi

if [[ "${MODE}" == "openai" ]]; then
  if [[ "${ALLOW_REAL_OPENAI}" != "true" ]]; then
    echo "Refusing real OpenAI evaluation without --allow-real-openai." >&2
    exit 2
  fi
  "${ROOT_DIR}/scripts/ai/check-openai-config.sh"
  echo "OpenAI real-call evaluation harness is intentionally manual until the durable usage ledger and spend limiter are wired."
  echo "Validated config for up to ${MAX_REQUESTS} approved requests."
  exit 0
fi

if [[ "${COMPARE}" == "true" ]]; then
  echo "Running deterministic mock baseline."
  "${ROOT_DIR}/scripts/ai/run-itinerary-evals.sh"
  echo "Provider comparison reports should be stored under evals/alpha-openai/reports/."
  exit 0
fi

if [[ "${MODE}" == "ollama" ]]; then
  echo "Ollama alpha eval uses the existing eval runner after local AI stack startup."
  AI_EVAL_MODE=ollama "${ROOT_DIR}/scripts/ai/run-itinerary-evals.sh"
  exit 0
fi

"${ROOT_DIR}/scripts/ai/run-itinerary-evals.sh"

