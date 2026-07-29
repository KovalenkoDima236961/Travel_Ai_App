#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ROOT_DIR}/infra/.env"

usage() {
  cat <<'USAGE'
Usage: scripts/ai/check-openai-config.sh [--env-file PATH]

Loads an env file if present, then validates OpenAI provider configuration
without printing secrets.
USAGE
}

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      ""|\#*) continue ;;
      *=*) export "${line}" ;;
    esac
  done < "$1"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file)
      [[ $# -ge 2 ]] || { echo "--env-file requires a path" >&2; exit 2; }
      ENV_FILE="$2"
      shift 2
      ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -f "${ENV_FILE}" ]]; then
  load_env_file "${ENV_FILE}"
  "${ROOT_DIR}/scripts/validate-env.sh" --env-file "${ENV_FILE}"
else
  echo "Env file not found at ${ENV_FILE}; checking exported environment only."
fi

if [[ "${AI_MODEL_PROVIDER:-mock}" == "openai" || "${AI_ITINERARY_GENERATOR_MODE:-mock}" == "openai" || "${ITINERARY_GENERATOR_MODE:-mock}" == "openai" ]]; then
  [[ "${OPENAI_ENABLED:-false}" == "true" ]] || { echo "OPENAI_ENABLED must be true when OpenAI is selected." >&2; exit 1; }
  [[ -n "${OPENAI_API_KEY:-}" ]] || { echo "OPENAI_API_KEY is required when OpenAI is selected." >&2; exit 1; }
  [[ -n "${OPENAI_MODEL_DEFAULT:-}${OPENAI_MODEL_ITINERARY:-}" ]] || { echo "OPENAI_MODEL_DEFAULT or OPENAI_MODEL_ITINERARY is required." >&2; exit 1; }
  [[ "${OPENAI_STORE_RESPONSES+x}" == x ]] || { echo "OPENAI_STORE_RESPONSES must be explicit." >&2; exit 1; }
fi

echo "OpenAI configuration check passed."
