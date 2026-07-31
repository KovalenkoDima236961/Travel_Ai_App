# Trip Workspace summary architecture

## Decision

Trip Workspace reuses `GET /trips/{tripId}/command-center-summary` as its compact overview aggregation contract. Adding a second `/workspace-summary` endpoint would duplicate authorization, cache, partial-result, and observability logic without changing the domain. The current endpoint is the compatibility-equivalent of the proposed workspace summary.

## Request flow

1. Trip Service authenticates the user and calls `requireViewerEditorOrOwner`.
2. The cache key includes trip revision/state, user ID, and access role to prevent cross-user leakage.
3. Health, budget confidence, group readiness, verification, checklist, reminders, expenses, and activity are loaded with an endpoint timeout and per-section timeout; configured work may run in parallel.
4. Optional failures become low-cardinality `sectionErrors`; primary trip/access failures fail the request.
5. The frontend translates the compact summary into the existing `TripCommandCenterInput`, combines safe offline state, and runs deterministic selectors.

The endpoint never returns receipt files, comment text, checklist contents, collaborator names, or public-share-only data.

## Next best action

`apps/web/src/lib/trip-command-center/next-best-action.ts` ranks deterministic candidates: blocking health/policy, requested decisions, missing core plan, route/transport, budget, preparation, collaboration, post-trip settlement, and offline synchronization. Candidates declare `view`, `collaborate`, or `edit` capability. A viewer receives the first permitted action; an edit-only fallback is converted to view-only. No LLM participates in ranking.

## Attention and readiness

Trip Health top fixes provide the bounded attention list. Readiness cards compose existing authoritative health, route, budget-confidence, group-readiness, checklist/reminder, expense/settlement, approval/policy, activity, and offline responses. Labels are deterministic; a score is shown only when an existing domain already computes one.

## Query and invalidation

The summary key is `queryKeys.trip.commandCenter(tripId)`. Trip detail remains `tripKeys.detail(tripId)`. Mutations invalidate the affected domain plus the summary; they do not clear the application cache. The summary query is enabled for Overview, while explicit section deep links enable the required detailed queries.

## Partial behavior and operations

The frontend keeps available cards when `sectionErrors` is non-empty and exposes a small degraded state. Trip Service records `trip_workspace_summary_requests_total{status}`, `trip_workspace_summary_duration_seconds{status}`, and `trip_workspace_summary_partial_total{source}` alongside the existing cache and cold-compute metrics. See the [failure runbook](../operations/runbooks/trip-workspace-summary-failures.md).
