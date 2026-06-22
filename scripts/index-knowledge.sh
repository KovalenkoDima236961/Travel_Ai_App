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

EMBEDDING_MODEL="$(env_value OLLAMA_EMBEDDING_MODEL "nomic-embed-text")"

echo "Ensuring Ollama is running..."
"${COMPOSE[@]}" up -d ollama

if ! "${COMPOSE[@]}" exec -T ollama ollama list | grep -q "^${EMBEDDING_MODEL}[[:space:]]"; then
  echo "Embedding model ${EMBEDDING_MODEL} is not present. Pulling it now..."
  "${COMPOSE[@]}" exec -T ollama ollama pull "${EMBEDDING_MODEL}"
fi

echo "Indexing local knowledge into ChromaDB..."
"${COMPOSE[@]}" run --rm ai-planning-service python -m app.scripts.index_knowledge
