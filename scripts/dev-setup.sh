#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
ENV_EXAMPLE="${PROJECT_ROOT}/infra/.env.example"
START=true
BUILD=false
WITH_AI=false
WITH_RAG=false
WITH_OBSERVABILITY=false
PULL_MODELS=true

usage() {
  cat <<'USAGE'
Usage: scripts/dev-setup.sh [options]

Bootstraps the mock-first local stack. It creates infra/.env when absent,
validates configuration, applies migrations once, starts the selected profiles,
and waits for readiness.

Options:
  --build             Rebuild images before starting.
  --ai                Start Ollama and AI Planning Service.
  --rag               Enable the RAG profile (also enables the AI profile).
  --observability     Start Prometheus and Grafana.
  --no-model-pull     Do not pull Ollama models when --ai/--rag is selected.
  --prepare-only      Validate and prepare infra/.env without starting services.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --build) BUILD=true; shift ;;
    --ai) WITH_AI=true; shift ;;
    --rag) WITH_RAG=true; WITH_AI=true; shift ;;
    --observability) WITH_OBSERVABILITY=true; shift ;;
    --no-model-pull) PULL_MODELS=false; shift ;;
    --prepare-only) START=false; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

cd "${PROJECT_ROOT}"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required. Install Docker Desktop, start it, then retry." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 is required (expected: docker compose)." >&2
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${ENV_EXAMPLE}" "${ENV_FILE}"
  echo "Created infra/.env from infra/.env.example. Local development defaults are safe only for local use."
fi

"${SCRIPT_DIR}/validate-env.sh" local --env-file "${ENV_FILE}"
"${SCRIPT_DIR}/compose-validate.sh" --env-file "${ENV_FILE}"

if [[ "${START}" == false ]]; then
  echo "Environment prepared. Start with: docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core up -d"
  exit 0
fi

COMPOSE=(docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core)
[[ "${WITH_AI}" == true ]] && COMPOSE+=(--profile ai)
[[ "${WITH_RAG}" == true ]] && COMPOSE+=(--profile rag)
[[ "${WITH_OBSERVABILITY}" == true ]] && COMPOSE+=(--profile observability)
UP_ARGS=(up -d)
[[ "${BUILD}" == true ]] && UP_ARGS+=(--build)

echo "Starting PostgreSQL for the one-shot migration runner..."
"${COMPOSE[@]}" up -d postgres

echo "Applying service migrations..."
"${COMPOSE[@]}" --profile dev-tools run --rm migration-runner

if [[ "${WITH_AI}" == true && "${PULL_MODELS}" == true ]]; then
  env_value() {
    local key="$1" default_value="$2" value
    value="$(grep -E "^${key}=" infra/.env | tail -n 1 | cut -d '=' -f 2- || true)"
    printf '%s' "${value:-${default_value}}"
  }
  OLLAMA_MODEL="$(env_value OLLAMA_MODEL llama3.1:8b)"
  OLLAMA_EMBEDDING_MODEL="$(env_value OLLAMA_EMBEDDING_MODEL nomic-embed-text)"
  "${COMPOSE[@]}" up -d ollama
  echo "Pulling Ollama models required by the selected AI profile..."
  "${COMPOSE[@]}" exec -T ollama ollama pull "${OLLAMA_MODEL}"
  if [[ "${WITH_RAG}" == true ]]; then
    "${COMPOSE[@]}" exec -T ollama ollama pull "${OLLAMA_EMBEDDING_MODEL}"
  fi
fi

echo "Starting selected application profiles..."
"${COMPOSE[@]}" "${UP_ARGS[@]}"

if [[ "${WITH_AI}" == true ]]; then
  "${SCRIPT_DIR}/wait-for-ready.sh" core ai
else
  "${SCRIPT_DIR}/wait-for-ready.sh" core
fi

echo
echo "Local stack is ready:"
echo "  Web app:             http://localhost:3000"
echo "  Trip Service ready:  http://localhost:8080/ready"
echo "  RabbitMQ management: http://localhost:15672"
if [[ "${WITH_OBSERVABILITY}" == true ]]; then
  echo "  Prometheus:          http://localhost:9090"
  echo "  Grafana:             http://localhost:3030"
fi
echo "Run ./scripts/smoke-test.sh --core to verify the core flow."
