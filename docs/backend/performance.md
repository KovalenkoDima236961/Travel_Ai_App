# Backend performance

Backend Performance Optimization v1 keeps the existing PostgreSQL and
RabbitMQ architecture. It concentrates on bounded reads, batch database
access, short-lived private summary caches, and observable fail-soft behavior.

## Targets

These are production-like p95 targets rather than local-build gates:

| Endpoint | Target |
| --- | ---: |
| `GET /trips` first page | < 300 ms |
| `GET /trips/{id}` compact detail | < 400 ms |
| Command Center warm / cold | < 700 ms / < 1200 ms |
| Trip Health warm / cold | < 500 ms / < 1200 ms |
| Verification warm / cold | < 600 ms / < 1500 ms |
| Budget summary | < 500 ms |
| Search (bounded) | < 300 ms |
| Library first page | < 500 ms |
| Unread count and generation-job status | < 150 ms |

## Hot-path rules

- Page-load `GET` endpoints use persisted metadata only. Verification does not
  invoke weather, places, transport, availability, or price providers; refresh
  actions are explicit writes.
- List responses remain compact and capped. Expenses, receipts, activity,
  notifications, jobs, search, and library results must retain their limits or
  cursor/offset pagination.
- Repository reads that fan out by ID must be batched. Health and Budget
  Confidence retrieve latest OCR records in one `DISTINCT ON` query; Group
  Readiness retrieves poll votes for all visible polls in one query.
- Never put raw IDs, search terms, request bodies, OCR, or AI prompts in metric
  labels or slow logs.

## Summary cache

Trip Service uses a process-local, size-bounded cache. Keys include the viewer,
effective role/options, trip ID, itinerary revision, and trip update time, so a
permission-filtered response cannot cross users. TTL is a fallback for
related-table writes that do not change the trip revision.

| Summary | TTL |
| --- | ---: |
| Command Center, Health, Budget Confidence, Group Readiness, Travel Day | 30 s default |
| Verification | 60 s |
| Library Insights | 300 s |

`archive` and `restore` clear Library Insights entries. Other mutation paths
are revision-aware when they update the trip; otherwise the short TTL applies.

Relevant environment variables:

```text
SUMMARY_CACHE_ENABLED=true
SUMMARY_CACHE_TTL_SECONDS=30
SUMMARY_CACHE_MAX_ITEMS=1000
SUMMARY_CACHE_LIBRARY_INSIGHTS_TTL_SECONDS=300
SUMMARY_ENDPOINT_TIMEOUT_SECONDS=8
COMMAND_CENTER_SECTION_TIMEOUT_MS=300
COMMAND_CENTER_PARALLEL_ENABLED=true
DB_QUERY_TIMEOUT_SECONDS=10
DB_SLOW_QUERY_THRESHOLD_MS=250
```

The Command Center runs independent optional sections in parallel by default.
Each receives its own deadline and reports a sanitized `sectionErrors` entry
instead of failing the core trip summary.

## Database and metrics

Trip Service exposes normalized HTTP route metrics, database operation latency,
connection-pool gauges, cache hit/miss/eviction counters, cold summary compute
duration (`trip_summary_compute_duration_seconds`), job queue metrics, and
provider timing/fallback metrics. Existing migrations add targeted indexes for
active expense/receipt feeds, latest OCR, checklists, reminders, notification
unread feeds, generation traces, archival lists, and trigram search.

Use the local helpers against seeded development data:

```bash
PERF_ACCESS_TOKEN=... PERF_TRIP_ID=... ./scripts/backend-perf-smoke.sh
TRIP_DB_URL=... PERF_TRIP_ID=... ./scripts/db-explain-common.sh
```

`backend-perf-smoke.sh` rejects 5xx responses and only fails latency when its
configurable `PERF_SMOKE_P95_THRESHOLD_MS` is exceeded. The existing
`scripts/performance-smoke-test.sh` additionally checks the compact Command
Center response for forbidden private fields.
