# Trip Archive & Long-Term Library v1

Trip lifecycle is derived at read time: `draft`, `planning`, `ready`, `active`,
`completed`, and `archived`. Archive always wins. Active and completed derive
from the inclusive date range; ready requires an itinerary plus available health
and verification snapshots that meet the configured thresholds. Missing
snapshots do not block a trip and fall back to planning.

Archiving is a reversible organisation action, not deletion. It retains the
trip's itinerary, route, expenses, receipts, recap, templates, collaborators,
comments, activity, and existing share configuration. It does not automatically
disable an existing public share link. Archive/restore records an activity event
but sends no notification by default.

`GET /trips` excludes archived rows unless `includeArchived=true` is explicitly
sent. Private historical browsing uses `GET /trips/library`; it returns compact
cards only and never raw receipt OCR, private notes/comments, calendar details,
share tokens, provider metadata, or AI prompts. `GET /trips/library/insights`
uses deterministic counts and averages over trips the requesting user can view.

Archive and restore are limited to a personal trip owner, its workspace owner or
admin, or the recorded owner of a workspace trip. Public-share viewers never
receive these endpoints or the library. Library budgets are not converted across
currencies: mixed values are marked instead of inventing a total.

Configuration:

- `TRIP_LIBRARY_ENABLED` (default `true`)
- `TRIP_READY_HEALTH_SCORE_THRESHOLD` (default `80`)
- `TRIP_READY_VERIFICATION_SCORE_THRESHOLD` (default `75`)

Current v1 limits: there is no automatic archival, no permanent deletion flow,
no public travel profile, and no predictive or cross-user analytics.
