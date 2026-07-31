# Trip Workspace consolidation

Trip Workspace organizes the existing private Trip Detail capabilities into six primary areas: **Overview**, **Plan**, **Money**, **Group**, **Prepare**, and **More**. It is an information-architecture layer over the existing Trip Service domains, not a new domain or data model.

## Repository audit

The pre-consolidation Trip Detail implementation was a single 3,700-line client page at `apps/web/src/_pages/trip-detail/ui/TripDetailPageContent.tsx`. It rendered a three-column anchored document and exposed more than twenty links in five sidebar groups. Mobile reused those links in an overflowing horizontal rail. Important audited foundations that are retained:

- one `tripKeys.detail(tripId)` query and user-scoped IndexedDB snapshot;
- `GET /trips/{tripId}/command-center-summary`, which already performs authorized, bounded, partial overview aggregation;
- deterministic selectors in `apps/web/src/lib/trip-command-center` for next action, attention/top fixes, and readiness;
- backend-provided `TripAccess` capabilities and server-side mutation authorization;
- the `?tab=` target map used by Global Search, notifications, health actions, email links, and bookmarks;
- stable entity DOM IDs, target focus/highlight, offline mutation queues, edit locks, version history, and public-share isolation;
- feature flags from Trip Service and four message catalogs (`en`, `es`, `uk`, `fr`).

The consolidation adds route-backed primary navigation, mobile section selection, a shared action registry, canonical deep links, archived/read-only status, and section-scoped loading. Existing panels remain their domain owners.

## Existing-to-new mapping

| Existing capability or tab | Workspace section | Canonical view |
| --- | --- | --- |
| Command Center, trip summary, Trip Health summary | Overview | `summary` |
| itinerary, agenda, timeline, calendar | Plan | `itinerary`, `agenda`, `timeline`, `calendar` |
| route, transport, map, accommodation, verification | Plan | `route`, `map`, `stay`, `verification` |
| budget, expenses, receipts, settlements, cost splits | Money | `budget`, `expenses`, `receipts`, `settlements`, `splits` |
| collaborators, comments, polls, availability, approvals, activity | Group | `people`, `discussion`, `decisions`, `availability`, `approvals`, `activity` |
| checklist, reminders, offline copy/sync | Prepare | `checklist`, `reminders`, `offline` |
| sharing, exports, versions, health detail, policy, recap, analytics, archive/restore | More | `sharing`, `exports`, `versions`, `health`, `policy`, `recap`, `tools` |

Only implemented capabilities appear as actions. Public share continues to use `/share/{shareToken}` and never mounts private workspace navigation.

## Behavior

- The Overview presents one deterministic next action, a bounded attention list, readiness states, recent activity, and four quick destinations.
- Viewers receive view/collaboration actions, not edit actions. Archived trips disable mutations and expose owner restore.
- Plan schedule view and selected day are represented in `view` and `day`; schedule view changes update the URL without losing the selected trip.
- Offline routes use cached identity/data and mark freshness. Unsupported online actions remain disabled.
- Heavy/detail queries are enabled for the selected workspace section or an explicit deep link, not for every primary section.
- Itinerary navigation warns before discarding an active unsaved edit and browser unload is protected.

## Rollout flags

All flags default on in local/test and off in production:

- `trip_workspace_consolidation_enabled`
- `trip_workspace_overview_v2_enabled`
- `trip_workspace_next_best_action_enabled`
- `trip_workspace_mobile_navigation_enabled`
- `trip_workspace_deep_link_v2_enabled`
- `trip_workspace_shared_actions_enabled`

The master flag restores the legacy anchored navigation. Additive section routes and the summary endpoint remain safe during rollback. Remove the old navigation only after search/notification/legacy-link, mobile, accessibility, and critical-journey checks remain green through general availability.

## Verification

Unit coverage is under `apps/web/tests/src/lib/trip-workspace`; component accessibility coverage is under `apps/web/tests/src/components/trip-workspace`; the critical Playwright journey verifies canonical navigation and a legacy expenses URL. See [navigation architecture](../architecture/trip-workspace-navigation.md) and [summary architecture](../architecture/trip-workspace-summary.md).
