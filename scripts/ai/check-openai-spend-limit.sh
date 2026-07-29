#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ROOT_DIR}/infra/.env"

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
    --help|-h)
      echo "Usage: scripts/ai/check-openai-spend-limit.sh [--env-file PATH]"
      exit 0
      ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [[ -f "${ENV_FILE}" ]]; then
  load_env_file "${ENV_FILE}"
fi

if [[ "${AI_MODEL_PROVIDER:-mock}" != "openai" && "${AI_ITINERARY_GENERATOR_MODE:-mock}" != "openai" && "${ITINERARY_GENERATOR_MODE:-mock}" != "openai" ]]; then
  echo "OpenAI is not selected; spend limit check skipped."
  exit 0
fi

if [[ -z "${OPENAI_DAILY_SPEND_LIMIT_UAH:-}" && -z "${OPENAI_MONTHLY_SPEND_LIMIT_UAH:-}" ]]; then
  echo "OpenAI selected but no UAH spend limits are configured." >&2
  exit 1
fi

echo "OpenAI spend limit configuration is present. Durable enforcement still depends on the provider usage ledger."
