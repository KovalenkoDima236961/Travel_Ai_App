#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
ENV_EXAMPLE="${PROJECT_ROOT}/infra/.env.example"

cd "${PROJECT_ROOT}"

COMPOSE=(docker compose -f infra/docker-compose.yml --env-file infra/.env)

if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${ENV_EXAMPLE}" "${ENV_FILE}"
  echo "Created infra/.env from infra/.env.example"
fi

env_value() {
  local key="$1"
  local default_value="$2"
  local value
  value="$(
    grep -E "^${key}=" "${ENV_FILE}" | tail -n 1 | cut -d '=' -f 2- || true
  )"
  if [[ -z "${value}" ]]; then
    printf '%s' "${default_value}"
  else
    printf '%s' "${value}"
  fi
}

OLLAMA_MODEL="$(env_value OLLAMA_MODEL "llama3.1:8b")"
OLLAMA_EMBEDDING_MODEL="$(env_value OLLAMA_EMBEDDING_MODEL "nomic-embed-text")"
RAG_ENABLED="$(env_value RAG_ENABLED "true")"
RAG_ENABLED_NORMALIZED="$(printf '%s' "${RAG_ENABLED}" | tr '[:upper:]' '[:lower:]')"

echo "Starting PostgreSQL and Ollama..."
"${COMPOSE[@]}" up -d postgres ollama

echo "Pulling Ollama itinerary model: ${OLLAMA_MODEL}"
"${COMPOSE[@]}" exec -T ollama ollama pull "${OLLAMA_MODEL}"

echo "Pulling Ollama embedding model: ${OLLAMA_EMBEDDING_MODEL}"
"${COMPOSE[@]}" exec -T ollama ollama pull "${OLLAMA_EMBEDDING_MODEL}"

echo "Building service images..."
"${COMPOSE[@]}" build

echo "Trip Service migrations are applied automatically on service startup."

case "${RAG_ENABLED_NORMALIZED}" in
  1|true|yes|on)
    echo "Indexing local AI Planning Service knowledge files..."
    "${COMPOSE[@]}" run --rm ai-planning-service python -m app.scripts.index_knowledge
    ;;
  *)
    echo "Skipping knowledge indexing because RAG_ENABLED=${RAG_ENABLED}"
    ;;
esac

echo "Starting full backend stack..."
"${COMPOSE[@]}" up
