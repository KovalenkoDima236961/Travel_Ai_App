#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
CONFIRMED=false

usage() {
  cat <<'USAGE'
Usage: scripts/restore-postgres.sh BACKUP_FILE_OR_DIRECTORY --yes [--env-file PATH]

Restores into a local/development database only. --yes is mandatory because
existing tables are replaced. A backup directory created by backup-postgres.sh
restores every database it contains.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes) CONFIRMED=true; shift ;;
    --env-file) [[ $# -ge 2 ]] || { usage; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) [[ -z "${INPUT:-}" ]] || { usage >&2; exit 2; }; INPUT="$1"; shift ;;
  esac
done

[[ -n "${INPUT:-}" && -e "${INPUT}" ]] || { usage >&2; exit 1; }
[[ "${CONFIRMED}" == true ]] || { echo "Refusing restore without --yes." >&2; exit 1; }
[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v psql >/dev/null 2>&1 && command -v pg_restore >/dev/null 2>&1 || { echo "psql and pg_restore are required." >&2; exit 1; }

while IFS= read -r line || [[ -n "${line}" ]]; do
  case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
done < "${ENV_FILE}"
case "${APP_ENV:-local}" in local|development|test) ;; *) echo "Restore is limited to local/development/test environments." >&2; exit 1 ;; esac
export PGHOST="${POSTGRES_HOST:-localhost}" PGPORT="${POSTGRES_PORT:-5432}" PGUSER="${POSTGRES_USER:-postgres}" PGPASSWORD="${POSTGRES_PASSWORD:-}"
[[ -n "${PGPASSWORD}" ]] || { echo "POSTGRES_PASSWORD is required." >&2; exit 1; }

restore_file() {
  local file="$1" database="$2"
  echo "Restoring ${database} from $(basename "${file}")..."
  case "${file}" in
    *.dump) pg_restore --clean --if-exists --no-owner --dbname="${database}" "${file}" ;;
    *.sql.gz) gzip -dc "${file}" | psql --set ON_ERROR_STOP=1 --dbname="${database}" ;;
    *.sql) psql --set ON_ERROR_STOP=1 --dbname="${database}" --file="${file}" ;;
    *) echo "Unsupported backup format: ${file}" >&2; return 1 ;;
  esac
  psql --set ON_ERROR_STOP=1 --dbname="${database}" --command "SELECT 1" >/dev/null
}

if [[ -d "${INPUT}" ]]; then
  shopt -s nullglob
  files=("${INPUT}"/*.dump "${INPUT}"/*.sql.gz)
  [[ ${#files[@]} -gt 0 ]] || { echo "No supported backups found in ${INPUT}" >&2; exit 1; }
  for file in "${files[@]}"; do
    database="$(basename "${file}" | sed -E 's/\.(dump|sql\.gz)$//')"
    [[ "${database}" =~ ^[A-Za-z0-9_]+$ ]] || { echo "Unsafe database name inferred from ${file}" >&2; exit 1; }
    restore_file "${file}" "${database}"
  done
else
  restore_file "${INPUT}" "${POSTGRES_DB:-trip_service}"
fi

echo "Postgres restore completed successfully."
