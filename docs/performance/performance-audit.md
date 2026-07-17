# Performance & Reliability Audit v1

Date: 2026-07-17. Scope: web, trip, worker, notification, external integrations, AI planning, and local observability. This is a code-path audit; latency targets must still be calibrated against staging data.

## Findings and status

| Priority | Location | Current problem and impact | Proposed fix | Implemented |
|---|---|---|---|---|
| High | `TripDetailPageContent.tsx` | The initial private trip render started roughly 20–25 reads, including health, readiness, money, decisions, collaboration, and activity. Slow dependencies delayed useful content and multiplied DB work. | Fetch trip plus one compact summary first; activate detailed queries when their section approaches the viewport or is deep-linked. | Yes |
| High | Trip Command Center | Cards were computed only after all detailed endpoint responses arrived. Any failed module removed most of the overview. | Add a viewer-scoped, fail-soft `GET /trips/{id}/command-center-summary` and retain the detailed-query builder as an error fallback. | Yes |
| High | Web query keys | Feature-owned key roots used incompatible layouts (`trip-health`, `group-readiness`, `expenses`), making invalidation broad or easy to miss. | Add one canonical factory under `src/lib/query-keys.ts`; move hot trip keys into the per-trip hierarchy. | Yes, hot paths |
| High | Expense mutations | Expense changes invalidated every expense query in the application and omitted Command Center/budget summary refreshes. | Invalidate only the affected trip’s expense subtree and its budget, health, activity, confidence, and summary dependents. | Yes |
| High | Checklist/reminder mutations | Summary/readiness dependencies could remain stale; reminder invalidation used the global reminder root. | Refresh trip-scoped health, group readiness, activity, and command summary. | Yes; legacy assigned-reminder invalidation remains global |
| High | Expense list service | Listing N expenses executed participant + receipt + OCR reads per expense/receipt. Latency grew linearly and put avoidable pressure on the pool. | Batch participants, receipts, and latest OCR results, then group in memory. | Yes |
| High | Expenses/receipts endpoints | Both lists were unbounded. A long-running trip could return arbitrarily large JSON and execute large joins/lookups. | Add default/max limits (expenses 50/100, receipts 30/100), offsets, `nextOffset`, and matching composite indexes. | Yes |
| High | Postgres calls | No server-side statement timeout or per-operation query latency/pool measurements existed. A stuck query could consume a pool slot indefinitely. | Configure `statement_timeout`, record low-cardinality duration/error/pool metrics, and log only sanitized slow-query operation metadata. | Yes |
| High | Summary calculations | Health, budget confidence, readiness, and Command Center repeatedly recomputed deterministic views during one browsing session. | Add bounded, 30-second in-process cache keyed by viewer, role, trip revision/update time, and options. | Yes |
| High | Provider fallbacks | Timeouts/fallbacks/metrics existed, but a failing live provider was retried on every request even while deterministic fallback was healthy. | Open a 30-second cooldown after three consecutive route/weather/place failures; expose open/short-circuit metrics. | Yes |
| High | Worker generation path | Reliability risk was assessed for duplicate delivery, terminal jobs, transient/permanent errors, max attempts, stale running jobs, DLQ, and shutdown. | Preserve the existing atomic claim/idempotency, classification, retry/DLQ, stale cleanup, and graceful shutdown implementation; document and smoke-test it. | Already present; documented in v1 |
| Medium | Trip detail JavaScript | Maps, route tools, receipts/expenses, health, readiness, checklist, and reminders were in the initial module graph. | Dynamically import heavy panels while preserving their existing skeleton and layout. | Yes |
| Medium | Notifications | SSE was supplemented by 45-second polling plus focus refetch, producing bursts when tabs regained focus. | Treat SSE as primary, reduce connected fallback polling to five minutes, disable focus refetch, and use 30-second freshness. | Yes |
| Medium | Ops AI traces | Backend cursor/limit support already exists; the UI polls every 20 seconds even when results are stable. | Retain cursor pagination, raise stale time, and pause polling when hidden in a follow-up. | Partial; backend bounded, UI polling unchanged |
| Medium | Activity | Cursor pagination and max 100 already existed; the overview issued a second five-item request. | Supply recent activity in compact summary; detailed rail remains lazy and cursor-paginated. | Yes |
| Medium | Generation jobs | Per-trip list defaults to 30 but the active-job status must remain fresh. | Keep the bounded list and poll only while an active job exists. | Already present |
| Medium | Receipt upload/download | Upload uses `MaxBytesReader`, streaming storage, 512-byte sniffing, MIME/size validation, and OCR timeout; download uses authorized streaming and no-store. | Retain streaming; batch list metadata/OCR and add explicit pagination. | Yes |
| Medium | Frontend vitals | No LCP/CLS/INP hook was installed. | Report normalized route-group vitals to an optional ingestion endpoint and emit a browser event for local tooling. | Yes |
| Medium | Database indexes | Single-column indexes did not cover newest-first active expense/receipt feeds, latest OCR, readiness filters, or combined ops filters. | Add partial/composite migrations based on actual filter and sort shapes. | Yes |
| Medium | Public share | Public page is already intentionally separate and does not mount private queries. | Keep private summary authenticated and deny public-share credentials. | Yes by existing private permission middleware/service access |
| Medium | AI Planning Service | HTTP timeouts exist at callers; local generation can still be CPU-bound and has no shared distributed cache. | Keep v1 incremental; profile prompts/validation with staging traces before changing algorithms. | No |
| Low | Large rendered lists | Expense/receipt responses are bounded, but UI virtualization/load-more is not consistently exposed for every panel. | Add explicit load-more controls or windowing after product interaction review. | No |
| Low | Distributed cache | A shared cache could reduce cross-instance recomputation but adds invalidation/privacy complexity. | Revisit only after short-TTL hit/miss metrics justify it. | No (intentionally out of scope) |

## Initial request budget

Before v1 the trip component directly activated trip detail, preferences, weather, budget summary/confidence, travelers/cost split, approval risk/state/policy, proposals, jobs, health, readiness, checklist, reminders, expenses/summary/settlements, availability, polls, activity, comments/reactions, and presence/edit-lock paths. Several child panels also had their own reads. The exact browser count varied with trip status and workspace access, but a completed collaborative workspace trip could exceed 25 requests plus streams.

After v1, the critical overview path is trip detail, compact Command Center summary, and bounded generation-job status. Detailed module reads begin from a deep link or as a stable section approaches the viewport. React Query coalesces identical in-flight keys, and dynamic imports keep inactive panel code out of the first page chunk.

## Reliability notes

- Summary section failures are returned in `sectionErrors`; no raw dependency errors, prompts, receipt bytes, OCR text, comments, or private notes are included.
- The cache includes viewer ID and effective role. It is bounded, expires lazily, and can be disabled.
- Query metrics label only SQL operation, never raw SQL, trip IDs, user IDs, or parameters.
- Existing jobs use atomic database claims, skip terminal rows, classify retryable failures, cap attempts, route exhausted messages to DLQ, mark stale running jobs, and stop consumers gracefully.
- Offset pagination was chosen for the existing expense/receipt UI contract. Activity, notifications, and AI traces retain keyset/cursor pagination.

## Follow-up measurements

Run the smoke script against seeded staging data and record p50/p95 for trip, summary, health, readiness, confidence, activity, and expense summary. Use the dashboard to compare DB p95 and summary cache hit ratio. The next optimization should be chosen from measured time, not request count alone.

