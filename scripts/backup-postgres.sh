#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/backup-postgres.sh <output.dump>

Creates a custom-format pg_dump. Uses DATABASE_URL when set, otherwise PG*
or POSTGRES_* environment variables. The database password is never printed.
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

OUTPUT="${1:-}"
[ -n "$OUTPUT" ] || { usage >&2; exit 1; }
mkdir -p "$(dirname "$OUTPUT")"

if [ -n "${DATABASE_URL:-}" ]; then
  pg_dump "$DATABASE_URL" --format=custom --file="$OUTPUT"
else
  export PGHOST="${PGHOST:-${POSTGRES_HOST:-localhost}}"
  export PGPORT="${PGPORT:-${POSTGRES_PORT:-5432}}"
  export PGDATABASE="${PGDATABASE:-${POSTGRES_DB:-trip_service}}"
  export PGUSER="${PGUSER:-${POSTGRES_USER:-postgres}}"
  export PGPASSWORD="${PGPASSWORD:-${POSTGRES_PASSWORD:-}}"
  [ -n "$PGPASSWORD" ] || { echo "POSTGRES_PASSWORD or PGPASSWORD is required" >&2; exit 1; }
  pg_dump --format=custom --file="$OUTPUT"
fi

echo "Postgres backup written to $OUTPUT"
