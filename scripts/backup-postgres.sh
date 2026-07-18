#!/usr/bin/env bash
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
BACKUP_ROOT="${BACKUP_DIR:-${PROJECT_ROOT}/backups}"
GZIP=false
declare -a DATABASES=()

usage() {
  cat <<'USAGE'
Usage: scripts/backup-postgres.sh [--output DIRECTORY] [--gzip] [DATABASE ...]
       scripts/backup-postgres.sh OUTPUT.dump

Creates timestamped backups for all service-owned databases by default. Custom
format dumps are suitable for pg_restore; --gzip creates portable .sql.gz
archives instead. The legacy single OUTPUT.dump form backs up the trip database.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output) [[ $# -ge 2 ]] || { echo "--output needs a directory" >&2; exit 2; }; BACKUP_ROOT="$2"; shift 2 ;;
    --gzip) GZIP=true; shift ;;
    --env-file) [[ $# -ge 2 ]] || { echo "--env-file needs a path" >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *.dump|*.sql.gz)
      [[ $# -eq 1 && ${#DATABASES[@]} -eq 0 ]] || { usage >&2; exit 2; }
      LEGACY_OUTPUT="$1"
      shift
      ;;
    *) DATABASES+=("$1"); shift ;;
  esac
done

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v pg_dump >/dev/null 2>&1 || { echo "pg_dump is required." >&2; exit 1; }

while IFS= read -r line || [[ -n "${line}" ]]; do
  case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
done < "${ENV_FILE}"
export PGHOST="${POSTGRES_HOST:-localhost}" PGPORT="${POSTGRES_PORT:-5432}" PGUSER="${POSTGRES_USER:-postgres}" PGPASSWORD="${POSTGRES_PASSWORD:-}"
[[ -n "${PGPASSWORD}" ]] || { echo "POSTGRES_PASSWORD is required." >&2; exit 1; }

if [[ -n "${LEGACY_OUTPUT:-}" ]]; then
  mkdir -p "$(dirname "${LEGACY_OUTPUT}")"
  pg_dump --format=custom --file="${LEGACY_OUTPUT}" "${POSTGRES_DB:-trip_service}"
  echo "Postgres backup written to ${LEGACY_OUTPUT} ($(du -h "${LEGACY_OUTPUT}" | awk '{print $1}'))."
  exit 0
fi

if [[ ${#DATABASES[@]} -eq 0 ]]; then
  DATABASES=(
    "${AUTH_POSTGRES_DB:-auth_service}"
    "${USER_POSTGRES_DB:-user_service}"
    "${POSTGRES_DB:-trip_service}"
    "${NOTIFICATION_POSTGRES_DB:-notification_service}"
    "${EXTERNAL_INTEGRATIONS_POSTGRES_DB:-external_integrations_service}"
  )
fi

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUTPUT_DIR="${BACKUP_ROOT%/}/postgres-${TIMESTAMP}"
mkdir -p "${OUTPUT_DIR}"

for database in "${DATABASES[@]}"; do
  [[ "${database}" =~ ^[A-Za-z0-9_]+$ ]] || { echo "Unsafe database name: ${database}" >&2; exit 1; }
  if [[ "${GZIP}" == true ]]; then
    pg_dump --format=plain "${database}" | gzip -c > "${OUTPUT_DIR}/${database}.sql.gz"
  else
    pg_dump --format=custom --file="${OUTPUT_DIR}/${database}.dump" "${database}"
  fi
done

(
  cd "${OUTPUT_DIR}"
  shopt -s nullglob
  shasum -a 256 ./*.dump ./*.sql.gz
) > "${OUTPUT_DIR}/SHA256SUMS"
echo "Postgres backups written to ${OUTPUT_DIR} ($(du -sh "${OUTPUT_DIR}" | awk '{print $1}'))."
