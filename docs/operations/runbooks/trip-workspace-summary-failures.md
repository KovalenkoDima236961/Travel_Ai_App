# Trip Workspace summary failures

## Symptoms

- Overview shows a partial-information notice or missing cards.
- summary request latency/error or partial-source metrics rise;
- Next Best Action falls back to cached/detail-composed state;
- Trip Detail itself may still load and sections remain navigable.

## Triage

1. Correlate the browser request ID with Trip Service structured logs. Do not copy trip titles, destinations, comments, or expense text into incident notes.
2. Separate primary failures (authentication/access/trip load) from optional `sectionErrors`.
3. Check the named source’s latency/error metrics and Trip Service endpoint/section timeouts. Verify database/RabbitMQ/provider health only when that source depends on it.
4. Confirm the cache key includes the actor/access role and that no cross-user response is possible.
5. Reproduce with an owner and viewer fixture; confirm a viewer response contains no restricted financial/collaboration data.

## Mitigation

- For one optional source, keep the partial response and restore that dependency; do not increase timeouts before identifying saturation.
- If the consolidated experience is materially impaired, disable `trip_workspace_consolidation_enabled`. Legacy navigation returns immediately; canonical/legacy URLs and the additive summary endpoint remain valid.
- Disable a narrower overview/mobile/deep-link flag when the incident is isolated. Do not roll back with data deletion or a database migration.

## Recovery

Re-enable internal users first, then selected alpha users. Verify summary p50/p95, partial rate, permission denials, deep-link failures, mobile navigation, and the itinerary/expenses/collaboration/checklist/settings findability journeys. Record the failing source and removal criteria before closing the incident.
