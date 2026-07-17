# Performance & Reliability v1

The v1 strategy keeps the architecture unchanged: render a useful trip shell from a compact authenticated summary, progressively activate detailed modules, bound list work, and make failure visible without turning optional dependencies into page-wide failures.

## Frontend loading rules

- Use `apps/web/src/lib/query-keys.ts` for new queries. A private trip’s dependent data belongs under `['trips', 'detail', tripId, ...]`.
- Never enable a query until its identifier, permission context, network requirement, and visible/deep-linked section are available.
- Baseline stale times are 30 seconds for compact/list summaries, 45 seconds for expensive deterministic readiness calculations, and 10 minutes for weather. Active jobs are the exception: poll only while queued/running.
- Mutation invalidation is dependency-based and trip-scoped. Expense writes refresh expense, budget, confidence, health, activity, and Command Center data—not every expense list for every trip.
- Heavy route, map, health, readiness, checklist, reminder, and expense panels are dynamically imported. Existing skeleton dimensions are retained to avoid layout shift.
- `WebVitalsReporter` emits LCP/CLS/INP metrics with a normalized route group. Set `NEXT_PUBLIC_WEB_VITALS_ENDPOINT` only to an approved ingestion endpoint.

## Backend summary and cache

`GET /trips/{id}/command-center-summary` authenticates through the normal private-trip access path. It returns compact trip, route, health, budget, group, checklist, reminder, expense, and recent activity summaries. Independent sections run concurrently under `SUMMARY_ENDPOINT_TIMEOUT_SECONDS`; failures appear in `sectionErrors` and do not expose raw errors.

Health, budget confidence, group readiness, and Command Center summaries use a bounded in-memory cache. Keys contain viewer ID, role/options, trip ID, itinerary revision, and trip update time. Defaults:

```text
SUMMARY_CACHE_ENABLED=true
SUMMARY_CACHE_TTL_SECONDS=30
SUMMARY_CACHE_MAX_ITEMS=1000
SUMMARY_ENDPOINT_TIMEOUT_SECONDS=8
```

This is per-process and intentionally best-effort. Writes can remain visible with at most the configured TTL for related-table-only changes.

## Pagination and database safeguards

- Expenses: default 50, maximum 100, `offset`/`nextOffset`.
- Receipts: default 30, maximum 100, `offset`/`nextOffset`.
- Activity, notifications, and AI traces use their existing bounded cursor pagination.
- Migration `000034` adds active-feed, latest-OCR, readiness, and AI trace composite indexes. Notification migration `000004` adds unread/feed indexes.
- `DB_QUERY_TIMEOUT_SECONDS` sets PostgreSQL `statement_timeout`; `DB_SLOW_QUERY_THRESHOLD_MS` controls sanitized slow-operation logs.

## Jobs and providers

Generation workers already atomically claim jobs, ignore terminal duplicates, classify transient/permanent failures, cap retries, use retry/DLQ routing, clean stale running jobs, and stop gracefully. Keep side effects behind the claimed job transition.

Live route, weather, and place providers keep their configured HTTP timeouts and deterministic fallback. Three consecutive failures open a 30-second per-provider/operation cooldown, after which one primary probe is allowed. Provider metrics expose requests, duration, failures, fallback, cache, open circuit, and short circuits.

## Verification

```bash
./scripts/performance-smoke-test.sh
./scripts/worker-reliability-smoke-test.sh
```

The performance script requires an access token and trip ID, or login credentials. It prints mean/p95/status counts, rejects 5xx responses, checks the p95 threshold, and scans the compact summary for forbidden private fields. See `scripts/web-smoke-test.md` for the browser/network checklist and the Performance & Reliability Grafana dashboard for API, DB, cache, worker, and provider signals.

