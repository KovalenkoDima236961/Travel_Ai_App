#!/usr/bin/env bash
set -euo pipefail

# Read-only local helper. TRIP_DB_URL must point at a development database.
# Do not use this script against production without an approved maintenance
# window because EXPLAIN ANALYZE executes the selected query.
: "${TRIP_DB_URL:?Set TRIP_DB_URL, for example postgres://postgres:postgres@localhost:5432/trip_service?sslmode=disable}"
: "${PERF_TRIP_ID:?Set PERF_TRIP_ID to a representative development trip UUID}"

command -v psql >/dev/null 2>&1 || {
  echo "psql is required." >&2
  exit 1
}

psql "${TRIP_DB_URL}" -v ON_ERROR_STOP=1 -v trip_id="${PERF_TRIP_ID}" <<'SQL'
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, expense_date, created_at
FROM trip_expenses
WHERE trip_id = :'trip_id'::uuid AND status = 'active'
ORDER BY expense_date DESC, created_at DESC, id DESC
LIMIT 50;

EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at
FROM trip_expense_receipts
WHERE trip_id = :'trip_id'::uuid AND deleted_at IS NULL AND status <> 'deleted'
ORDER BY created_at DESC, id DESC
LIMIT 30;

EXPLAIN (ANALYZE, BUFFERS)
SELECT DISTINCT ON (receipt_id) receipt_id, created_at
FROM receipt_ocr_results
WHERE trip_id = :'trip_id'::uuid
ORDER BY receipt_id, created_at DESC, id DESC;
SQL
