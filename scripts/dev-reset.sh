#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
MAKE_BACKUP=false
RERUN_SETUP=true

usage() {
  cat <<'USAGE'
Usage: scripts/dev-reset.sh --yes [--backup] [--no-setup]

Stops the local Compose stack and removes its named volumes. This permanently
removes local Postgres, RabbitMQ, AI/RAG, export, and receipt data. It refuses
to run without --yes.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes) CONFIRMED=true; shift ;;
    --backup) MAKE_BACKUP=true; shift ;;
    --no-setup) RERUN_SETUP=false; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ "${CONFIRMED:-false}" == true ]] || {
  echo "Refusing destructive reset. Re-run with --yes after backing up anything important." >&2
  exit 1
}
[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }

cd "${PROJECT_ROOT}"
if [[ "${MAKE_BACKUP}" == true ]]; then
  "${SCRIPT_DIR}/backup-postgres.sh"
fi

echo "Removing local Compose containers and named volumes..."
docker compose -f infra/docker-compose.yml --env-file infra/.env --profile core --profile ai --profile rag --profile observability --profile dev-tools down --volumes --remove-orphans

if [[ "${RERUN_SETUP}" == true ]]; then
  "${SCRIPT_DIR}/dev-setup.sh"
fi
