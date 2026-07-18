#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"

usage() {
  cat <<'USAGE'
Usage: scripts/compose-validate.sh [--env-file PATH]

Checks that Docker Compose is available, that the requested environment file
exists, and that infra/docker-compose.yml resolves successfully. Secret values
are never printed.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file)
      [[ $# -ge 2 ]] || { echo "--env-file needs a path" >&2; exit 2; }
      ENV_FILE="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required. Install Docker Desktop, then rerun this command." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 is required (expected: docker compose)." >&2
  exit 1
fi
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Environment file not found: ${ENV_FILE}" >&2
  echo "Create it with: cp infra/.env.example infra/.env" >&2
  exit 1
fi

cd "${PROJECT_ROOT}"
if ! docker compose -f infra/docker-compose.yml --env-file "${ENV_FILE}" config --quiet; then
  echo "Docker Compose configuration is invalid. Fix the reported error and retry." >&2
  exit 1
fi

echo "Compose configuration is valid: infra/docker-compose.yml"
