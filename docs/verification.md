# Real-World Travel Data Verification v1

Trip verification is a private, advisory view of how current trip assumptions
are. It combines persisted provider results, manual trip data, calendar-sync
state, and explicitly marked estimates. It is not a booking, purchase, or
availability guarantee.

## API and access

- `GET /trips/{tripId}/verification` is available to the owner, accepted
  collaborators, and permitted workspace members with normal private trip
  access.
- `POST /trips/{tripId}/verification/actions` accepts an explicit refresh or
  review action. Owner/editor access is required for actions that contact an
  existing provider integration or persist refreshed data.
- Public shares never expose verification details, provider metadata, calendar
  state, receipts, or refresh endpoints.

The response contains a 0–100 readiness score, a readiness level, scoped
details, status counts, top issues, and safe deep links. It intentionally
omits provider credentials, calendar event details, booking links, raw
responses, and user profile data.

## Status and source semantics

`verified` means a relevant provider-backed or confirmed record is current
according to its scope. `estimated` identifies AI, mock, fallback, heuristic,
or unverified data. `stale`, `missing`, `unavailable`, `failed`, and
`needs_review` identify distinct follow-up cases. `not_applicable` does not
reduce the score.

Sources are explicit: `provider`, `manual`, `receipt`, `calendar_sync`, `ai`,
`mock`, `fallback`, `heuristic`, `imported`, and `unknown`. Mock and fallback
data are never presented as verified.

## Freshness policy

The default stale windows are weather 12 hours near the trip / 24 hours
otherwise, transport 7 days, availability 48 hours, prices 7 days, place
details 30 days, route estimates 14 days, and calendar sync 7 days. These are
configuration defaults, not a background polling schedule: provider calls only
occur through existing user-initiated product flows or a user-requested refresh.

## Configuration and observability

`VERIFICATION_ENABLED`, `VERIFICATION_CACHE_ENABLED`, and
`VERIFICATION_CACHE_TTL_SECONDS` control the feature and its bounded in-process
cache. The remaining `VERIFICATION_*_STALE_*`, `VERIFICATION_NEAR_TRIP_DAYS`,
`VERIFICATION_MAX_DETAILS`, and `VERIFICATION_PLACE_MIN_CONFIDENCE` variables
set evaluator policy.

Prometheus metrics use the `trip_verification_` prefix for reads, duration,
score, observed statuses, stale items, and explicit actions. Labels exclude
trip IDs, users, item names, raw provider payloads, and credentials.
