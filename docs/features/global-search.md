# Global Search and Command Palette

Global Search is the authenticated `Cmd+K` / `Ctrl+K` navigation layer for
private trip data. The browser mounts `GlobalCommandPalette` only for
authenticated, non-public-share routes. Mobile users also get a visible floating
search action from the same component.

## Backend Contract

Trip Service owns the frontend-facing endpoint:

`GET /search?q=&scope=all&types=trip,itinerary_item&limit=20&tripId=&workspaceId=&includeArchived=false&includeCommands=false`

The response keeps the existing `items`, `groups`, and `hasMore` fields and also
exposes `data`, `pagination`, and `queryMeta` aliases for generated clients.
Every result includes a safe `href`, type, title, optional description/context,
score, matched fields, source service, and bounded metadata. Raw database rows
are never returned.

## Search Scope

Current v1 backend results cover:

- Trips, excluding archived trips unless `includeArchived=true`.
- Route stops, route legs, selected transport options, and itinerary items from
  bounded trip JSON payloads.
- Expenses, receipt metadata, checklist items, reminders, polls, collaborators,
  templates, and workspace names.
- Optional safe navigation commands when `includeCommands=true`.

Notifications remain handled by the notification UI/API rather than aggregated
into Trip Service search in this v1 implementation.

## Authorization And Privacy

Trip Service filters all data server-side using trip owner, accepted
collaborator, and accessible workspace predicates before returning results.
Pending/removed collaborators do not satisfy the access predicate. Public share
pages do not mount the private palette or call `/search`.

Search excludes raw OCR text, receipt bytes, private files, comments, prompts,
provider secrets, public share credentials, and arbitrary attachment contents.
The endpoint rejects control characters, caps normalized query length at 200
runes by default, escapes SQL `LIKE` wildcards, and rate-limits authenticated
users through `GLOBAL_SEARCH_REQUESTS_PER_MINUTE`.

## PostgreSQL Strategy

V1 uses PostgreSQL directly. Migration `000035` enables `pg_trgm` and adds
trigram indexes for the normalized tables where search is common. JSON-backed
route and itinerary data is scanned only from a bounded set of already
permission-filtered trips, ordered by recent update. A separate projection table
is not required for the current data volume and result scope; add one only if
query plans show the bounded JSON scan becoming a bottleneck.

## Frontend Behavior

The frontend debounces server queries, cancels stale work through TanStack
Query, mixes backend results with local current-trip route/itinerary matches and
safe command shortcuts, and stores recent selections in user-scoped
localStorage. When the browser is offline, it searches the existing user-scoped
IndexedDB cache for saved trips, route/itinerary items, checklist items,
reminders, and expenses, and labels those results as offline. Recent query text
is not sent to the server. Logout clears the user-scoped recent item cache.
