#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
RESTORE_TEST=false
CONFIRMED=false

usage() {
  cat <<'USAGE'
Usage: scripts/verify-backup.sh BACKUP_FILE_OR_DIRECTORY [--restore-test --yes] [--env-file PATH]

Verifies that backups are readable. --restore-test additionally restores a
custom-format dump into a temporary local database and removes it afterwards.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --restore-test) RESTORE_TEST=true; shift ;;
    --yes) CONFIRMED=true; shift ;;
    --env-file) [[ $# -ge 2 ]] || { usage; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) [[ -z "${INPUT:-}" ]] || { usage >&2; exit 2; }; INPUT="$1"; shift ;;
  esac
done
[[ -n "${INPUT:-}" && -e "${INPUT}" ]] || { usage >&2; exit 1; }
command -v pg_restore >/dev/null 2>&1 || { echo "pg_restore is required." >&2; exit 1; }

verify_file() {
  local file="$1"
  case "${file}" in
    *.dump) pg_restore --list "${file}" >/dev/null ;;
    *.sql.gz) gzip -t "${file}" ;;
    *.sql) test -r "${file}" ;;
    *) echo "Unsupported backup format: ${file}" >&2; return 1 ;;
  esac
  echo "PASS $(basename "${file}")"
}

if [[ -d "${INPUT}" ]]; then
  shopt -s nullglob
  files=("${INPUT}"/*.dump "${INPUT}"/*.sql.gz "${INPUT}"/*.sql)
  [[ ${#files[@]} -gt 0 ]] || { echo "No supported backups found in ${INPUT}" >&2; exit 1; }
else
  files=("${INPUT}")
fi
for file in "${files[@]}"; do verify_file "${file}"; done

if [[ "${RESTORE_TEST}" == false ]]; then
  echo "Backup readability verification passed."
  exit 0
fi
[[ "${CONFIRMED}" == true ]] || { echo "--restore-test requires --yes." >&2; exit 1; }
[[ ${#files[@]} -eq 1 && "${files[0]}" == *.dump ]] || { echo "Restore testing currently supports one custom .dump file." >&2; exit 1; }
[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v createdb >/dev/null 2>&1 && command -v dropdb >/dev/null 2>&1 && command -v psql >/dev/null 2>&1 || { echo "createdb, dropdb, and psql are required for --restore-test." >&2; exit 1; }

while IFS= read -r line || [[ -n "${line}" ]]; do
  case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
done < "${ENV_FILE}"
case "${APP_ENV:-local}" in local|development|test) ;; *) echo "Restore testing is limited to local environments." >&2; exit 1 ;; esac
export PGHOST="${POSTGRES_HOST:-localhost}" PGPORT="${POSTGRES_PORT:-5432}" PGUSER="${POSTGRES_USER:-postgres}" PGPASSWORD="${POSTGRES_PASSWORD:-}"
TEMP_DB="travel_ai_backup_verify_${RANDOM}_${RANDOM}"
cleanup() { dropdb --if-exists "${TEMP_DB}" >/dev/null 2>&1 || true; }
trap cleanup EXIT
createdb "${TEMP_DB}"
pg_restore --no-owner --dbname="${TEMP_DB}" "${files[0]}"
psql --set ON_ERROR_STOP=1 --dbname="${TEMP_DB}" --command "SELECT 1" >/dev/null
echo "Backup restore test passed in temporary database."
