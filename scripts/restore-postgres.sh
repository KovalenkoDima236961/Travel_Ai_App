#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/restore-postgres.sh <input.dump-or-sql>

Restores a Postgres backup. Refuses production restores unless
CONFIRM_PRODUCTION_RESTORE=restore is set.
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

INPUT="${1:-}"
[ -n "$INPUT" ] && [ -f "$INPUT" ] || { usage >&2; exit 1; }

if [ "${APP_ENV:-local}" = "production" ] && [ "${CONFIRM_PRODUCTION_RESTORE:-}" != "restore" ]; then
  echo "Refusing production restore. Set CONFIRM_PRODUCTION_RESTORE=restore to proceed." >&2
  exit 1
fi

if [ -n "${DATABASE_URL:-}" ]; then
  TARGET="$DATABASE_URL"
else
  export PGHOST="${PGHOST:-${POSTGRES_HOST:-localhost}}"
  export PGPORT="${PGPORT:-${POSTGRES_PORT:-5432}}"
  export PGDATABASE="${PGDATABASE:-${POSTGRES_DB:-trip_service}}"
  export PGUSER="${PGUSER:-${POSTGRES_USER:-postgres}}"
  export PGPASSWORD="${PGPASSWORD:-${POSTGRES_PASSWORD:-}}"
  [ -n "$PGPASSWORD" ] || { echo "POSTGRES_PASSWORD or PGPASSWORD is required" >&2; exit 1; }
  TARGET="$PGDATABASE"
fi

case "$INPUT" in
  *.sql)
    if [ -n "${DATABASE_URL:-}" ]; then
      psql "$TARGET" --file="$INPUT"
    else
      psql --file="$INPUT"
    fi
    ;;
  *)
    pg_restore --clean --if-exists --no-owner --dbname="$TARGET" "$INPUT"
    ;;
esac

echo "Postgres restore completed from $INPUT"
